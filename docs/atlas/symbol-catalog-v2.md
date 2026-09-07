# Atlas Symbol Catalog v2

- Status: Implemented output-only exact generic declaration catalog
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`
- Implementation: `internal/symbolcatalog/catalog_generic.go`
- CLI: `fairyc experimental atlas-symbols list --catalog-version 2`

## Contract

`EncodeV2` accepts only an exact-current v2 map query. The existing `Encode` accepts only v1 maps
and retains symbol catalog v1. Neither entry point converts map schemas or silently reinterprets
versioned output. The CLI defaults to catalog version `1`; explicit `1` has identical behavior.
Explicit `2` uses the same generic-aware, uncached workspace compiler as stable workspace commands,
then prepares v2 snapshot, map, current-source expectation, and query objects from that workspace.
Existing lock and error/output policies remain unchanged.

The catalog still lists original source declarations in full-SymbolId order, not concrete compiler
implementations. Multiple specializations produce one catalog entry for their original declaration;
unused generic declarations remain discoverable. Generic arity and positional parameter bindings
participate in the original SymbolId contract, not a new mangled-name selector. The catalog is an
identity listing, not a signature or executable-fact dump. Its v2 compatibility context retains
generic source-interface identity and template-body completeness limitations.

Exact module/name/kind filters, full-ID cursors, total counts, selection-byte limits, final byte
limits, canonical envelope/record order, no-newline output, and empty-array normalization follow
symbol catalog v1. No source body, documentation, host path, fuzzy result, execution authority, or
cross-version cursor migration is added. Final output overflow returns no bytes.

## Integrity

```text
catalog version:    fairyc.symbol-catalog/v2
encoding:           fairyc.symbol-catalog-json/v2
checksum algorithm: fairyc.symbol-catalog-checksum/v2-sha256
checksum prefix:    fyscat2-sha256-
```

SHA-256 covers ASCII `fairyc.symbol-catalog-checksum`, bytes `00 01`, then the catalog version,
encoding, and canonical catalog JSON. Each of those last three strings is prefixed by its unsigned
64-bit big-endian byte length. `ParseChecksumV2` accepts only the v2 prefix and 64 lowercase hex
digits; `ChecksumV2.String` preserves that prefix. V1 checksum parsing rejects v2. Checksums detect
byte changes, not authenticity. No input decoder or persisted catalog format is admitted.

## Remaining integration

Pass an original declaration ID to `atlas-calls list --catalog-version 1` for generic-aware
workspace call queries, or `atlas-pack show|around --pack-version 2` for bounded compiler-fact
neighborhoods. The existing `atlas-text source|documentation` commands accept these original IDs
through generic-aware preparation while retaining attributed-text v0 output. Generic Evidence
Bundle v1 composition is available with `atlas-evidence show --bundle-version 1`.
