package reqresp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type payloadBuffer bytes.Buffer

func (p *payloadBuffer) Close() error {
	return nil
}

func (p *payloadBuffer) Write(b []byte) (n int, err error) {
	return (*bytes.Buffer)(p).Write(b)
}

func (p *payloadBuffer) OutputSizeVarint(w io.Writer) error {
	size := (*bytes.Buffer)(p).Len()
	sizeBytes := [binary.MaxVarintLen64]byte{}
	sizeByteLen := binary.PutUvarint(sizeBytes[:], uint64(size))
	_, err := w.Write(sizeBytes[:sizeByteLen])
	return err
}

func (p *payloadBuffer) WriteTo(w io.Writer) (n int64, err error) {
	return (*bytes.Buffer)(p).WriteTo(w)
}

type noCloseWriter struct {
	w io.Writer
}

func (nw *noCloseWriter) Write(p []byte) (n int, err error) {
	return nw.w.Write(p)
}

func (nw *noCloseWriter) Close() error {
	return nil
}

// StreamHeaderAndPayload reads a payload and streams (and optionally compresses) it to the writer.
// To do so, it requires the (uncompressed) payload length to be known in advance.
func StreamHeaderAndPayload(size uint64, r io.WriterTo, w io.Writer, comp Compression) error {
	sizeBytes := [binary.MaxVarintLen64]byte{}
	sizeByteLen := binary.PutUvarint(sizeBytes[:], size)
	n, err := w.Write(sizeBytes[:sizeByteLen])
	if err != nil {
		return fmt.Errorf("failed to write size bytes: %v", err)
	}
	if n != sizeByteLen {
		return fmt.Errorf("failed to write size bytes fully: %d/%d", n, sizeByteLen)
	}
	if comp != nil {
		compressedWriter := comp.Compress(&noCloseWriter{w: w})
		defer compressedWriter.Close()
		if _, err := r.WriteTo(compressedWriter); err != nil {
			return fmt.Errorf("failed to write payload through compressed writer: %v", err)
		}
		return nil
	} else {
		if _, err := r.WriteTo(w); err != nil {
			return fmt.Errorf("failed to write payload: %v", err)
		}
		return nil
	}
}

// EncodeResult writes the result code to the output writer.
func EncodeResult(result ResponseCode, w io.Writer) error {
	_, err := w.Write([]byte{uint8(result)})
	return err
}

// StreamChunk takes the (decompressed) response message from the msg io.WriterTo,
// and writes it as a chunk with given result code to the output writer. The compression is optional and may be nil.
func StreamChunk(result ResponseCode, size uint64, r io.WriterTo, w io.Writer, comp Compression) error {
	if err := EncodeResult(result, w); err != nil {
		return err
	}
	return StreamHeaderAndPayload(size, r, w, comp)
}
