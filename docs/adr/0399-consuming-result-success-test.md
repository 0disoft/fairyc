# ADR 0399: Consuming Result success test

- Status: Implemented
- Date: 2026-09-23
- Product source: [product specification](../product/02-spec.md)
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

Admit `result.is_ok` for an owned `Result<T, E>`. It returns `true` for `.ok`
and `false` for `.err`, evaluating the receiver once. Its handling, ownership,
and cleanup semantics are those of the existing exhaustive expression
`switch (result) { case .ok(_): { true } case .err(_): { false } }`.
The active payload is dropped. Both possible payload types must permit implicit
drop, and the enclosing function must declare all required drop effects.
Resource-bearing Result locals move into the test; copyable locals follow their
ordinary copy rules. Borrowed Results, `Option`, payload extraction, and a
general tagged-value reflection property are not admitted. Existing fields
named `is_ok` on unrelated aggregates retain ordinary field semantics.

## Evidence and lowering

The [lexer candidate](../../examples/lexer_candidate/src/body_state.fy) and
other checked-in parser sources repeatedly convert a `Vec::push` Result to a
boolean with the exact two-arm switch above. A library helper would create a
specialized function for each payload type or add a call boundary, whereas a
compiler-known property can use the existing match path.

The type checker produces two ordinary typed-HIR match arms. Result Flow
discharges the must-use obligation; Resource Flow moves the receiver when it
owns resources and enforces the same drop ability and effect checks. MIR
lowering, the interpreter, and C emission use their existing Result switch,
payload extraction, and drop operations. No new runtime tag, helper, or
backend operation is introduced. The existing field-expression grammar is
unchanged.

The semantic and MIR regression tests cover scalar and resource payloads,
both branch shapes, required effects, forbidden implicit drop, and verified
cleanup paths. The lexer candidate adopts the property at two `Vec::push`
sites. A same-workspace comparison against their previous switches emitted
1,114,847 versus 1,114,718 C bytes and 407 MIR functions in both variants.
With MSVC 14.51.36231, the low-profile EXE was 410,624 bytes and the
optimized EXE 221,696 bytes in both variants; `.text` VirtualSize was 351,584
and 162,624 bytes respectively. Four fixed parser inputs produced matching
declaration and diagnostic output across four builds. The optimized runtime
object differed by 4 bytes while the final EXE size remained equal. The native
comparison does not exercise an allocation-failure branch of the newly adopted
sites, prove compiler-time parity, or establish the cumulative 8% budget
against a C implementation.

## Compatibility

This adds source behavior, so the exact language pin advances from
`fairyc-language/16` to `/17` and the bundled SDK from `0.0.0-stage0.61` to
`0.0.0-stage0.62`. Existing sources remain valid after manifest pin and lock
refresh. Runtime ABI and public-interface schemas are unchanged.
