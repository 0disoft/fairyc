# ADR 0092: Borrowed view byte length

- Status: Accepted
- Date: 2026-07-20
- Language surface: `(&str).len`, `(&[u8]).len`

## Context

Compiler-owned UTF-8 and byte-string literals already carry an exact byte length through typed HIR,
verified MIR, the interpreter, and private generated-C view records. Source code could expose the
data address to narrow unsafe FFI, but it could not read that existing length. Callers therefore had
to duplicate a literal's byte count by hand, which is especially error-prone for UTF-8 and makes a
safe output wrapper impossible to express without another privileged constant.

## Decision

1. A safe borrow whose element type is `str` or `[u8]` exposes the read-only property `.len`.
   The result type is `usize`.
2. The value is the number of bytes in the borrowed view. It is not a Unicode scalar-value count,
   grapheme count, code-point count, capacity, or terminator-inclusive C-string length.
3. Reading `.len` copies existing view metadata. It allocates no storage, calls no function, performs
   no scan, cannot trap, and does not change the borrow origin or mutability state.
4. Typed HIR lowers the property to a distinct `view-length` operation. Verified MIR records the
   exact borrowed owner type, `len` field identity, one matching view operand, and a `usize`
   destination. A malformed family, result, operand, or property is `V1001`.
5. The interpreter reads the byte count retained by its private view value. Generated C reads the
   `length` member of the existing private `fy_str` or `fy_byte_slice` transport. Those records and
   member names remain compiler-private and are not a public layout or foreign ABI.
6. `.len` is admitted for static literals and local borrowed views. The Windows output example uses
   it for both the requested byte count and progress comparison instead of duplicating `16`.
7. No `.capacity`, indexing, slicing, iteration, mutation, `CStr.len`, Unicode counting, public
   string ABI, or safe/portable I/O API is added by this decision.

## Consequences

- UTF-8 text such as `"λx"` reports 3, while `b"\x00ABC"` reports 4.
- Length-bearing raw calls can now use source-derived bounds without a manually synchronized
  integer constant.
- A later indexing or slicing design must define bounds and UTF-8 boundary rules separately; this
  property does not prejudge either surface.

## References

- `docs/adr/0066-compiler-owned-byte-string-slice-storage.md`
- `docs/adr/0067-compiler-owned-utf8-string-static-storage.md`
- `docs/adr/0091-windows-stdout-exec-target.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
