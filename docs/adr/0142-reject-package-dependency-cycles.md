# ADR 0142: Reject resolved package dependency cycles in Stage 0

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/package-graph-v0.md`
- Fixture source: `spec/fixtures/package/manifest-v1.json`

## Context

The resolved package graph previously validated identity, direct visibility, duplicate bindings,
and module providers without rejecting cycles. A public manifest can therefore be perfectly parsed
yet still hand the compiler a graph with no valid package build order. Fairy C Stage 0 has not
defined cyclic initialization, mutually recursive package compilation, duplicate mutable package
state, partial interface loading, or cycle-aware cache scheduling.

Rejecting cycles only in a future manifest layer would leave other already-resolved graph producers
able to bypass the rule. The invariant belongs at the reusable `internal/packagegraph` boundary.

## Decision

1. Stage 0 resolved package graphs are directed acyclic graphs after exact PackageInstanceId class
   deduplication.
2. An edge whose owner and target deduplicate to the same instance is a self-cycle and rejects,
   even when the fixture used two different node handles.
3. Duplicate binding rejection keeps precedence over cycle detection. Cycle rejection keeps
   precedence over import lookup, so an unrelated missing or ambiguous import cannot hide the
   structural graph defect.
4. Adjacency lists sort by build-local resolved class order before deterministic depth-first search.
   The first discovered simple cycle reports each involved instance class once and omits the
   repeated closing node.
5. Fixture subjects contain every node handle belonging to the reported instance classes. This is
   repository evidence vocabulary, not a stable user diagnostic path or ordering promise.
6. `package-dependency-cycle` is added as a v1 fixture rejection reason. The v1 manifest explicitly
   includes all historical v0 cases; the runner continues to accept v0 directly.
7. This decision does not choose manifest syntax, workspace discovery, source kinds, version
   solving, feature unification, or a released diagnostic code.

## Consequences

- Every resolved graph producer, including a later lockfile reader, fails closed on cycles.
- Package checking, cache scheduling, and later topological build order may rely on acyclicity after
  successful resolution.
- Mutually recursive declarations remain possible within one package; this decision concerns only
  package dependency edges.
- Supporting package cycles later requires explicit initialization, interface-loading, build-order,
  cache, and diagnostic semantics plus a new package fixture version.

## Verification

- a v1 filesystem-backed fixture rejects a two-package cycle;
- unit tests reject a three-package cycle before a missing import;
- unit tests reject a self-cycle created only after equal-instance deduplication;
- all eight historical v0 cases remain executable through both the v0 reader and the v1 include.
