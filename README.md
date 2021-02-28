# `go-eth2-reqresp`

This package implements the [Eth2 Req-Resp protocol](https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/p2p-interface.md#the-reqresp-domain) in Go.

It builds on top of:
- [`github.com/protolambda/zrnt`](https://github.com/protolambda/zrnt/) for Eth2 types.
- [`github.com/protolambda/ztyp`](https://github.com/protolambda/ztyp/) for SSZ encoding.
- [`github.com/golang/snappy`](https://github.com/golang/snappy/) for Snappy compression.
- [`github.com/libp2p/go-libp2p-core`](https://github.com/libp2p/go-libp2p-core/) for Libp2p stream interface

It supports streaming of SSZ data: the output size can be determined quickly in advance, and the data can then be encoded as it is transferred.
This is done with both requests and responses. A bufio writer is used to avoid encoding overhead:
it uses a 1 KiB buffer currently, but will be tuned based on future performance testing. 

This package is based on an earlier Req-Resp implementation in [Rumor](https://github.com/protolambda/rumor),
but was refactored to improve streaming, improve the RPC method definitions, and to use the new LibP2P stream read/write-closer Go API.

Phase0 Req-resp method definitions can be found in the `methods` package.

## Usage

```go
import (
	"context"
	"fmt"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/protolambda/go-eth2-reqresp/methods"
	"github.com/protolambda/go-eth2-reqresp/reqresp"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/eth2/configs"
	"log"
)
```

Server
```go
	method := methods.BlocksByRangeRPCv1(spec)
	host.SetStreamHandler(method.Protocol,
		method.MakeStreamHandler(ctxFn, func(ctx context.Context, peerID peer.ID, handler reqresp.ChunkedRequestHandler) {
		var req methods.BlocksByRangeReqV1
		if err := handler.ReadRequest(&req); err != nil {
			log.Printf("failed to decode request by %s: %v", peerID, err)
			_ = handler.WriteErrorChunk(reqresp.InvalidReqCode, "cannot decode request")
		} else {
			log.Printf("processing request for start %d, count %d, step %d", req.StartSlot, req.Count, req.Step)
			var blocks []*beacon.SignedBeaconBlock = nil // TODO: source some response blocks based on the request.
			for i, bl := range blocks {
				if err := handler.StreamSSZ(reqresp.SuccessCode, spec.Wrap(bl)); err != nil {
					log.Printf("failed to write block %d: %v", i, err)
				}
			}
		}
	}))
```

Client
```go
	req := methods.BlocksByRangeReqV1{StartSlot: 123, Step: 1, Count: 3}
	peerID := peer.ID("16Uiu2HAkwFP2d1RbKgGcBtQrD1UKKD6R8DP7Z9Eh7veb1xjMzTYy")
	maxRespChunks := uint64(3)
	err := method.RunRequest(context.Background(), host.NewStream, peerID, &req,
		maxRespChunks, func(chunk reqresp.ChunkedResponseHandler) error {
			var block beacon.SignedBeaconBlock
			if err := chunk.ReadObj(spec.Wrap(&block)); err != nil {
				return fmt.Errorf("failed to decode block response chunk %d: %v", chunk.ChunkIndex(), err)
			}
			log.Printf("chunk %d matched block at slot %d", chunk.ChunkIndex(), block.Message.Slot)
			// or return an error to stop reading any later chunks. (E.g. don't read all chunks if the peer is giving us bad data)
			return nil
		})
	if err != nil {
		log.Println(err)
	}
```

Adding methods:
- Define a new `reqresp.Method` struct value, with full protocol name,
  SSZ size bounds (`reqresp.NewSSZCodec`) and compression (e.g. `nil` or `reqresp.SnappyCompression`)

## License

MIT. See [`LICENSE`](./LICENSE) file.
