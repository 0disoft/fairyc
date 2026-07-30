# ADR 0225: Bounded exact-current workspace call join

- Status: Implemented
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Atlas source: `docs/atlas/workspace-call-join-v0.md`
- Package-local prerequisite: `docs/adr/0188-bounded-current-fymap-queries.md`
- Phase boundary: `docs/atlas/phase4-exit-evidence.md`
- Depends on: ADR 0182, ADR 0183, ADR 0185, ADR 0188, ADR 0203

## Context

Package-local Atlas queries deliberately retain a cross-package call as a declaration descriptor
and report that its dependency map is not joined. That is honest, but it leaves an LLM-facing or IDE
consumer unable to connect a root-package call to the exact dependency SymbolId or ask for exact
incoming callers. Jumping directly to stable anchors, `impact`, or a workspace graph would hide the
missing prerequisite behind a larger and less trustworthy feature.

The verified workspace pipeline already holds the necessary facts at one bounded point: every
logical package partition, every independently derived exact-current `LoadExpectation`, and every
resolved call target as `PackageInstance + module + declaration kind + name`. The missing operation
is an in-memory exact join over those caller-supplied facts. It needs no source scan, map discovery,
file reopening, network request, model call, or guessed name match.

## Decision

1. Add an internal immutable `WorkspaceCallIndex` over a caller-supplied package-partition set,
   exact load-expectation set, and explicit maximum package and call counts.
2. Rebuild each package through `NewQueryIndex`, preserving its complete transport, compiler-build,
   public-interface, package-source, file-content, and line-map validation. The expectation and
   partition package sets must be identical.
3. Sort packages by exact `PackageInstanceId` before joining. Input order must not affect the call
   ledger or either direction of lookup.
4. Require every declared direct dependency to have an exact-current partition. A non-foreign call
   may target only its own package or one declared direct dependency and must resolve by the complete
   snapshot `DeclarationKey`. Cross-package targets must be public.
5. Admit only exact, compiler-observed resolved calls. Missing spans, opaque or foreign calls,
   absent declarations, package-set contradictions, stale maps, incompatible maps, duplicate
   packages, and exhausted budgets fail closed instead of becoming partial joined facts.
6. Qualify every joined symbol with both the exact package instance and canonical visible SymbolId.
   This keeps identical package-relative source paths and stable SymbolIds scoped to the map that
   owns them.
7. Expose detached deterministic all-call, outgoing-call, and incoming-call records plus exact
   package/call/cross-package usage counts. Both visible and full SymbolId selectors canonicalize to
   the visible joined identity.
8. Keep the package-local `QueryIndex`, Context Pack v1, experimental commands, `.fymap` transport,
   and omission meanings unchanged. This join ledger is not yet workspace traversal or output.
9. Do not infer indirect calls, dependency closure beyond the supplied verified workspace,
   transitive impact, rename identity, safe-change reachability, dynamic dispatch, stable anchors,
   LSP, SCIP, MCP, structural edits, permissions, or model behavior.

## Consequences

Fairy Atlas can now prove one cross-package relation that was previously only describable: the
quickstart root call joins to the exact dependency SymbolId, and the dependency symbol reports that
exact root caller. Local calls use the same ledger without being mislabeled cross-package. Missing
maps and stale or contradictory evidence reject the whole join.

The implementation adds no file or network I/O, persistent index, public command, wire schema,
language revision, MIR change, runtime surface, target behavior, or authority. A later workspace
query proposal may consume this ledger, but it must budget result selection and preserve any
non-call relation or unsupported-package omissions separately.
