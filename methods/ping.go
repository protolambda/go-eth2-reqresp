package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
)

var PingRPCv1 = reqresp.Method{
	Protocol:         "/eth2/beacon_chain/req/ping/1/ssz_snappy",
	RequestMinMax:    reqresp.MinMaxSize{Min: 8, Max: 8},
	Compression:      reqresp.SnappyCompression{},
	ReadContextBytes: NoContext(reqresp.MinMaxSize{Min: 8, Max: 8}),
}
