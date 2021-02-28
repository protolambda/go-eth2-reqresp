package reqresp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/protolambda/ztyp/codec"
	"io"
)

type Request interface {
	fmt.Stringer
}

type SSZCodec struct {
	minByteLen uint64
	maxByteLen uint64
}

func NewSSZCodec(minByteLen uint64, maxByteLen uint64) *SSZCodec {
	return &SSZCodec{
		minByteLen: minByteLen,
		maxByteLen: maxByteLen,
	}
}

func (c *SSZCodec) MinByteLen() uint64 {
	if c == nil {
		return 0
	}
	return c.minByteLen
}

func (c *SSZCodec) MaxByteLen() uint64 {
	if c == nil {
		return 0
	}
	return c.maxByteLen
}

func (c *SSZCodec) Encode(w io.Writer, input codec.Serializable) error {
	if c == nil {
		if input != nil {
			return errors.New("expected empty data, nil input. This codec is no-op")
		}
		return nil
	}
	return input.Serialize(codec.NewEncodingWriter(w))
}

func (c *SSZCodec) Decode(r io.Reader, bytesLen uint64, dest codec.Deserializable) error {
	if c == nil {
		if bytesLen != 0 {
			return errors.New("expected 0 bytes, no definition")
		}
		return nil
	}
	return dest.Deserialize(codec.NewDecodingReader(r, bytesLen))
}

type Method struct {
	// Protocol ID, e.g. /eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz_snappy
	Protocol           protocol.ID
	RequestCodec       *SSZCodec
	ResponseChunkCodec *SSZCodec
	// Compression to apply to requests and response chunks. Nil if no compression.
	Compression Compression
}

type ResponseCode uint8

const (
	SuccessCode    ResponseCode = 0
	InvalidReqCode              = 1
	ServerErrCode               = 2
)

// 256 bytes max error size
const MAX_ERR_SIZE = 256

type OnResponseListener func(chunk ChunkedResponseHandler) error

type ChunkedResponseHandler interface {
	ChunkSize() uint64
	ChunkIndex() uint64
	ResultCode() ResponseCode
	ReadRaw() ([]byte, error)
	ReadErrMsg() (string, error)
	ReadObj(dest codec.Deserializable) error
}

type chRespHandler struct {
	m          *Method
	r          io.Reader
	result     ResponseCode
	chunkSize  uint64
	chunkIndex uint64
}

func (c *chRespHandler) ChunkSize() uint64 {
	return c.chunkSize
}

func (c *chRespHandler) ChunkIndex() uint64 {
	return c.chunkIndex
}

func (c *chRespHandler) ResultCode() ResponseCode {
	return c.result
}

func (c *chRespHandler) ReadRaw() ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(io.LimitReader(c.r, int64(c.chunkSize)))
	return buf.Bytes(), err
}

func (c *chRespHandler) ReadErrMsg() (string, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(io.LimitReader(c.r, int64(c.chunkSize)))
	return string(buf.Bytes()), err
}

func (c *chRespHandler) ReadObj(dest codec.Deserializable) error {
	return c.m.ResponseChunkCodec.Decode(c.r, c.chunkSize, dest)
}

type writerToFn func(w io.Writer) (n int64, err error)

func (fn writerToFn) WriteTo(w io.Writer) (n int64, err error) {
	return fn(w)
}

func (m *Method) RunRequest(ctx context.Context, newStreamFn NewStreamFn,
	peerId peer.ID, req codec.Serializable, maxRespChunks uint64, onResponse OnResponseListener) error {

	handleChunks := ResponseChunkHandler(func(ctx context.Context, chunkIndex uint64, chunkSize uint64, result ResponseCode, r io.Reader) error {
		return onResponse(&chRespHandler{
			m:          m,
			r:          r,
			result:     result,
			chunkSize:  chunkSize,
			chunkIndex: chunkIndex,
		})
	})

	reqSize := req.ByteLength()
	reqTo := writerToFn(func(w io.Writer) (n int64, err error) {
		// pick a buffer size based on the
		size := 1024
		if size > int(reqSize) {
			size = int(reqSize)
		}
		bw := bufio.NewWriterSize(w, size)
		defer bw.Flush()
		return int64(reqSize), m.RequestCodec.Encode(bw, req)
	})

	protocolId := m.Protocol
	maxChunkContentSize := m.ResponseChunkCodec.MaxByteLen()
	if m.Compression != nil {
		if s, err := m.Compression.MaxEncodedLen(maxChunkContentSize); err != nil {
			return err
		} else {
			maxChunkContentSize = s
		}
	}

	respHandler := handleChunks.MakeResponseHandler(maxRespChunks, maxChunkContentSize, m.Compression)

	// Runs the request in sync, which processes responses,
	// and then finally closes the channel through the earlier deferred close.
	return newStreamFn.Request(ctx, peerId, protocolId, reqSize, reqTo, m.Compression, respHandler)
}

type ReadRequestFn func(dest interface{}) error
type WriteSuccessChunkFn func(data interface{}) error
type WriteMsgFn func(msg string) error

type RequestReader interface {
	// nil if not an invalid input
	InvalidInput() error
	ReadRequest(dest codec.Deserializable) error
	RawRequest() ([]byte, error)
}

type RequestResponder interface {
	StreamSSZ(code ResponseCode, data codec.Serializable) error
	WriteRawResponseChunk(code ResponseCode, chunk []byte) error
	StreamResponseChunk(code ResponseCode, size uint64, r io.WriterTo) error
	WriteErrorChunk(code ResponseCode, msg string) error
}

type ChunkedRequestHandler interface {
	RequestReader
	RequestResponder
}

type chReqHandler struct {
	m               *Method
	respBuf         bufio.Writer
	reqLen          uint64
	r               io.ReadCloser
	w               io.Writer
	invalidInputErr error
}

func (h *chReqHandler) InvalidInput() error {
	return h.invalidInputErr
}

func (h *chReqHandler) ReadRequest(dest codec.Deserializable) error {
	defer h.r.Close()
	if h.invalidInputErr != nil {
		return h.invalidInputErr
	}
	var r io.Reader = h.r
	if h.m.Compression != nil {
		r = h.m.Compression.Decompress(r)
	}
	return h.m.RequestCodec.Decode(r, h.reqLen, dest)
}

func (h *chReqHandler) RawRequest() ([]byte, error) {
	defer h.r.Close()
	if h.invalidInputErr != nil {
		return nil, h.invalidInputErr
	}
	var buf bytes.Buffer
	var r io.Reader = h.r
	if h.m.Compression != nil {
		r = h.m.Compression.Decompress(r)
	}
	if _, err := buf.ReadFrom(io.LimitReader(r, int64(h.reqLen))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *chReqHandler) StreamSSZ(code ResponseCode, data codec.Serializable) error {
	reqSize := data.ByteLength()
	reqTo := writerToFn(func(w io.Writer) (n int64, err error) {
		// re-use the same buffer to smooth write performance
		h.respBuf.Reset(w)
		defer h.respBuf.Flush()
		return int64(reqSize), h.m.RequestCodec.Encode(w, data)
	})
	return StreamChunk(code, reqSize, reqTo, h.w, h.m.Compression)
}

func (h *chReqHandler) WriteRawResponseChunk(code ResponseCode, chunk []byte) error {
	return StreamChunk(code, uint64(len(chunk)), bytes.NewReader(chunk), h.w, h.m.Compression)
}

func (h *chReqHandler) StreamResponseChunk(code ResponseCode, size uint64, r io.WriterTo) error {
	return StreamChunk(code, size, r, h.w, h.m.Compression)
}

func (h *chReqHandler) WriteErrorChunk(code ResponseCode, msg string) error {
	if len(msg) > MAX_ERR_SIZE {
		msg = msg[:MAX_ERR_SIZE-3]
		msg += "..."
	}
	b := []byte(msg)
	return StreamChunk(code, uint64(len(b)), bytes.NewReader(b), h.w, h.m.Compression)
}

type OnRequestListener func(ctx context.Context, peerId peer.ID, handler ChunkedRequestHandler)

func (m *Method) MakeStreamHandler(newCtx StreamCtxFn, listener OnRequestListener) network.StreamHandler {
	return RequestPayloadHandler(func(ctx context.Context, peerId peer.ID, requestLen uint64, r io.ReadCloser, w io.Writer, comp Compression, invalidInputErr error) {
		listener(ctx, peerId, &chReqHandler{
			m: m, respBuf: *bufio.NewWriterSize(w, 1024), reqLen: requestLen, r: r, w: w, invalidInputErr: invalidInputErr,
		})
	}).MakeStreamHandler(newCtx, m.Compression, m.RequestCodec.MinByteLen(), m.RequestCodec.MaxByteLen())
}
