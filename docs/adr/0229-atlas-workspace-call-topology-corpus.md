# ADR 0229: Atlas workspace-call topology correctness corpus

- Status: Implemented deterministic correctness control
- Date: 2026-07-30
- Join prerequisite: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Catalog prerequisite: `docs/adr/0226-experimental-workspace-call-catalog-command.md`
- Scaling companion: `docs/adr/0228-atlas-workspace-call-scaling-control.md`

## Context

ADR 0228 varies package and call counts with generated star workspaces. That isolates a useful
scaling denominator, but one root calling many leaf packages cannot exercise a multi-level chain,
shared dependency, or multiple callers converging on one target. Treating the star shapes as
representative workspace structure would hide correctness gaps.

Synthetic non-star workspaces can close those deterministic shape gaps without pretending to be
real-world performance evidence. They should reuse the ordinary manifest, loader, compiler,
verified-MIR, Atlas planning, exact-current join, and canonical catalog paths instead of constructing
private test records by hand.

## Decision

1. Add a Go correctness corpus under `cmd/fairyc-windows-host-smoke` that creates complete temporary
   `.fy` workspaces and compiles them through the ordinary project pipeline.
2. Fix three closed topologies:
   - `topology-chain`: root calls middle and middle calls leaf;
   - `topology-diamond`: root calls left and right, both of which call leaf; and
   - `topology-fan-in`: root calls three workers, all of which call one sink.
3. Use canonical sorted workspace members, lowercase kebab-case manifest bindings, workspace-root-
   relative dependency paths, and explicit underscore module aliases where package names contain
   hyphens. The corpus must obey the public manifest rules rather than bypass them.
4. Require every generated workspace to produce clean verified MIR, exact logical partitions, and
   current-source expectations before constructing `WorkspaceCallIndex`.
5. Require exact package, total-call, and cross-package-call usage. For each topology, check root
   outgoing calls, one intermediate symbol in `both` direction, and the final shared target's
   incoming calls.
6. Encode every selected catalog twice and require byte identity under one explicit output bound.
7. Add no schema, CLI command, default budget, traversal, transitive-impact, indirect-call,
   performance, retained-memory, or representative-repository claim.

## Evidence

`TestAtlasCallTopologyCorpusJoinsNonStarWorkspaces` creates and compiles all three workspaces. It
requires 3 packages and 2 calls for the chain, 4 packages and 4 calls for the diamond, and 5 packages
and 6 calls for fan-in. Every call is cross-package. Direction checks freeze the expected 1/2/1,
2/2/2, and 3/2/3 selected-call distributions respectively, and repeated catalog encoding must be
byte-identical.

This corpus proves only those generated direct-call structures. Real repositories, broader package
and call distributions, indirect or foreign calls, other relation families, reference-machine
measurements, and product traversal or impact policy remain separate evidence.
