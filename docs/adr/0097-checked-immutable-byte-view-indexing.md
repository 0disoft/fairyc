# ADR 0097: Checked immutable byte-view indexing

- Status: Accepted
- Date: 2026-07-20
- Language surface: `(&[u8])[usize] -> u8`

## Context

Stage 0 already parses postfix index syntax, carries compiler-owned immutable byte storage through
typed HIR and verified MIR, and exposes exact byte lengths through `.len`. Source programs could
therefore pass a complete byte view to a helper but could not read even one byte without converting
the view to a raw pointer inside `unsafe`. General indexing would prematurely force decisions for
mutable slices, arrays, text boundaries, places, assignment, iteration, and unchecked operations.

## Decision

1. Only an immutable safe borrow of `[u8]` supports read-only indexing in this slice. The receiver is
   `&[u8]`, the index is `usize`, and the result is `u8`.
2. An unsuffixed integer literal in the index position receives contextual type `usize`. Other index
   types are rejected as `E1006`; `str`, `CStr`, arrays, mutable views, and non-view receivers remain
   unsupported.
3. Receiver and index evaluate exactly once from left to right. Lowering reuses the same evaluated
   operands for both the check and load.
4. MIR emits `view-index-check`, a two-way `checked-view-index` switch, a non-returning `bounds` trap
   block, and a continuation containing `view-index`. `V1501` rejects a missing, incomplete, or
   non-dominating check; `V1001` rejects malformed owner, operand, operator, destination, or result
   facts.
5. The interpreter retains exact bytes in its private view value, reports `bounds` before an invalid
   read, and loads one byte after the checked edge.
6. Generated C compares the `uintptr_t` index with the private descriptor length, routes failure to
   `FY_TRAP_BOUNDS`, and emits `data[index]` only in the checked continuation. The private descriptor
   remains neither a public layout nor a foreign ABI.

## Consequences

- A Fairy C program can read compiler-owned byte-string data without entering `unsafe`.
- Interpreter, deterministic C-emission, and Windows MSVC acceptance paths observe byte value `65`
  from `b"\x00A"[1]`; the Windows output example additionally observes the leading NUL byte.
- Empty views and an index equal to the byte length trap deterministically without cleanup.
- Bounds-check elimination is not implemented; every admitted index currently keeps the explicit
  dynamic check even when a future optimizer could prove it safe.
- Slicing, indexed assignment, mutable views, arrays, `str` indexing, Unicode boundary policy,
  iteration, and unchecked indexing remain deferred decisions.

## References

- `docs/adr/0066-compiler-owned-byte-string-slice-storage.md`
- `docs/adr/0092-borrowed-view-byte-length.md`
- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
