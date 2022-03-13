package methods

import (
	"fmt"
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"io"
)

func NoContext(minMaxSize reqresp.MinMaxSize) reqresp.ReadContextFn {
	return func(blr *reqresp.BufLimitReader) (contextBytes []byte, minMax reqresp.MinMaxSize, err error) {
		return nil, minMaxSize, nil
	}
}

func BlocksContext(blocksMinMax map[common.ForkDigest]reqresp.MinMaxSize) reqresp.ReadContextFn {
	return func(blr *reqresp.BufLimitReader) (contextBytes []byte, minMax reqresp.MinMaxSize, err error) {
		blr.N = 4
		blr.PerRead = false
		var digest common.ForkDigest
		_, err = io.ReadFull(blr, digest[:])
		if err != nil {
			return nil, reqresp.MinMaxSize{}, err
		}
		blockMinMax, ok := blocksMinMax[digest]
		if !ok {
			return nil, reqresp.MinMaxSize{}, fmt.Errorf("unknown fork-digest: %s", digest)
		}
		return digest[:], blockMinMax, nil
	}
}
