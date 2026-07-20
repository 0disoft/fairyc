# ADR 0098: Value-preserving integer casts

- Status: Accepted
- Date: 2026-07-20
- Language surface: `u8 as i32|u32|usize`, `u32 as usize`

## Context

The executable Stage 0 subset already carries `u8`, `u32`, `usize`, and `i32` values, and checked
byte indexing produces `u8`. It could compare that byte but could not return it from `main() -> i32`
or deliberately widen it for another typed API. The surface grammar already has `as`, while the
typing contract says that `as` is limited to value-preserving conversions and narrowing needs a
fallible operation. The concrete safe integer pairs were not yet fixed or executable.

## Decision

1. Safe integer `as` admits exactly `u8 -> i32`, `u8 -> u32`, `u8 -> usize`, and `u32 -> usize`.
   Every mathematical source value is representable by its target on every admitted Stage 0 target.
2. No identity, narrowing, signedness-changing, pointer/integer, float, C-scalar, or other numeric
   pair is admitted by this slice. Rejected pairs do not truncate, wrap, reinterpret, or trap; they
   remain unsupported until a separate explicit conversion contract exists.
3. Typed HIR uses `scalar-cast` with exact source and target types. The operation is safe and carries
   no unsafe scope.
4. MIR retains one source operand, its exact owner type, the exact result type, the `as` operator,
   and one destination. `V1001` rejects a forged pair, malformed immediate family, extra operand, or
   missing destination.
5. The interpreter constructs the exact target runtime value. The C backend emits one explicit cast
   to `int32_t`, `uint32_t`, or `uintptr_t`; the admitted ranges avoid undefined, implementation-
   defined, or lossy conversion behavior.
6. `examples/windows_quickstart/main.fy` returns `b"*"[0] as i32`, so the separately built Go Stage 0
   compiler must compile an indexed Fairy byte through MSVC and observe native process result 42.

## Consequences

- Indexed bytes can participate directly in an `i32` program result without unsafe code.
- The conversion set is intentionally small but mechanically shared by semantic typing, MIR
  lowering and verification, the interpreter, and C backend inspection.
- Future fallible narrowing should return an explicit typed result rather than silently changing
  `as` semantics.

## References

- `docs/adr/0097-checked-immutable-byte-view-indexing.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
