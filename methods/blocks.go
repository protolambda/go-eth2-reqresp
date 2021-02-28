package methods

import (
	"encoding/hex"
	"fmt"
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
)

type BlocksByRangeReqV1 struct {
	StartSlot beacon.Slot
	Count     view.Uint64View
	Step      view.Uint64View
}

func (d *BlocksByRangeReqV1) Deserialize(dr *codec.DecodingReader) error {
	return dr.FixedLenContainer(&d.StartSlot, &d.Count, &d.Step)
}

func (d *BlocksByRangeReqV1) Serialize(w *codec.EncodingWriter) error {
	return w.FixedLenContainer(&d.StartSlot, &d.Count, &d.Step)
}

const blocksByRangeReqByteLen = 8 + 8 + 8

func (d BlocksByRangeReqV1) ByteLength() uint64 {
	return blocksByRangeReqByteLen
}

func (*BlocksByRangeReqV1) FixedLength() uint64 {
	return blocksByRangeReqByteLen
}

func (d *BlocksByRangeReqV1) HashTreeRoot(hFn tree.HashFn) beacon.Root {
	return hFn.HashTreeRoot(&d.StartSlot, &d.Count, &d.Step)
}

func (r *BlocksByRangeReqV1) String() string {
	return fmt.Sprintf("%v", *r)
}

func BlocksByRangeRPCv1(spec *beacon.Spec) *reqresp.Method {
	return &reqresp.Method{
		Protocol:           "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz_snappy",
		RequestCodec:       reqresp.NewSSZCodec(blocksByRangeReqByteLen, blocksByRangeReqByteLen),
		ResponseChunkCodec: reqresp.NewSSZCodec(0, spec.SignedBeaconBlock().MaxByteLength()),
		Compression:        reqresp.SnappyCompression{},
	}
}

const MAX_REQUEST_BLOCKS_BY_ROOT = 1024

type BlocksByRootReq []beacon.Root

func (a *BlocksByRootReq) Deserialize(dr *codec.DecodingReader) error {
	return tree.ReadRootsLimited(dr, (*[]beacon.Root)(a), MAX_REQUEST_BLOCKS_BY_ROOT)
}

func (a BlocksByRootReq) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, a)
}

func (a BlocksByRootReq) ByteLength() (out uint64) {
	return uint64(len(a)) * 32
}

func (a *BlocksByRootReq) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

func (r BlocksByRootReq) Data() []string {
	out := make([]string, len(r), len(r))
	for i := range r {
		out[i] = hex.EncodeToString(r[i][:])
	}
	return out
}

func (r BlocksByRootReq) String() string {
	if len(r) == 0 {
		return "empty blocks-by-root request"
	}
	out := make([]byte, 0, len(r)*66)
	for i, root := range r {
		hex.Encode(out[i*66:], root[:])
		out[(i+1)*66-2] = ','
		out[(i+1)*66-1] = ' '
	}
	return "blocks-by-root requested: " + string(out[:len(out)-1])
}

func BlocksByRootRPCv1(spec *beacon.Spec) *reqresp.Method {
	return &reqresp.Method{
		Protocol:           "/eth2/beacon_chain/req/beacon_blocks_by_root/1/ssz_snappy",
		RequestCodec:       reqresp.NewSSZCodec(0, 32*MAX_REQUEST_BLOCKS_BY_ROOT),
		ResponseChunkCodec: reqresp.NewSSZCodec(0, spec.SignedBeaconBlock().MaxByteLength()),
		Compression:        reqresp.SnappyCompression{},
	}
}
