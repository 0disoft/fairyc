# ADR 0104: Borrowed-view emptiness metadata

- Status: Accepted
- Date: 2026-07-20
- Language surface: `(&str).is_empty` and `(&[u8]).is_empty`

## Context

Borrowed `str` and `[u8]` views already carry an exact byte length and expose it through `.len`.
Testing emptiness as `.len == 0` is correct but noisy in guards and makes a basic metadata query look
like user arithmetic. The property can be admitted without scanning data, calling a runtime helper,
or changing the private view representation.

## Decision

1. A shared or mutable safe borrow whose element is `str` or `[u8]` exposes the read-only property
   `.is_empty` with type `bool`.
2. The result is true exactly when the existing view byte length is zero. UTF-8 scalar values,
   graphemes, capacity, pointer identity, and terminators are irrelevant.
3. The property evaluates its receiver once, performs no allocation, scan, call, trap, unsafe
   operation, or ownership transition, and preserves the receiver's borrow origin.
4. Typed HIR uses a distinct `view-empty` expression and MIR uses a distinct `view-empty`
   operation with one exact borrowed-view operand and one `bool` destination. The verifier rejects
   forged owner, field, result, operand, or destination metadata as `V1001`.
5. The interpreter compares retained length metadata with zero. C emission compares the private
   `uintptr_t` length field with zero and normalizes the result to `INT32_C(0)` or `INT32_C(1)`.
6. `CStr.is_empty`, capacity queries, general collection protocols, slicing, and Unicode counting
   remain separate decisions.
7. The interpreter fixture contract grows to 78 cases. C emission stays at 27 cases by extending
   the existing view-metadata case, so the configured Linux 108/54 matrix counts do not change.
   The checked-in Windows quickstart executes empty and nonempty byte-view queries in its existing
   low/optimized host-smoke outcomes without WSL.

## Consequences

- Common empty-view guards become direct and backend-independent.
- The property adds no runtime or ABI surface and is as cheap as reading `.len`.
- `.is_empty` is not an implicit truthiness conversion; views remain invalid boolean conditions.

## References

- `docs/adr/0092-borrowed-view-byte-length.md`
- `docs/adr/0097-checked-immutable-byte-view-indexing.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
- `docs/runtime/runtime-stdlib-v0.md`
