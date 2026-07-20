# ADR 0101: Checked unsigned division and remainder

- Status: Accepted
- Date: 2026-07-20
- Language surface: matching-type `/` and `%` for `u8`, `u32`, and `usize`

## Context

Stage 0 already gives `i32` division and remainder explicit zero and signed-overflow trap edges.
Unsigned values could be compared and use checked addition, subtraction, and multiplication, but
could not divide. That blocks ordinary length, chunk, alignment, and hashing calculations.

## Decision

1. Binary `/` and `%` admit matching operands of `i32`, `u8`, `u32`, or `usize` and return that type.
2. A zero divisor traps as `division-by-zero` for every admitted type. Only `i32` minimum divided or
   remaindered by negative one can additionally trap as `signed-division-overflow`.
3. No promotion, signedness conversion, wrapping mode, or build-profile-dependent behavior occurs.
4. The MIR marker, binary result, and switch carry the same operand/result type. `V1501` rejects a
   missing or type-forged check edge before interpretation or C emission.
5. Generated C materializes the checked divisor through a function-local volatile slot of the exact
   Fairy integer representation. This preserves strict-C compilation even for a source constant zero.
6. Unsigned shifts and bitwise operators remain separate in this decision and are admitted later by
   ADR 0102. Wider integer families and mixed arithmetic remain deferred.

## Consequences

- Safe programs can perform ordinary unsigned quotient and remainder calculations.
- Interpreter and generated C share one explicit zero-divisor contract.
- The constrained C backend pays one volatile store/read per checked division or remainder.

## References

- `docs/adr/0023-explicit-checked-arithmetic-mir.md`
- `docs/adr/0071-checked-division-c-materialization.md`
- `docs/adr/0100-checked-unsigned-arithmetic.md`
- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/mir/mir-v0.md`
