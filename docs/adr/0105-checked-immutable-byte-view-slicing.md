# ADR 0105: Checked immutable byte-view slicing

- Status: Accepted
- Date: 2026-07-20
- Language surface: `(&[u8])[start..end]`

## Context

Immutable byte views already carry private non-null data and exact byte-length metadata. Read-only
indexing proves one offset before loading one byte, but callers still cannot derive a smaller safe
view without exposing a raw pointer. A narrow end-exclusive slicing rule can reuse the same borrow
origin and representation while keeping allocation, ownership, and text-boundary questions out of
this decision.

## Decision

1. An immutable `&[u8]` admits `bytes[start..end]`. Both bounds are mandatory `usize` expressions,
   the end is exclusive, and the result is another immutable `&[u8]`.
2. The receiver, start, and end evaluate exactly once from left to right. The operation is valid
   exactly when `start <= end && end <= bytes.len`; otherwise it traps with category `bounds`.
3. The result preserves the receiver's borrow origin. It creates no allocation, scan, call,
   ownership transition, mutable capability, or public ABI surface.
4. HIR uses `view-slice`. MIR emits a `view-slice-check` marker, a two-way
   `checked-view-slice` switch, a non-returning `bounds` trap block, and a checked continuation
   containing `view-slice`. All three carry the same already-evaluated operands in source order.
5. The verifier rejects malformed operand types, destination/result metadata, trap categories,
   reordered operands, or a missing dominating checked edge as `V1001` or `V1501`.
6. The interpreter checks both inequalities before deriving the retained byte range. Generated C
   checks both inequalities before pointer addition, then constructs the private view as
   `{ data + start, end - start }`. `bytes[len..len]` is valid and empty.
7. Omitted bounds, inclusive ranges, mutable slicing, indexed assignment, `str` slicing or
   indexing, general collection slicing, and unchecked access remain deferred.
8. The interpreter fixture contract grows from 78 to 82 cases and C emission from 27 to 28. The
   derived Linux matrix plans 112 compiler/profile executions and 56 sanitizer executions. This
   Windows host validates success and `bounds` trapping through local x64 MSVC without WSL.

## Consequences

- Safe code can derive zero-copy byte subviews without raw pointers.
- The check order prevents unsigned underflow in `end - start` and pointer formation outside the
  retained object.
- This decision does not solve Unicode code-point boundaries or stabilize the private slice layout.

## References

- `docs/adr/0066-compiler-owned-byte-string-slice-storage.md`
- `docs/adr/0097-checked-immutable-byte-view-indexing.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
- `docs/runtime/runtime-stdlib-v0.md`
