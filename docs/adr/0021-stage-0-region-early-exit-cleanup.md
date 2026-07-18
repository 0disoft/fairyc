# ADR 0021: Stage 0 Region early-exit cleanup

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0003-regions-and-arenas.md`
- MIR source: `docs/mir/drop-elaboration-v0.md`

## Context

ADR 0020 made Region fallthrough executable but rejected `return` and Result `?` paths that cross
a Region boundary. Adding a reset immediately before a function return would be incorrect for
nested Regions: resources owned by the inner lexical layer must drop before its reset, while
resources owned by an outer layer must remain live until the inner reset has completed.

## Decision

MIR lowering maintains a lexical Region stack. A function exit walks that stack from inner to
outer. For each layer it emits reverse-order drops for live resources declared in that Region,
then one `region-reset`; after all Region layers close, it drops the remaining function resources.

An explicit `return` and the error edge of Result `?` use the same scope-indexed cleanup routine.
The success edge of `?` keeps every enclosing Region active. A reset operation records one closed
exit kind: `fallthrough`, `return`, or `propagate`. The MIR verifier rejects a reset with any other
kind and still requires every normal return to have no active Region.

Source-linked MIR observations record each early reset, its exit span and kind, and the resources
dropped immediately before it. Interpreter traces expose the same exit kind without exposing an
Arena address, offset, capacity, or representation.

`break` and `continue` remain deferred because the current HIR/MIR slice does not yet implement
those control transfers. Traps remain non-unwinding and do not run Region cleanup.

## Consequences

- Direct return, Result error propagation, and nested Region return paths now execute through the
  same verified cleanup machinery as normal fallthrough.
- Cleanup facts can contain `region-reset` actions between resource drops; consumers must preserve
  their order rather than treating cleanup as a drop-only list.
- The unstable one-file interpreter can run these paths, but this does not provide native code,
  user-defined drop-body effects, I/O, or self-hosting.

## Rejected alternatives

- Reset every active Region after one global drop pass: destroys outer resources before the inner
  Region closes and loses the lexical ownership boundary.
- Duplicate return and `?` cleanup lowering: invites ordering drift between equivalent exits.
- Treat reset as an interpreter-only action: lets future backends disagree with verified MIR.
