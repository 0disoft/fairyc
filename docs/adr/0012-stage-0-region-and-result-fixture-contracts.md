# Stage 0 Region and Result Fixture Contracts

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Region safety and `Result` handling are both Flow concerns, but they expose different stable facts. Region checks need hidden origins, escape decisions, and capability restrictions. `Result` checks need handling mode, single evaluation, payload types, and propagation compatibility. Folding both into the general semantic schema would weaken validation and silently broaden an existing v0 contract.

The local lexical-region escape case also depends on a real `std::alloc::Arena` package instance. Package manifests, dependency injection for fixtures, and standard-library fixture identity remain undecided, so inventing a magical local `Arena` would test a language that has not been specified.

## Decision

1. `spec/fixtures/region/manifest-v0.json` owns Stage 0 Region origin and capability acceptance data.
2. `spec/fixtures/result/manifest-v0.json` owns Stage 0 `Result` handling and `?` acceptance data.
3. Every case declares whether it completes through Flow or fails in `type` or `flow`. Parsing and resolution must always succeed; an unlisted earlier-phase diagnostic is a fixture setup failure.
4. Region pass assertions record the allocation operation, source span, hidden origin, last use when local, and whether the derived borrow escapes through the return value.
5. Region fail assertions use complete independent `E31xx` diagnostics. The initial cases cover a local use from `RegionParam(0)`, a permitted single-origin return, resource placement rejection, capability storage rejection, and direct capability construction rejection.
6. A lexical local-region escape fixture is deferred until the fixture contract can name a real standard-library package instance and `Arena` dependency without an implicit prelude or invented constructor.
7. Result pass assertions record the expression, handling mode (`return`, `discard`, or `propagate`), canonical result, success, and error types, and exactly-once evaluation.
8. Result fail assertions use complete independent `E41xx` diagnostics. The initial cases cover an unused result, `?` on a non-result, and incompatible propagated error types.
9. `Ok` and `Err` construction syntax is not added by these fixtures. Cases use typed parameters so the suite does not choose a constructor spelling ahead of the language contract.
10. Full active-payload move and drop traces remain MIR or runtime fixture concerns rather than Flow-schema fields.

## Rationale

Separate schemas keep a Region origin from being confused with a `Result` payload type and let each runner reject unknown fields. Parameter-driven cases exercise the already accepted type forms and `?` syntax without smuggling in package resolution, stdlib assembly, or variant constructors that are still open decisions.

## Consequences

- A compiler may organize Region and Result analysis however it wants internally, but it must reproduce the declared owning phase, canonical source facts, and diagnostics.
- The first Region suite is intentionally incomplete for `E3101`; the gap is visible rather than papered over with a fake arena.
- The first Result suite proves handling classification and error compatibility, not concrete tagged representation or cleanup order.
- No fixture currently proves an implementation because no compiler runner exists.

## Revisit conditions

Add or version fixtures when package and stdlib dependency identity is fixed, `Ok` and `Err` construction spelling is accepted, resource-bearing tagged payload traces become executable, or a required assertion depends on MIR control-flow identity.
