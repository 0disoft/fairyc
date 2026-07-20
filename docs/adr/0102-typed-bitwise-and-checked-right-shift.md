# ADR 0102: Typed bitwise operations and checked right shift

- Status: Accepted
- Date: 2026-07-20
- Language surface: typed `~`, matching-type `&`, `^`, `|`, and checked `>>` for `i32`, `u8`, `u32`, and `usize`
- Superseded in part: ADR 0103 later admits checked matching-type left shift and removes `E1011`

## Context

Stage 0 already executed `i32` bitwise operations and arithmetic right shift, while unsigned
integers could use checked arithmetic, division, remainder, casts, and comparisons. Leaving
bitwise work and right shift restricted to `i32` forced otherwise unsigned mask, flag, byte, and
size calculations through unsupported conversions.

Copying C's implicit promotions would make the result type and shift width depend on the backend.
It would also let an out-of-range shift become C undefined behavior before the Fairy C trap edge
could run.

## Decision

1. Unary `~` admits one operand of `i32`, `u8`, `u32`, or `usize`, flips exactly the bits of that
   type, and returns the same type without trapping or integer promotion.
2. Binary `&`, `^`, and `|` admit exactly two matching operands of `i32`, `u8`, `u32`, or `usize`
   and return that same type. They do not promote, change signedness, trap, or depend on build profile.
3. Binary `>>` admits the same matching types and returns that type. `i32` uses signed arithmetic
   right shift. `u8`, `u32`, and `usize` use logical right shift.
4. A negative `i32` shift amount or an amount at least 32 traps as `invalid-shift`. Unsigned shift
   amounts trap at or above the operand width: 8 for `u8`, 32 for `u32`, and 64 for the currently
   admitted `usize` targets.
5. Unary and binary operations retain exact owner and result types. The right-shift marker, checked
   switch, and continuation binary operation retain the exact common type.
   `V1001` rejects an invalid typed binary shape and `V1501` rejects missing, mismatched, or forged
   check metadata before interpretation or C emission.
6. The interpreter checks the shift amount before computing the result. Generated C emits the same
   width check before any C shift; unsigned continuation operations use their exact unsigned C
   representation, while `i32` retains the bit-preserving arithmetic-shift helper.
7. At this decision point, signed and unsigned left shift remained rejected as `E1011`; ADR 0103
   later replaces that restriction with checked matching-type left shift. Rotates, wrapping shifts,
   mixed-type operands, C-scalar bitwise operations, and wider integer families remain deferred.
8. The interpreter fixture contract grows to 72 cases and the C-emission contract to 26 cases. The
   derived Linux matrix therefore plans 104 low/optimized compiler executions and 52 sanitizer
   executions. The local Windows smoke executes unsigned success and invalid-width cases under both
   MSVC profiles, bringing its direct `run-host` outcome count to 28 without WSL.

## Consequences

- Byte masks, complements, flags, hashes, and size calculations can stay in their natural unsigned type.
- Interpreter and C backend observe one explicit shift-width rule instead of inheriting host behavior.
- Right shift remains potentially trapping even for unsigned operands; bitwise `~`, `&`, `^`, and `|` do not.

## References

- `docs/adr/0023-explicit-checked-arithmetic-mir.md`
- `docs/adr/0100-checked-unsigned-arithmetic.md`
- `docs/adr/0101-checked-unsigned-division.md`
- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
