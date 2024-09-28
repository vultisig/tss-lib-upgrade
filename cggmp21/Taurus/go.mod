module Taurus

go 1.21.5

require github.com/taurusgroup/multi-party-sig v0.6.0-alpha-2021-09-21

require (
	github.com/cronokirby/saferith v0.33.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zeebo/blake3 v0.2.3 // indirect
	golang.org/x/sys v0.9.0 // indirect
)

replace github.com/taurusgroup/multi-party-sig => ./Taurus_fork/multi-party-sig
