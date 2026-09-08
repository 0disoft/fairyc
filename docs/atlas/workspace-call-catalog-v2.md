# Workspace call catalog v2

- Status: Implemented opt-in output-only encoder; CLI integration pending
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Join: `NewWorkspaceCallIndexV3`, `docs/atlas/workspace-call-join-v0.md`
- Implementation: `internal/workspacecallcatalog/catalog_aggregate.go`

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
CLI, Context Pack and Evidence Bundle adoption remain separate work. Stable
language, SDK and runtime ABI versions are unchanged by this internal encoder.
