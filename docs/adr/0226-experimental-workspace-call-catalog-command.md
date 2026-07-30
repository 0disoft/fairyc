# ADR 0226: Experimental exact workspace call catalog command

- Status: Implemented
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Prerequisite: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Architecture boundary: `docs/adr/0203-enforced-one-way-atlas-import-boundary.md`
- Output contract: `docs/atlas/workspace-call-catalog-v0.md`

## Context

ADR 0225 established an exact-current in-memory call join, but deliberately exposed no output or
command. Package-local Context Pack v1 must continue to report its dependency and incoming-package
omissions because changing that established output would hide which evidence it actually consumed.
The next useful consumer is narrower: select one exact workspace symbol and inspect only the direct
compiler-observed calls entering or leaving it.

This consumer must not imply transitive impact, indirect-call recovery, safe-change analysis, stable
anchors, or a complete program graph. It also must not invent default construction limits or serve
partial output after the workspace join exceeds a bound.

## Decision

Add this deliberately unstable command:

```text
fairyc experimental atlas-calls list \
  --package-uid <uid> \
  --symbol <visible-or-full-symbol-id> \
  --direction <outgoing|incoming|both> \
  --max-packages <count> \
  --max-calls <count> \
  --max-output-bytes <bytes> \
  [--locked]
```

The command compiles the manifest-owned workspace once and derives every logical partition and
independent current-source expectation from that same retained state. It then:

1. selects exactly one unique manifest package UID;
2. constructs the complete `WorkspaceCallIndex` under the caller's positive package and call
   limits;
3. resolves one canonical visible or full SymbolId within the selected package instance;
4. selects exact outgoing, incoming, or deduplicated bidirectional calls; and
5. writes one canonical checksummed `fairyc.workspace-call-catalog-json/v0` value with no trailing
   newline.

All six value flags are mandatory. There are no numeric, direction, package, or symbol defaults.
Counts are capped by the existing Atlas CLI safety ceiling of 65,536, while the final output bound
is capped at 1,073,741,824 bytes. Construction is atomic: missing dependencies, stale or
incompatible partitions, target contradictions, or package/call budget exhaustion reject the
command instead of producing a partial catalog. A valid subject with no selected calls succeeds
with `calls: []` and `complete: true`.

Ordinary mode may create or refresh the canonical lock before a later selection or budget failure.
`--locked` verifies the exact lock and writes nothing. For unchanged compiler bytes, workspace
bytes, lock, arguments, and target-independent map facts, ordinary and locked successful output is
byte-identical.

## Output and failure boundary

The output is a new output-only compatibility family. It does not revise `.fymap` v1, Context Pack
v1, Symbol Catalog v1, attributed-text v0, or Evidence Bundle v0. The envelope binds its version,
encoding, canonical logical catalog, and domain-separated full SHA-256 checksum. It carries only
compiler facts: exact package-qualified symbols, source span identity, certainty, provenance,
construction budget, usage, context, selected calls, and an explicit completion bit.

Stable prototype exit classes remain:

- `0`: one complete catalog was written;
- `2`: invalid or missing arguments, unknown package or symbol, construction-budget exhaustion, or
  final-output overflow;
- `3`: source diagnostics;
- `4`: internal contradiction or stdout failure; and
- `7`: manifest, workspace, graph, or lock failure.

## Consequences

The first external consumer of `WorkspaceCallIndex` is now measurable without mutating an existing
Atlas format or pretending package-local queries joined dependencies. The separate output package
is admitted to the one-way Atlas/export layer; compiler, backend, runtime, and library packages
remain forbidden from importing it.

This decision adds no decoder, persistent catalog, source text, documentation, map-store default,
workspace traversal, transitive closure, other relation family, indirect call, impact, semantic
diff, safe-change claim, stable anchor, LSP, SCIP, MCP, embedding, model call, permission, or
execution authority. Representative time and retained-memory measurements remain required before
considering traversal or a default budget. ADR 0227 supplies only a tiny fresh-process Windows
control whose whole-process peaks and uncontrolled OS caches do not satisfy that representative
gate.
