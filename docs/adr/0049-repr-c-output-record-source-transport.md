# Source `@repr(C)` Output Record Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- ABI source: `docs/abi/c-abi-v0.md`
- MIR source: `docs/mir/mir-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0048 proved that generated Fairy C can call the actual text-view C implementation, but only
with typed null. The output-producing validator needs a layout-compatible record and the address of
live Fairy storage. Treating every cast as an integer or pointer conversion would open a large
undefined-behavior surface before provenance, alignment, and alias rules exist.

## Decision

1. Item attributes are preserved through AST and typed HIR. The current executable `@repr(C)`
   subset in this slice is a nonempty flat inert struct over `u8`, `i32`, `u32`, `usize`, and raw
   pointers. ADR 0057 later adds exact TargetSpec C-scalar fields and opaque `void` pointers.
2. MIR owns a separate inert struct definition and `struct-construct` operation. It never borrows
   resource cleanup semantics. Field order and source names remain explicit.
3. The only source-to-raw-address cast is an exact direct-local conversion from `&T` to `*T`,
   `&mut T` to `*T`, or `&mut T` to `*mut T` inside `unsafe`. Pointee changes and mutability
   strengthening are rejected. Dereference, arithmetic, integer conversion, and arbitrary pointer
   casts remain unsupported.
4. MIR `raw-address` records the place, pointee, pointer type, borrow mutability, source span, and
   containing unsafe scope. The verifier requires exact provenance. The C backend emits `&local`
   without `restrict` or `noalias`; the interpreter fails closed at the native-only operation.
5. Public `@repr(C)` tags and field names are emitted verbatim after C11 identifier checks. Target-
   computed `sizeof`, `_Alignof`, and every `offsetof` are asserted in generated C.
6. The closed `fy_std_` allowlist admits `fy_std_text_validate_utf8_v0` in addition to the existing
   predicate. It does not open the prefix generally.
7. A generated Fairy caller initializes a sentinel output record, passes its raw address to the
   actual `TextViewSource()` unit, and returns 42 only after status success and observed field
   mutation. Clang/GCC low/optimized execution remains owned by the Linux native lane.
8. `fairyc.native-c-report/v15` adds four `textViewOutputSourceCallExecutions` without rewriting
   the meaning of the v14 predicate-call counter.

## Consequences

- A `.fy` program can now provide non-null local output storage to a real project-owned C helper.
- This does not create a public safe text API, a general pointer-cast facility, by-value aggregate
  ABI, nested C layout, or nonempty input byte storage.
- Linux execution evidence remains configured but unobserved on the Windows development host.
