package methods

import (
	"bytes"
	"context"
	"github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

var phase0Digest = common.ForkDigest{0xaa}
var altairDigest = common.ForkDigest{0xbb}

func TestBlocksByRangeRPCv2(t *testing.T) {

	spec := configs.Mainnet

	// bitvector must not be empty
	body := altair.BeaconBlockBody{SyncAggregate: altair.SyncAggregate{
		SyncCommitteeBits: make(altair.SyncCommitteeBits, spec.SYNC_COMMITTEE_SIZE/8)}}

	t.Run("single block", func(t *testing.T) {
		var mock mock.Mock
		req := &BlocksByRangeReqV1{StartSlot: 10, Count: 1, Step: 1}
		blocks := []common.SpecObj{&altair.SignedBeaconBlock{Message: altair.BeaconBlock{Slot: 10, Body: body}}}
		digests := []common.ForkDigest{altairDigest}
		mock.On("req", req).Return(blocks, digests)
		mock.On("readSuccess", uint64(0), blocks[0].ByteLength(spec), reqresp.SuccessCode, blocks[0])
		blocksByRangeExchange(t, &mock, spec, req)
	})

	t.Run("multiple blocks", func(t *testing.T) {
		var mock mock.Mock
		req := &BlocksByRangeReqV1{StartSlot: 10, Count: 3, Step: 1}

		blocks := []common.SpecObj{
			&phase0.SignedBeaconBlock{Message: phase0.BeaconBlock{Slot: 10}},
			&altair.SignedBeaconBlock{Message: altair.BeaconBlock{Slot: 11, Body: body}},
			&altair.SignedBeaconBlock{Message: altair.BeaconBlock{Slot: 12, Body: body}},
		}
		digests := []common.ForkDigest{phase0Digest, altairDigest, altairDigest}
		mock.On("req", req).Return(blocks, digests)
		mock.On("readSuccess", uint64(0), blocks[0].ByteLength(spec), reqresp.SuccessCode, blocks[0])
		mock.On("readSuccess", uint64(1), blocks[1].ByteLength(spec), reqresp.SuccessCode, blocks[1])
		mock.On("readSuccess", uint64(2), blocks[2].ByteLength(spec), reqresp.SuccessCode, blocks[2])

		blocksByRangeExchange(t, &mock, spec, req)
	})
}

func blocksByRangeExchange(t *testing.T, mock *mock.Mock, spec *common.Spec, realReq *BlocksByRangeReqV1) {
	phase0Typ := phase0.SignedBeaconBlockType(spec)
	altairTyp := altair.SignedBeaconBlockType(spec)
	method := BlocksByRangeRPCv2(spec, map[common.ForkDigest]reqresp.MinMaxSize{
		phase0Digest: {Min: phase0Typ.MinByteLength(), Max: altairTyp.MaxByteLength()},
		altairDigest: {Min: altairTyp.MinByteLength(), Max: altairTyp.MaxByteLength()},
	})

	assert := assert.New(t)

	mNet := mocknet.New()

	peerA, err := mNet.GenPeer()
	assert.NoError(err)
	peerB, err := mNet.GenPeer()
	assert.NoError(err)
	mNet.LinkPeers(peerA.ID(), peerB.ID())
	mNet.ConnectPeers(peerA.ID(), peerB.ID())

	h := method.MakeStreamHandler(func() context.Context {
		return context.Background()
	}, func(ctx context.Context, peerId peer.ID, handler reqresp.ChunkedRequestHandler) {
		var req BlocksByRangeReqV1
		if err := handler.ReadRequest(&req); err != nil {
			handler.WriteErrorChunk(reqresp.InvalidReqCode, "bad input")
			return
		}
		args := mock.MethodCalled("req", &req)
		for i := uint64(0); i < uint64(req.Count); i++ {
			block := args.Get(0).([]common.SpecObj)[i]
			digest := args.Get(1).([]common.ForkDigest)[i]
			err := handler.StreamSSZ(reqresp.SuccessCode, digest[:], spec.Wrap(block))
			assert.NoError(err)
		}
	})
	peerA.SetStreamHandler(method.Protocol, h)

	err = method.RunRequest(context.Background(), peerB.NewStream, peerA.ID(), realReq, 3, func(chunk reqresp.ChunkedResponseHandler) error {
		for i := uint64(0); i < uint64(realReq.Count); i++ {
			var block common.SpecObj
			err := chunk.ReadObj(func(contextBytes []byte) (dest codec.Deserializable, err error) {
				if bytes.Compare(contextBytes, phase0Digest[:]) == 0 {
					block = new(phase0.SignedBeaconBlock)
				} else if bytes.Compare(contextBytes, altairDigest[:]) == 0 {
					block = new(altair.SignedBeaconBlock)
				} else {
					mock.MethodCalled("bad", "unknown context bytes", contextBytes)
				}
				return spec.Wrap(block), nil
			})
			if err != nil {
				mock.MethodCalled("readFail", err)
				return err
			} else {
				mock.MethodCalled("readSuccess", chunk.ChunkIndex(), chunk.ChunkSize(), chunk.ResultCode(), block)
				return nil
			}
		}
		return nil
	})
	assert.NoError(err)

	mock.AssertExpectations(t)
}
