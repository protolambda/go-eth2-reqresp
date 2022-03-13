package methods

import (
	"encoding/hex"
	"fmt"
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
)

type BlocksByRangeReqV1 struct {
	StartSlot common.Slot
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

func (d *BlocksByRangeReqV1) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(&d.StartSlot, &d.Count, &d.Step)
}

func (r *BlocksByRangeReqV1) String() string {
	return fmt.Sprintf("%v", *r)
}

func BlocksByRangeRPCv1(spec *common.Spec) *reqresp.Method {
	typ := phase0.SignedBeaconBlockType(spec)
	minMax := reqresp.MinMaxSize{Min: typ.MinByteLength(), Max: typ.MaxByteLength()}
	return &reqresp.Method{
		Protocol:         "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz_snappy",
		RequestMinMax:    reqresp.MinMaxSize{Min: blocksByRangeReqByteLen, Max: blocksByRangeReqByteLen},
		Compression:      reqresp.SnappyCompression{},
		ReadContextBytes: NoContext(minMax),
	}
}

func BlocksByRangeRPCv2(spec *common.Spec, blocksMinMax map[common.ForkDigest]reqresp.MinMaxSize) *reqresp.Method {
	return &reqresp.Method{
		Protocol:         "/eth2/beacon_chain/req/beacon_blocks_by_range/2/ssz_snappy",
		RequestMinMax:    reqresp.MinMaxSize{Min: blocksByRangeReqByteLen, Max: blocksByRangeReqByteLen},
		Compression:      reqresp.SnappyCompression{},
		ReadContextBytes: BlocksContext(blocksMinMax),
	}
}

const MAX_REQUEST_BLOCKS_BY_ROOT = 1024

type BlocksByRootReqV1 []common.Root

func (a *BlocksByRootReqV1) Deserialize(dr *codec.DecodingReader) error {
	return tree.ReadRootsLimited(dr, (*[]common.Root)(a), MAX_REQUEST_BLOCKS_BY_ROOT)
}

func (a BlocksByRootReqV1) Serialize(w *codec.EncodingWriter) error {
	return tree.WriteRoots(w, a)
}

func (a BlocksByRootReqV1) ByteLength() (out uint64) {
	return uint64(len(a)) * 32
}

func (a *BlocksByRootReqV1) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

func (r BlocksByRootReqV1) Data() []string {
	out := make([]string, len(r), len(r))
	for i := range r {
		out[i] = hex.EncodeToString(r[i][:])
	}
	return out
}

func (r BlocksByRootReqV1) String() string {
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

func BlocksByRootRPCv1(spec *common.Spec) *reqresp.Method {
	typ := phase0.SignedBeaconBlockType(spec)
	minMax := reqresp.MinMaxSize{Min: typ.MinByteLength(), Max: typ.MaxByteLength()}
	return &reqresp.Method{
		Protocol:         "/eth2/beacon_chain/req/beacon_blocks_by_root/1/ssz_snappy",
		RequestMinMax:    reqresp.MinMaxSize{0, 32 * MAX_REQUEST_BLOCKS_BY_ROOT},
		Compression:      reqresp.SnappyCompression{},
		ReadContextBytes: NoContext(minMax),
	}
}

func BlocksByRootRPCv2(spec *common.Spec, blocksMinMax map[common.ForkDigest]reqresp.MinMaxSize) *reqresp.Method {
	return &reqresp.Method{
		Protocol:         "/eth2/beacon_chain/req/beacon_blocks_by_root/2/ssz_snappy",
		RequestMinMax:    reqresp.MinMaxSize{0, 32 * MAX_REQUEST_BLOCKS_BY_ROOT},
		Compression:      reqresp.SnappyCompression{},
		ReadContextBytes: BlocksContext(blocksMinMax),
	}
}
