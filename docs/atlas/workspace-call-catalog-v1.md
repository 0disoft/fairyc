# Workspace call catalog v1

- Status: Implemented experimental output-only contract and explicit CLI entry point
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`
- Join prerequisite: `docs/atlas/workspace-call-join-v0.md`
- Implementation: `internal/workspacecallcatalog/catalog_generic.go`
- CLI: `fairyc experimental atlas-calls list --catalog-version 1`

`EncodeV1` emits one compact canonical JSON value without a trailing newline from an exact-current
v2 workspace call index. The legacy `Encode` accepts only v1 maps and retains catalog v0 bytes.
Neither encoder converts an index across map versions. There is no decoder or migration.

## Version and shape

```text
catalog version:    fairyc.workspace-call-catalog/v1
encoding:           fairyc.workspace-call-catalog-json/v1
checksum algorithm: fairyc.workspace-call-catalog-checksum/v1-sha256
checksum prefix:    fywcall1-sha256-
```

The envelope and catalog field order remain those of catalog v0. The selected package context
retains v2 map and generic source-interface identity and template-body completeness limitations.
Subject selection is still an exact package-qualified declaration SymbolId, never an implementation
selector. Each call endpoint extends the v0 shape with an optional final `specialization` field:

```json
{"package_instance":"...","symbol":"...","specialization":"g0_..."}
```

Ordinary endpoints omit that field. Generic implementation endpoints retain their exact current
compiler IDs alongside their original source declaration SymbolId. These IDs are not stable source
anchors, cross-build identity promises, caller annotations, or execution authority. The subject
never carries a specialization. No source body or documentation is emitted.

## Ordering and completeness

Calls sort by source package, source SymbolId, source implementation ID, target package, target
SymbolId, target implementation ID, span package, source path, start byte, and end byte. An absent
implementation ID sorts as an empty string. `both` removes only identical joined records; calls
from different implementations at the same source span remain distinct.

The v0 direction, whole-workspace package/call budget, selected-call usage, and atomic final byte
limit rules remain unchanged. `complete` means every selected compiler-observed resolved call in
the admitted workspace, not universal template semantics, runtime dispatch recovery, or a concrete
execution-path graph. Unused template bodies remain unavailable in the context ledger.

## Checksum

SHA-256 covers ASCII `fairyc.workspace-call-catalog-checksum`, bytes `00 01`, then the catalog
version, encoding, and canonical catalog JSON. Each of the last three byte strings is preceded by
its unsigned 64-bit big-endian byte length. `ParseChecksumV1` accepts only the v1 prefix and exactly
64 lowercase hexadecimal digits; `ChecksumV1.String` retains that prefix. V0 parsing rejects v1
checksums. The checksum detects byte changes, not authenticity or trusted compiler provenance.

## CLI selection

`atlas-calls list` defaults to catalog v0; explicit `--catalog-version 0` is identical. Selecting
`--catalog-version 1` uses the shared generic-aware uncached workspace compiler and exact-current
v2 map preparation, then the bounded v2 join and this encoder. Only canonical values `0` and `1`
are accepted, at most once. The existing package/symbol/direction/budget flags remain required.
`--locked` retains its exact no-write lock behavior; otherwise lock refresh may precede a later
selection or budget failure. No map, catalog, compiler cache, or artifact file is written.

Discover an original function's SymbolId using `atlas-symbols list --catalog-version 2` with the
desired exact package/module/name filters and explicit limits. Pass its visible or full ID to
`atlas-calls list --catalog-version 1 --symbol <id>` with the same workspace and all required call
limits. The two catalog versions belong to different output families, not competing map versions.

## Not enabled

Context Pack, Evidence Bundle, attributed text, persisted reverse-call discovery, implementation
selectors, and transitive execution traversal are not enabled by this command.
