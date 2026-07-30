# Exact-current workspace call join v0

- Status: Implemented internal prerequisite
- Decision: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Implementation: `internal/atlasmap/workspace_call_join.go`
- Package-local query prerequisite: `docs/atlas/fymap-query-v1.md`
- Current-source prerequisite: `docs/atlas/fymap-store-v1.md`
- Output consumer: `docs/atlas/workspace-call-catalog-v0.md`
- Process control: `docs/adr/0227-atlas-call-process-measurement-control.md`
- Scaling control: `docs/adr/0228-atlas-workspace-call-scaling-control.md`

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
and stores one ledger record per resolved call. No latency, retained-memory, representative-project,
or model-token threshold is currently accepted. ADR 0227 records only a tiny fresh-process Windows
control whose whole-process peaks cannot be attributed to this index. ADR 0228 adds three generated
star shapes around index construction plus outgoing encoding, but its process-wide allocation deltas
are not retained-index memory and its shape growth proves no complexity law.

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
