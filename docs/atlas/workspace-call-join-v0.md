# Exact-current workspace call join v0

- Status: Implemented internal prerequisite
- Decision: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Implementation: `internal/atlasmap/workspace_call_join.go`
- Package-local query prerequisite: `docs/atlas/fymap-query-v1.md`
- Current-source prerequisite: `docs/atlas/fymap-store-v1.md`
- Output consumer: `docs/atlas/workspace-call-catalog-v0.md`
- Process control: `docs/adr/0227-atlas-call-process-measurement-control.md`
- Scaling control: `docs/adr/0228-atlas-workspace-call-scaling-control.md`
- Topology corpus: `docs/adr/0229-atlas-workspace-call-topology-corpus.md`
- Retained-heap control: `docs/adr/0230-atlas-workspace-call-retained-heap-control.md`

## Purpose

`WorkspaceCallIndex` joins compiler-resolved calls across the complete exact-current package set
already retained by one verified workspace compilation. It supplies exact outgoing and reverse
incoming call ledgers without changing package-local query output or claiming a complete workspace
impact graph.

## Admission boundary

Construction requires:

- a nonempty set of logical `.fymap` partitions;
- exactly one independently derived `LoadExpectation` for every partition package instance;
- a positive maximum package count; and
- a nonnegative maximum resolved-call count.

Every partition re-enters `NewQueryIndex`. Therefore a successful join has revalidated canonical
transport facts, exact compiler executable identity, public-interface hash, package identity and
source digest, and every source-file content and line-map identity. Construction reads no source,
map store, directory, environment, process, network, or caller text.

The partition set must close every direct dependency named by every package header. Extra
expectations, duplicate packages, missing dependency partitions, and any stale or incompatible map
reject the complete join.

## Exact call identity

Only `calls` relations produced from a resolved internal call are joined. Each must retain exact
certainty, compiler-observed provenance, and a source span. The target package must be the source
package or one of its declared direct dependencies. The target resolves by the full compiler
snapshot key:

```text
PackageInstanceId + module + declaration kind + declaration name
```

The corresponding exact-current target declaration supplies its canonical visible SymbolId.
Cross-package targets must be public. An existing package-local SymbolId must agree with the joined
target. Any mismatch rejects construction.

`WorkspaceSymbol` carries both exact package instance and visible SymbolId. The package qualifier is
mandatory even though SymbolId includes a package-UID namespace: exact currentness and relative
source paths belong to one package instance.

## Query surface

### Opt-in generic-aware construction

`NewWorkspaceCallIndexV2` uses `NewQueryIndexV2` admission for every partition. It accepts only v2
maps and independently supplied v2 current-source expectations. Direct-dependency closure, public
cross-package targets, currentness, and package/call budgets remain mandatory. The v1 constructor
does not admit v2 maps.

Each `JoinedCall` retains `SourceSpecialization` and `TargetSpecialization` when present. The target
implementation must exist under the exact resolved target declaration; a generic target with no
implementation ID rejects the join. Both fields are empty for ordinary calls. Declaration selectors
aggregate the call records of their implementations without dropping implementation IDs or merging
different records. This is not concrete execution-path traversal. Returned IDs are current-build
compiler identities, not stable source anchors. The separate internal call catalog v1 encoder now
consumes this v2 entry point, as does `atlas-calls list --catalog-version 1`; the default v0
output/CLI remains on v1 construction. See
`docs/atlas/workspace-call-catalog-v1.md`.

### Aggregate-aware v3 maps

`NewWorkspaceCallIndexV3` explicitly admits v3 maps and their independently derived
current-source expectations. It keeps the same dependency closure, public-target,
concrete-target, package and call budget checks. The older constructors reject v3.

Calls originating in a concrete resource cleanup retain `SourceBody = "cleanup"`,
the original resource SymbolId and `SourceSpecialization`. Their targets retain
the original function SymbolId and `TargetSpecialization`, including across direct
package dependencies. Ordinary function calls keep an empty `SourceBody`. Incoming
and outgoing selectors return detached records without merging distinct cleanup
implementations that share source spans. Package context still marks universal
template-body facts unavailable. This ledger describes observed explicit calls;
it does not add edges for implicit drop scheduling or prove cleanup execution counts.

The v3 join is an internal opt-in API. Call catalog v2 preserves cleanup-owned
calls as documented in `docs/atlas/workspace-call-catalog-v2.md`, exposed through
`atlas-calls list --catalog-version 2` without changing the default command.

### Declaration selectors

- `Calls` returns every resolved call in deterministic package and relation order.
- `OutgoingCalls` selects calls made by one exact package-qualified visible or full SymbolId.
- `IncomingCalls` selects calls targeting one exact package-qualified visible or full SymbolId.
- `Usage` reports package, total call, and cross-package call counts.

Returned records are detached. Selector lookup never performs exact-name, prefix, fuzzy, or rename
search. Unknown packages and unknown or malformed SymbolIds fail instead of returning a guessed
empty result.

## Budget and complexity boundary

The package and call maxima are construction budgets, not defaults or performance promises. The
implementation sorts the supplied package set, builds one exact-current package index per package,
uses each already validated canonical sorted direct-dependency list for logarithmic target-admission
lookup, and stores one ledger record per resolved call. Full package-local query indexes exist only
during construction; the returned index retains the call ledger plus detached call-completeness
context and visible/full SymbolId canonicalization tables for each package. This structural reduction
is not a measured retained-memory result. No latency, retained-memory, representative-project, or
model-token threshold is currently accepted. ADR 0227 records only a tiny fresh-process Windows
control whose whole-process peaks cannot be attributed to this index. ADR 0228 adds three generated
star shapes around index construction plus outgoing encoding, but its process-wide allocation deltas
are not retained-index memory and its shape growth proves no complexity law.
ADR 0229 adds generated chain, diamond, and fan-in correctness controls for direct calls, but does
not promote those synthetic shapes to representative-workload or traversal evidence.
ADR 0230 adds forced-GC process live-heap deltas for fixed 16-index batches over the same generated
star shapes. Prepared inputs and runtime state remain in the process, so those observations are not
exact Atlas ownership, per-index cost, a product budget, or representative evidence.

## Completeness boundary

Success proves exact joins for the resolved non-foreign calls present in the supplied complete
workspace snapshot. It does not prove indirect-call recovery, runtime dispatch, source outside that
workspace, a reverse index persisted across compilations, or completeness of unrelated relation
families.

Package-local `show` and `around` still emit `dependency-map-not-joined` and
`incoming-package-maps-not-joined`, because they do not consume this workspace ledger. Context Pack
v1 and its package-local experimental commands remain byte-identical. ADR 0226 exposes the joined
direct-call ledger only through a separate workspace call catalog v0 command.

## Not enabled

- no workspace `show`, `around`, Context Pack, Evidence Bundle, or graph-traversal CLI; the only
  output consumer is exact-symbol direct-call catalog v0;
- no `.fymap` schema or store change;
- no automatic dependency-map discovery or default budget;
- no transitive `impact`, semantic `diff`, safe-change, stable anchor, rename, LSP, SCIP, MCP, or
  structural-edit claim; and
- no source text, documentation, model call, embedding, permission, or execution authority.
