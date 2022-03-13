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

Phase0 and Altair Req-resp method definitions can be found in the `methods` package.

## License

MIT. See [`LICENSE`](./LICENSE) file.
