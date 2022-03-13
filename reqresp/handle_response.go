package reqresp

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
)

// ResponseChunkHandler is a function that processes a response chunk. The index, size and result-code are already parsed.
// The response context-bytes are nil if the result is not SuccessCode.
// The contents (decompressed if previously compressed) can be read from r. Optionally an answer can be written back to w.
// If the response chunk could not be processed, an error may be returned.
type ResponseChunkHandler func(ctx context.Context, chunkIndex uint64, chunkSize uint64, result ResponseCode, contextBytes []byte, r io.Reader) error

// ResponseHandler processes a response by internally processing chunks, any error is propagated up.
type ResponseHandler func(ctx context.Context, r io.ReadCloser) error

// ReadContextFn reads <context-bytes> from the response stream, and determines the max chunk size for decoding.
// The blr.N and blr.PerRead fields should be customized to read the input.
type ReadContextFn func(blr *BufLimitReader) (contextBytes []byte, minMax MinMaxSize, err error)

// MakeResponseHandler builds a ResponseHandler, which won't take more than maxChunkCount chunks, or chunk contents larger than maxChunkContentSize.
// Compression is optional and may be nil. Chunks are processed by the given ResponseChunkHandler.
func (handleChunk ResponseChunkHandler) MakeResponseHandler(
	maxChunkCount uint64,
	readContext ReadContextFn,
	comp Compression) ResponseHandler {
	//		response  ::= <response_chunk>*
	//      response_chunk  ::= <result> | <context-bytes> | <encoding-dependent-header> | <encoded-payload>
	//		result    ::= “0” | “1” | “2” | [“128” ... ”255”]
	//
	// note: a phase0 type of req-resp method may simply read 0 <context-bytes>.
	return func(ctx context.Context, r io.ReadCloser) error {
		// Stop reading chunks as soon as we exit.
		defer r.Close()
		if maxChunkCount == 0 {
			return nil
		}
		blr := NewBufLimitReader(r, 1024, 0)
		for chunkIndex := uint64(0); chunkIndex < maxChunkCount; chunkIndex++ {
			blr.N = 1
			resByte, err := blr.ReadByte()
			if err == io.EOF { // no more chunks left.
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to read chunk %d result byte: %v", chunkIndex, err)
			}
			var contextBytes []byte
			var minMax MinMaxSize
			if ResponseCode(resByte) == SuccessCode {
				// read the <context-bytes>, if any.
				if readContext != nil {
					contextBytes, minMax, err = readContext(blr)
					if err != nil {
						return fmt.Errorf("failed to read context-bytes: %v", err)
					}
				}
			}
			// varints need to be read byte by byte.
			blr.N = 1
			blr.PerRead = true
			chunkSize, err := binary.ReadUvarint(blr)
			blr.PerRead = false
			// TODO when input is incorrect, return a different type of error.
			if err != nil {
				return fmt.Errorf("failed to read chunk size: %v", err)
			}
			chunkMax := chunkSize
			if ResponseCode(resByte) == SuccessCode {
				if chunkSize < minMax.Min {
					return fmt.Errorf("chunk size %d of chunk %d lower than chunk min %d", chunkSize, chunkIndex, minMax.Min)
				}
				if chunkSize > minMax.Max {
					return fmt.Errorf("chunk size %d of chunk %d higher than chunk max %d", chunkSize, chunkIndex, minMax.Max)
				}
			} else {
				if chunkSize > MAX_ERR_SIZE {
					return fmt.Errorf("chunk size %d of chunk %d exceeds error size limit %d", chunkSize, chunkIndex, MAX_ERR_SIZE)
				}
				chunkMax = MAX_ERR_SIZE
			}
			if comp != nil {
				chunkMax, err = comp.MaxEncodedLen(chunkMax)
				if err != nil {
					return fmt.Errorf("failed to compute max compressed length: %v", err)
				}
			}
			blr.N = int(chunkMax)
			cr := io.Reader(blr)
			if comp != nil {
				cr = comp.Decompress(cr)
			}
			if err := handleChunk(ctx, chunkIndex, chunkSize, ResponseCode(resByte), contextBytes, cr); err != nil {
				return err
			}
		}
		return nil
	}
}
