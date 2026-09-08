# Workspace call catalog v2

- Status: Implemented opt-in output-only encoder and experimental CLI
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Join: `NewWorkspaceCallIndexV3`, `docs/atlas/workspace-call-join-v0.md`
- Implementation: `internal/workspacecallcatalog/catalog_aggregate.go`
- CLI: `fairyc experimental atlas-calls list --catalog-version 2`

`EncodeV2` accepts only an exact-current v3 workspace call index. Legacy `Encode`
and `EncodeV1` retain their exact-version admission and v0/v1 JSON bytes. There
is no decoder, conversion, map discovery, source loading or output-file write.

## Format and ownership

```text
catalog version:    fairyc.workspace-call-catalog/v2
encoding:           fairyc.workspace-call-catalog-json/v2
checksum algorithm: fairyc.workspace-call-catalog-checksum/v2-sha256
checksum prefix:    fywcall2-sha256-
```

The v1 envelope and catalog layout remain unchanged. Each call may start with
`source_body: "cleanup"` before `source`. It is present only for calls observed in
a concrete resource cleanup. Source and target retain their package instance,
original declaration SymbolId and optional concrete specialization ID. Ordinary
function calls omit `source_body`. The subject remains a declaration selector,
never a specialization or cleanup execution selector.

Ordering follows v1 with source body inserted after source SymbolId and before
source specialization. An absent body sorts as an empty string. Direction `both`
deduplicates only identical joined records; distinct cleanup implementations at
the same source span remain separate. All canonical JSON is compact with no
trailing newline; empty calls encode as `[]`.

V3 package context retains current source and interface identity and unavailable
universal template-body facts. `complete` means the selected explicit resolved
calls in the admitted workspace, not implicit drop scheduling, runtime execution
counts, or transitive execution-path completeness. No source text or documentation
is emitted; `source_body` is a compiler-owned kind marker, not source content.

## Budgets and checksum

Whole-workspace package/call admission budgets remain those of the join. The
encoder applies its byte limit to the complete envelope including the body marker
and checksum. Exceeding that limit returns no partial bytes.

SHA-256 covers ASCII `fairyc.workspace-call-catalog-checksum`, bytes `00 02`, then
the catalog version, encoding and canonical catalog JSON, each prefixed by its
unsigned 64-bit big-endian byte length. The source body participates in that JSON
and checksum. `ParseChecksumV2` accepts only the v2 prefix followed by 64 lowercase
hexadecimal digits. V0/v1 parsers reject this prefix. Checksums detect changed
bytes; they do not authenticate the compiler or grant authority.

## Integration boundary

Tests cover two cleanup specializations calling a generic dependency, incoming,
outgoing, both and empty selections, exact final byte limits, version rejection,
independently computed checksums, body mutation, and legacy field omission.
Context Pack and Evidence Bundle adoption remain separate work.

## CLI selection

First discover the original resource SymbolId with
`atlas-symbols list --catalog-version 3 --kind resource --name Token` and the
required package/selection/output limits. Supply that ID to
`atlas-calls list --catalog-version 2` with the required package, symbol,
direction, package/call and output limits. Both commands use the same current
workspace and aggregate-aware uncached compiler. Incoming selection on a helper
function reports cleanup callers with their original resource IDs.

Only canonical catalog values 0, 1 and 2 are accepted once. Default 0 and explicit
0/1 keep their existing source-admission and output policies. `--locked` retains
its exact no-write behavior; otherwise lock refresh can precede later failures.
Source failures return diagnostics with no catalog, invalid selectors/budgets
return invocation errors with no partial bytes, and output writer failure remains
an internal failure. No map, catalog or compiler artifact cache is written.

This experimental selection does not promote aggregate syntax in stable commands;
language, SDK and runtime ABI revisions remain unchanged. CLI tests discover IDs
before querying cross-package cleanup calls, compare incoming/outgoing/both,
preserve lock bytes, check unused aggregates and enforce atomic failures.
