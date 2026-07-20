# ADR 0100: Checked unsigned arithmetic

- Status: Accepted
- Date: 2026-07-20
- Language surface: matching-type `+`, `-`, and `*` for `u8`, `u32`, and `usize`

## Context

Stage 0 ordinary arithmetic already trapped on unrepresentable `i32` results, but `u8`, `u32`,
and `usize` values could only be compared or transported. That made a basic byte-view loop stall at
`index = index + 1` even though indexing, length reads, comparison, and widening the loaded byte were
already implemented.

Using host C unsigned wrap as the source meaning would be a trap: it would make ordinary Fairy C
arithmetic silently change value while the signed path traps. It would also let interpreter and C
backend behavior drift at exactly the integer boundary where optimization bugs are hardest to spot.

## Decision

1. Ordinary binary `+`, `-`, and `*` admit exactly two matching operands of `i32`, `u8`, `u32`, or
   `usize` and return that same type. No implicit promotion, widening, or signedness conversion occurs.
2. An addition or multiplication result above the type maximum, or a subtraction result below zero,
   traps as `integer-overflow`. Ordinary arithmetic never wraps and is not controlled by build profile.
3. The current admitted targets define `usize` as an unsigned 64-bit value. Its checked arithmetic
   therefore uses the exact range `0...18446744073709551615`; a future target with a different pointer
   width must supply its own target fact before this source surface can execute there.
4. MIR `binary` and `checked-arithmetic` records carry the exact common operand/result type, and the
   `checked-arithmetic` terminator carries the same `SwitchType`. `V1001` rejects an invalid binary
   shape; `V1501` rejects a missing or type-forged dominating check edge.
5. The interpreter performs explicit range checks before constructing the typed result. The C backend
   emits pre-operation checks against `UINT8_MAX`, `UINT32_MAX`, or `UINTPTR_MAX`; it never depends on
   wrapped unsigned output to decide whether a trap should have happened.
6. Division, remainder, shifts, and bitwise operators are unchanged by this decision; ADRs 0101 and
   0102 admit their matching-type subsets later. Wrapping, saturating, explicit checked-result APIs,
   broader literals, conversions, and mixed-type arithmetic remain unchanged.
7. `examples/windows_quickstart/main.fy` iterates a two-byte `&[u8]` with a `usize` counter and returns
   their sum after value-preserving `u8 as i32` conversions. The Windows host smoke executes that path
   and a maximum-`usize` addition trap under both low and optimized MSVC profiles without WSL.

## Consequences

- A normal source loop can now traverse an immutable byte view without raw pointers or a host helper.
- Overflow behavior is identical in the interpreter and generated C instead of inheriting C wrapping.
- Unsigned division and remainder are admitted later by ADR 0101, and matching-type bitwise
  operations plus checked right shift by ADR 0102. Mixed arithmetic and wider integer families
  remain deliberate future slices.

## References

- `docs/adr/0023-explicit-checked-arithmetic-mir.md`
- `docs/adr/0072-defined-checked-arithmetic-c-construction.md`
- `docs/adr/0097-checked-immutable-byte-view-indexing.md`
- `docs/adr/0098-value-preserving-integer-casts.md`
- `docs/adr/0099-typed-scalar-comparisons.md`
- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
