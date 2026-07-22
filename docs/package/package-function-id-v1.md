# Package function identity v1

- Status: Historical; superseded by ADR 0174 FunctionID v2
- Decision: `docs/adr/0152-package-stable-function-identity.md`
- Implementation source: `internal/packageartifact/function_identity.go`

## Purpose

HIR `SymbolID`, semantic `PackageID`, MIR function order, and current `fy_fn_<index>` C names are
disposable build-local facts. A reusable package artifact cannot use any of them as its durable
cross-package call identity. `FunctionID` v1 supplies the missing package-stable key without making
it a source-language name or promising a native ABI.

## Descriptor

The descriptor contains the exact owning `PackageInstanceId`, the logical module-qualified function
name with its `$pN::` prefix removed, ordered parameter types, result type, canonical effects, and an
optional drop-glue resource type. Parameter names, source spans, HIR symbols, MIR block numbers,
function order, and bodies are excluded.

Every nominal type is rewritten from its disposable package number to the complete referenced
`PackageInstanceId`. Built-ins and recursive borrow, raw-pointer, fixed-array, `Option`, and `Result`
shapes receive deterministic compiler-private spellings. Missing package mappings, unknown types,
noncanonical effects, and an owner/name mismatch reject descriptor extraction.

## Encoding and text form

The byte stream begins with ASCII `fairyc.package-function`, a zero byte, and version byte `01`.
It then writes the 32 owner-package digest bytes followed by the logical name, parameter-type list,
result type, effect list, and drop-glue type. Strings and list counts use unsigned 64-bit network
byte order. SHA-256 hashes the complete stream and the only text form is:

```text
fyfn1-sha256-<64 lowercase hexadecimal digits>
```

Readers reject alternate prefixes, uppercase text, shortened values, and malformed hexadecimal.
Changing the descriptor or byte encoding requires a new version.

## Boundary

Equal IDs mean equal package identity and signature facts under this contract. They do not prove
equal bodies, verified MIR, object compatibility, or native calling convention. A package artifact
must still be keyed by source/interface/compiler/target compatibility, carry explicit imported
function descriptors, and pass standalone artifact verification before reuse. The C backend may
eventually derive stable private linker symbols from these IDs, but v1 does not expose them as ABI.
