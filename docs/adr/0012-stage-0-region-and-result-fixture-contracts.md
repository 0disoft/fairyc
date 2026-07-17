# Stage 0 Region and Result Fixture Contracts

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Region safety and `Result` handling are both Flow concerns, but they expose different stable facts. Region checks need hidden origins, escape decisions, and capability restrictions. `Result` checks need handling mode, single evaluation, payload types, and propagation compatibility. Folding both into the general semantic schema would weaken validation and silently broaden an existing v0 contract.

The local lexical-region escape case depends on a real `std::alloc::Arena` package instance. The resolved package graph provides explicit standard-library identity and injection. A bounded fixture now exports an opaque resource type and lets lexical Region syntax borrow it through `&mut std::alloc::Arena`, without choosing construction or capacity APIs.

## Decision

1. `spec/fixtures/region/manifest-v1.json` owns current Stage 0 Region origin and capability acceptance data. v1 adds an optional resolved-package-graph reference; v0 remains historical.
2. `spec/fixtures/result/manifest-v0.json` owns Stage 0 `Result` handling and `?` acceptance data.
3. Every case declares whether it completes through Flow or fails in `type` or `flow`. Parsing and resolution must always succeed; an unlisted earlier-phase diagnostic is a fixture setup failure.
4. Region pass assertions record the allocation operation, source span, hidden origin, last use when local, and whether the derived borrow escapes through the return value.
5. Region fail assertions use complete independent `E31xx` diagnostics. The cases cover a local use from `RegionParam(0)`, a permitted single-origin return, lexical `LocalRegion` escape, resource placement rejection, capability storage rejection, and direct capability construction rejection.
6. A package-aware case names `pass.package.stdlib-arena-region`, its `app` root node, and the `std::alloc` provider. The fixture-owned `Arena` resource is opaque to the app and defines no constructor or ordinary allocation methods.
7. Result pass assertions record the expression, handling mode (`return`, `discard`, or `propagate`), canonical result, success, and error types, and exactly-once evaluation.
8. Result fail assertions use complete independent `E41xx` diagnostics. The initial cases cover an unused result, `?` on a non-result, and incompatible propagated error types.
9. `Ok` and `Err` construction syntax is not added by these fixtures. Cases use typed parameters so the suite does not choose a constructor spelling ahead of the language contract.
10. Full active-payload move and drop traces remain MIR or runtime fixture concerns rather than Flow-schema fields.

## Rationale

Separate schemas keep a Region origin from being confused with a `Result` payload type and let each runner reject unknown fields. Region v1 makes package resolution explicit only for the case that needs standard-library type identity. The remaining parameter-driven cases and all Result cases avoid smuggling in constructors that are still open decisions.

## Consequences

- A compiler may organize Region and Result analysis however it wants internally, but it must reproduce the declared owning phase, canonical source facts, and diagnostics.
- Region v1 covers `E3101` without papering over standard-library ownership with a fake local arena.
- The first Result suite proves handling classification and error compatibility, not concrete tagged representation or cleanup order.
- No fixture currently proves an implementation because no compiler runner exists.

## Revisit conditions

Add or version fixtures when public `Arena` construction spelling is accepted, `Ok` and `Err` construction spelling is accepted, resource-bearing tagged payload traces become executable, or a required assertion depends on MIR control-flow identity.
