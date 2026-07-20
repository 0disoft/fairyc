# ADR 0099: Typed scalar comparisons

- Status: Accepted
- Date: 2026-07-20
- Language surface: matching-type `==`, `!=`, `<`, `<=`, `>`, and `>=`

## Context

Stage 0 already parses comparison operators and generated C can compare exact scalar operands.
Semantic typing and the MIR interpreter, however, admitted only `i32` ordering while some native
Windows source paths already depended on `u32` and `usize` equality. That split prevented ordinary
safe checks such as `bytes.len > 0` from having one executable meaning across both backends.

## Decision

1. `==` and `!=` admit exactly two matching operands of `bool`, `i32`, `u8`, `u32`, or `usize` and
   return `bool`.
2. `<`, `<=`, `>`, and `>=` admit exactly two matching operands of `i32`, `u8`, `u32`, or `usize`
   and return `bool`.
3. Equality also preserves the pre-existing native boundary for two matching raw-pointer types.
   This permits null and pointer-identity checks but not dereference, arithmetic, ordering, or mixed
   pointee/mutability types. The interpreter executes only the typed-null pair; non-null pointer
   identity remains a native backend capability.
4. Comparisons never promote, widen, sign-change, reinterpret, or mix operand types. Aggregate,
   borrow, C-scalar, floating, pointer ordering, and boolean ordering remain unsupported.
5. MIR `binary` records the common operand type in `OwnerType` and result type in `Type`. `V1001`
   rejects a forged pair, invalid immediate family, missing destination, or unsupported operator.
6. The interpreter compares signed `i32` mathematically and the unsigned `u8`, `u32`, and `usize`
   domains as exact unsigned values. The C backend independently checks both operand types and emits
   a matching fixed-width or `uintptr_t` comparison normalized to `INT32_C(0)` or `INT32_C(1)`.
7. `examples/windows_quickstart/main.fy` guards its checked byte load with `bytes.len > 0`, so both
   low and optimized MSVC execution prove the `usize` ordering path before returning byte value 42.

## Consequences

- User programs can make natural length and unsigned-state decisions without casts to `i32`.
- Existing native-only `u32` and `usize` equality now has matching interpreter evidence.
- General numeric arithmetic, implicit promotion, aggregate equality, and pointer ordering remain
  separate design decisions.

## References

- `docs/adr/0097-checked-immutable-byte-view-indexing.md`
- `docs/adr/0098-value-preserving-integer-casts.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
