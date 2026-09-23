# ADR 0402: Direct-return-only Result switch arm

- Status: Implemented
- Date: 2026-09-24
- Product source: [product specification](../product/02-spec.md)
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

A shallow Result `switch` may combine a value-producing case with a case block
whose final statement is a direct `return` and which has no tail expression.
The returning case does not supply a value to the switch join. Its return
expression is checked against the enclosing function's result type. A
continuing empty or void case still fails when the switch needs a value.

This is deliberately narrower than a general `never` type: there is no new
source type, implicit coercion, nested control-flow inference, `let-else`, or
change to Option and user-enum switches. The admitted value-producing form
requires at least one continuing case.

## Lowering and cost

The checker ignores the return-only case when selecting or comparing the
switch's result type. Existing Result ownership and must-use flow checks,
verified MIR branches and cleanup, and constrained C emission remain in use.
The other case still provides the join value. No runtime helper, metadata,
tag, or new backend operation is introduced. A same-behavior C comparison is
still needed before claiming a measured build-time or binary-size result.

## Compatibility and evidence

This changes accepted source semantics, so the exact language pin advances
from `/17` to `/18` and the bundled SDK from `0.0.0-stage0.64` to
`0.0.0-stage0.65`. Existing source remains valid after manifest pin and lock
refresh. Runtime ABI, MIR version, and public-interface schemas are unchanged.

`TestResultBranchReviewReturnOnlyArm` verifies resource-bearing binding and
the complete verified compiler path. The void-arm negative case retains
`E1006`. Windows low and optimized native runs cover the success and early
return paths. ADR 0398's example-only deferral is superseded for this one
Result arm shape; its broader `never` and `let-else` candidates remain deferred.
