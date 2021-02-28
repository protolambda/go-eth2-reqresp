package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon"
)

var StatusRPCv1 = reqresp.Method{
	Protocol:           "/eth2/beacon_chain/req/status/1/ssz_snappy",
	RequestCodec:       reqresp.NewSSZCodec(beacon.StatusByteLen, beacon.StatusByteLen),
	ResponseChunkCodec: reqresp.NewSSZCodec(beacon.StatusByteLen, beacon.StatusByteLen),
	Compression:        reqresp.SnappyCompression{},
}
