package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon/common"
)

var MetaDataRPCv1 = reqresp.Method{
	Protocol:         "/eth2/beacon_chain/req/metadata/1/ssz_snappy",
	RequestMinMax:    reqresp.MinMaxSize{0, 0}, // no request data, just empty bytes.
	Compression:      reqresp.SnappyCompression{},
	ReadContextBytes: NoContext(reqresp.MinMaxSize{Min: common.MetadataByteLen, Max: common.MetadataByteLen}),
}
