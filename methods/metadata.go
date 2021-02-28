package methods

import (
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon"
)

var MetaDataRPCv1 = reqresp.Method{
	Protocol:           "/eth2/beacon_chain/req/metadata/1/ssz_snappy",
	RequestCodec:       (*reqresp.SSZCodec)(nil), // no request data, just empty bytes.
	ResponseChunkCodec: reqresp.NewSSZCodec(beacon.MetadataByteLen, beacon.MetadataByteLen),
	Compression:        reqresp.SnappyCompression{},
}
