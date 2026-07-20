# ADR 0103: Checked left shift

- Status: Accepted
- Date: 2026-07-20
- Language surface: matching-type checked `<<` for `i32`, `u8`, `u32`, and `usize`

## Context

Stage 0 originally rejected left shift because inheriting C's signed-shift and overflow rules would
make safe Fairy C behavior depend on backend undefined behavior. ADR 0102 established exact-width
checked right shift and matching-type integer operations, leaving enough typed MIR and backend
structure to define left shift without promotions or host-language leakage.

## Decision

1. Binary `<<` admits exactly two matching operands of `i32`, `u8`, `u32`, or `usize` and returns
   that same type. Mixed types, implicit promotions, C-scalar operands, and wider integer families
   remain unsupported.
2. The shift amount uses the exact operand type and width. A negative `i32` amount or an `i32`
   amount at least 32 traps as `invalid-shift`. Unsigned amounts trap at or above 8 for `u8`, 32
   for `u32`, and 64 for `usize` on the currently admitted targets.
3. After the amount is accepted, the mathematical result `value * 2^amount` must be representable
   in the exact operand type. Otherwise the operation traps as `integer-overflow`; it never wraps.
4. Trap precedence is stable: `invalid-shift` is checked before `integer-overflow`. An input that
   has an invalid amount and would overflow with a valid amount reports `invalid-shift`.
5. MIR lowers left shift through an arithmetic-check marker, a `checked-shift` switch, and a
   separate continuation binary operation. The ordered categories are exactly
   `invalid-shift`, then `integer-overflow`. `V1501` rejects missing, reordered, or mismatched
   marker metadata before interpretation or C emission.
6. The interpreter and generated C perform the width check before the representability check and
   perform both before the shift. Generated C computes the signed overflow predicate in `int64_t`,
   performs the admitted `i32` shift through `uint32_t`, and reconstructs the signed result from
   bits. Unsigned checks use the exact maximum shifted right by the already-validated amount.
7. The semantic fixture contract contains 22 cases after removing the obsolete `E1011` rejection.
   The interpreter contract grows to 77 cases and C emission to 27. The derived Linux matrix plans
   108 low/optimized compiler executions and 54 sanitizer executions. The WSL-free Windows smoke
   executes left-shift success and overflow under both MSVC profiles, bringing direct `run-host`
   outcomes to 32.

## Consequences

- Safe left shift has one backend-independent meaning and two explicit trap outcomes.
- Constant and dynamic inputs cannot reach a C shift until the amount and mathematical result are
  valid, so optimization level cannot turn a Fairy C trap into C undefined behavior.
- Rotates, wrapping shifts, saturating shifts, mixed-type shifts, and shift-result APIs remain
  separate future language decisions.

## References

- `docs/adr/0023-explicit-checked-arithmetic-mir.md`
- `docs/adr/0072-defined-checked-arithmetic-c-construction.md`
- `docs/adr/0102-typed-bitwise-and-checked-right-shift.md`
- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
