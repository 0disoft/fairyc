# Atlas Symbol Catalog v3

- Status: Implemented opt-in encoder; CLI integration pending
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Implementation: `internal/symbolcatalog/catalog_aggregate.go`

`EncodeV3` accepts only an exact-current v3 map query. Legacy `Encode` and
`EncodeV2` retain their exact map admission, catalog v1/v2 bytes and checksums.
No constructor converts map versions or accepts arbitrary declaration facts.

The catalog lists original declarations in full-SymbolId order, including used
and unused generic structs and resources. Concrete specializations and cleanup
bodies do not become additional symbols. Kind/name/module filters refer to the
original declaration. It is an identity list, not a field/signature/body dump.

The canonical envelope and entry layout, full-ID cursor, total count, selection
budget, final byte budget, compact JSON and empty-array normalization follow
`docs/atlas/symbol-catalog-v2.md`. The context identifies v3 maps and current
ordinary/generic source interfaces and retains template-body uncertainty.
No source text, documentation, host path, fuzzy selector, authority or cross-version
cursor migration is added. Output overflow returns no partial bytes.

## Integrity

```text
catalog version:    fairyc.symbol-catalog/v3
encoding:           fairyc.symbol-catalog-json/v3
checksum algorithm: fairyc.symbol-catalog-checksum/v3-sha256
checksum prefix:    fyscat3-sha256-
```

SHA-256 covers ASCII `fairyc.symbol-catalog-checksum`, bytes `00 02`, then the
catalog version, encoding and canonical catalog JSON. Each of the last three
strings is prefixed by its unsigned 64-bit big-endian byte length.
`ParseChecksumV3` accepts only the v3 prefix and 64 lowercase hexadecimal digits;
legacy parsers reject that prefix. This detects changed bytes, not authenticity.

Tests cover exact kind/name selection of four used/unused aggregate templates,
single-entry pagination without specialization duplicates, empty output, exact
final byte limits, independent checksum calculation and version rejection.
The returned original resource ID can select cleanup calls in call catalog v2
(`docs/atlas/workspace-call-catalog-v2.md`). CLI adoption remains separate work;
stable language, SDK and runtime ABI versions are unchanged.
