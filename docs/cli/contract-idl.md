# Experimental contract IDL command

`fairyc-contract-idl` validates one chain-neutral contract-interface file and writes either the
canonical JSON IDL or a typed client to standard output.

```text
go run ./cmd/fairyc-contract-idl --format json examples/contract_idl/token.fy
go run ./cmd/fairyc-contract-idl --format typescript examples/contract_idl/token.fy > token.ts
go run ./cmd/fairyc-contract-idl --format go --package tokenbindings examples/contract_idl/token.fy > token.go
```

The default format is `json`. `idl` aliases `json`, and `ts` aliases `typescript`. Go output uses
`contractbindings` unless `--package` selects another valid lowercase package name. The command
accepts one regular `.fy` file no larger than eight MiB and writes diagnostics as
`path:line:byte-column: code: message`.

The generated `ContractTransport` is intentionally codec-neutral. A consumer supplies the adapter
that chooses EVM ABI, BCS, Borsh, SCALE, or another chain encoding and owns address conversion,
signing, submission, confirmation, and response decoding. Every generated request carries the
`external_call` effect. `@payable` adds an explicit value argument; `@view` and `@payable` are
mutually exclusive.

Contract IDL v0 accepts `bool`, fixed-width signed and unsigned integers through `u256`, `String`,
`Bytes`, `Option<T>`, `Result<T, E>`, fixed arrays, event records, and opaque nominal paths. It rejects
safe references, raw pointers, slices, `usize`, `isize`, `str`, `c::*`, arbitrary generic types, zero
length arrays, duplicate declarations, and ambiguous attributes. `void` is accepted only as a direct
function result or the success payload of `Result`.

Exit status zero means successful emission. Status two identifies an invalid invocation or unreadable
input, status three identifies source diagnostics, and status four identifies an output or internal
failure.
