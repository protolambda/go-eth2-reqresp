package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon/common"
)

var StatusRPCv1 = reqresp.Method{
	Protocol:         "/eth2/beacon_chain/req/status/1/ssz_snappy",
	RequestMinMax:    reqresp.MinMaxSize{Min: common.StatusByteLen, Max: common.StatusByteLen},
	Compression:      reqresp.SnappyCompression{},
	ReadContextBytes: NoContext(reqresp.MinMaxSize{Min: common.StatusByteLen, Max: common.StatusByteLen}),
}
