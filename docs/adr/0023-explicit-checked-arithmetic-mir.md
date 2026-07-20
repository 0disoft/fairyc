# ADR 0023: Explicit checked arithmetic MIR

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0004-results-traps-and-control-flow.md`
- MIR source: `docs/mir/mir-v0.md`
- Fixture sources: `spec/fixtures/mir/manifest-v0.json`, `spec/fixtures/mir-verifier/manifest-v0.json`, and `spec/fixtures/interpreter/manifest-v0.json`

## Context

The interpreter already checked `i32` arithmetic while evaluating an arithmetic operation, but the
MIR graph made only division checks explicit. That let the Go interpreter produce the specified
trap while leaving a later backend without a verified edge that distinguished success from
overflow or an invalid shift. It also let a trapping arithmetic operation survive verification
without proof that its operands, operator, trap categories, and continuation belonged to the same
check.

Stage 0 additionally excludes signed left shift. Accepting `i32 << i32` merely because the parser
and prototype evaluator happened to understand it would contradict the language and C transport
contracts.

## Decision

1. Ordinary `i32` unary negation, addition, subtraction, and multiplication lower through an
   explicit arithmetic-check marker, a two-successor checked switch, a non-returning trap block,
   and a distinct continuation containing the arithmetic operation.
2. Signed right shift uses the same shape with `invalid-shift` as its only trap category. A shift
   amount below zero or at least 32 traps; an accepted right shift preserves signed arithmetic
   shift behavior.
3. Division and remainder retain their existing `division-by-zero` and
   `signed-division-overflow` categories, but their marker and switch now also carry the exact
   operands and operator checked for the continuation operation.
4. `V1501` requires the continuation arithmetic operation to have one checked predecessor whose
   marker, switch, operands, operator, success edge, trap edge, and ordered category set agree.
   The canonical v0 lowering therefore uses a fresh single-predecessor continuation block.
5. The MIR interpreter selects the success or trap outcome from the checked switch. It does not
   discover overflow for the first time when executing the continuation operation.
6. Left shift remains outside Stage 0 and is rejected during type checking as `E1011`.
   It produces no executable MIR.
7. MIR normalization fixtures record the static possible categories. Interpreter and CLI fixtures
   record the one dynamic category actually selected for a concrete input.

## Consequences

- A later backend can implement checked arithmetic from verified MIR without reconstructing source
  operators or inheriting host-language signed-overflow behavior.
- Arithmetic traps remain non-unwinding and perform no language cleanup.
- The verifier mutation suite now includes a valid checked-arithmetic graph and proves that removing
  its marker rejects the continuation operation with `V1501`.
- This decision proves source-to-interpreter behavior only. It does not prove generated C, native
  code, optimizer check elimination, or target conformance.

## Rejected alternatives

- Keep checks inside the Go arithmetic evaluator: makes interpreter behavior stronger than MIR and
  leaves backends to rediscover semantics.
- Treat overflow as a build-profile choice: violates stable safe-code behavior.
- Admit signed left shift with the evaluator's current behavior: turns an implementation accident
  into source semantics without resolving negative inputs, representability, or backend transport.
- Encode all arithmetic failures as one trap: loses the stable distinction between
  `integer-overflow` and `invalid-shift`.
