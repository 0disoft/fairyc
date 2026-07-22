# Atlas Semantic Snapshot Boundary

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Fairy Atlas should help humans and tools navigate compiler facts without turning the compiler frontend into an LLM product, allowing documentation to masquerade as authority, or making type checking depend on an optional indexer.

## Decision

1. Typed HIR and analyzed MIR produce an immutable, deterministic `SemanticSnapshot`.
2. Atlas exporters and queries consume that snapshot. Frontend, analysis, and MIR packages cannot import Atlas packages.
3. Atlas failure cannot change diagnostics, type-check success, or generated program semantics.
4. Build-local node identity is never serialized as stable external identity.
5. Stable symbol identity is derived from semantic coordinates and a separately versioned
   algorithm. ADR 0183 implements the first provisional algorithm with collision and compatibility
   regressions; released cross-version stability remains deferred.
6. Compiler facts remain in body-free map and pack contracts. ADR 0195 refines free text into a
   separate attributed companion with fixed `untrusted-text` authority and explicit origin.
7. Unknown, candidate, external, and opaque graph edges remain explicit. Atlas does not invent a complete call graph.
8. Transitive closure is computed at query time rather than stored eagerly.
9. Compiler-fact Context Packs report budget, completeness, and omissions. Any future attributed-
   text companion separately reports origin, byte budgets, usage, and omissions.
10. Embeddings, model calls, vector databases, automatic rename certainty, structural edits, agent permission annotations, and source-body duplication are outside the initial schema.

## Rationale

The one-way snapshot boundary keeps compiler correctness independent of Atlas and gives tests a deterministic seam. Provenance and incompleteness are part of the data model because a plausible but false graph is more dangerous to an automated tool than an explicit unknown.

## Consequences

- Whole-package `.fymap` rewrites are acceptable initially.
- Atlas queries may be conservative when foreign or indirect calls are opaque.
- A future attributed-text companion may improve navigation but cannot override compiler facts,
  completeness, prompt roles, or host authority; Context Pack v1 remains text-free.
- `impact` and semantic `diff` remain deferred until uncertainty reporting is demonstrated.

## Revisit conditions

Consider finer-grained map chunks, compression, new query families, or stable anchors only after measured package size, query latency, rewrite cost, and real navigation usage justify the added compatibility surface.
