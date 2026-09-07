# Workspace call catalog v0

- Status: Implemented experimental output-only contract
- Decision: `docs/adr/0226-experimental-workspace-call-catalog-command.md`
- Join prerequisite: `docs/atlas/workspace-call-join-v0.md`
- Implementation: `internal/workspacecallcatalog/`
- Experimental command: `internal/cli/atlas_calls.go`
- Process control: `docs/adr/0227-atlas-call-process-measurement-control.md`
- Scaling control: `docs/adr/0228-atlas-workspace-call-scaling-control.md`
- Topology corpus: `docs/adr/0229-atlas-workspace-call-topology-corpus.md`
- Retained-heap control: `docs/adr/0230-atlas-workspace-call-retained-heap-control.md`

## Identity

The v0 encoder emits one compact canonical JSON value with no trailing newline:

```text
catalog version:    fairyc.workspace-call-catalog/v0
encoding:           fairyc.workspace-call-catalog-json/v0
checksum algorithm: fairyc.workspace-call-catalog-checksum/v0-sha256
checksum prefix:    fywcall0-sha256-
```

This is an output-only pre-release format. No decoder or cross-version input is admitted.
`Encode` requires v1 maps and rejects a v2 workspace index rather than dropping generic endpoint
identity. The separate internal `EncodeV1` contract is documented in
`docs/atlas/workspace-call-catalog-v1.md`; the CLI defaults to v0 and accepts explicit
`--catalog-version 1` for generic-aware output.

## Canonical envelope

Fields are emitted in this exact order:

```json
{
  "catalog_version": "fairyc.workspace-call-catalog/v0",
  "encoding": "fairyc.workspace-call-catalog-json/v0",
  "checksum_algorithm": "fairyc.workspace-call-catalog-checksum/v0-sha256",
  "checksum": "fywcall0-sha256-...",
  "catalog": {
    "context": {},
    "subject": {"package_instance": "...", "symbol": "..."},
    "direction": "outgoing",
    "budget": {"max_packages": 2, "max_calls": 8},
    "usage": {"packages": 2, "calls": 1, "cross_package_calls": 1, "selected_calls": 1},
    "calls": [],
    "complete": true
  }
}
```

`context` is the selected package's detached exact-current `atlasmap.QueryContext`, including map,
package, compiler/interface, completeness, and omission identity. Empty context slices and `calls`
are encoded as `[]`, never `null`.

`subject` always uses the exact package instance and canonical visible SymbolId resolved from the
caller's visible or full selector. Every call has this closed shape and order:

```json
{
  "source": {"package_instance": "...", "symbol": "..."},
  "target": {"package_instance": "...", "symbol": "..."},
  "span": {"package_instance": "...", "source": "module.fy", "start_byte": 0, "end_byte": 1},
  "certainty": "exact",
  "provenance": "compiler-observed"
}
```

`span.source` is a canonical package-relative source path, not a host path. No source body,
documentation, free text, generated summary, or caller annotation is present.

## Selection and ordering

The direction vocabulary is exactly `outgoing`, `incoming`, or `both`. Outgoing selects calls whose
source is the subject. Incoming selects calls whose target is the subject. Both concatenates those
sets, orders them canonically, and removes an exact duplicate such as a self-call observed from both
directions.

Calls sort by source package, source SymbolId, target package, target SymbolId, span package, source
path, start byte, and end byte. The selected order is independent of caller partition order and map
record order. There is no name, prefix, substring, fuzzy, relevance, or probabilistic selection.

## Budget, usage, and completeness

`budget.max_packages` and `budget.max_calls` echo the complete workspace-join construction limits.
They are not pagination limits and have no defaults. `usage.packages`, `usage.calls`, and
`usage.cross_package_calls` describe the complete accepted joined ledger;
`usage.selected_calls` describes the selected output slice.

Budget exhaustion fails the complete operation. The encoder separately requires a positive final
byte limit and returns no bytes when the envelope exceeds it. That final byte limit is command
policy rather than a logical catalog field, matching the other canonical Atlas output writers.

`complete: true` means the output contains every direct non-foreign compiler-resolved call matching
the subject and direction within the admitted complete exact-current workspace snapshot. It does
not mean indirect calls, runtime dispatch, foreign call targets, source outside the workspace,
unrelated relation families, or a transitive impact graph are complete.

## Checksum

The checksum is SHA-256 over a domain-separated, length-delimited sequence containing:

1. the catalog version bytes;
2. the encoding bytes; and
3. the compact canonical JSON bytes of the logical `catalog` value.

The domain is `fairyc.workspace-call-catalog-checksum` followed by two zero bytes. Each member is
prefixed by its unsigned 64-bit big-endian byte length. The checksum provides deterministic
integrity, not authentication, trust, permission, or freshness by itself.

## Non-goals

The format adds no decoder, persistence, map discovery, default budget, Context Pack mutation,
source text, documentation, other relation family, workspace graph traversal, transitive impact,
safe-change analysis, stable anchor, editor protocol, model integration, or execution authority.
