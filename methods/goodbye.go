package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
)

var GoodbyeRPCv1 = reqresp.Method{
	Protocol:           "/eth2/beacon_chain/req/goodbye/1/ssz_snappy",
	RequestCodec:       reqresp.NewSSZCodec(8, 8),
	ResponseChunkCodec: reqresp.NewSSZCodec(8, 8),
	Compression:        reqresp.SnappyCompression{},
}
