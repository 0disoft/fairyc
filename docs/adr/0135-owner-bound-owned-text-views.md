# ADR 0135: Add owner-bound borrowed views for example owned text

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Borrow source: `docs/language/0002-borrows-and-return-origins.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`

## Context

The example-level `String` and `CString` resources already own validated storage and deterministic
drop, but their safe module could expose only scalar lengths. Reconstructing `&str` or `&CStr` by
copying a raw pointer into a private C descriptor would not prove which Fairy owner keeps that
storage alive. Treating a raw pointer alone as a safe borrow would therefore manufacture lifetime
provenance.

## Decision

1. The compiler recognizes exactly two unsafe built-in calls:
   `str::from_raw_parts(owner, data, length) -> &str` and
   `CStr::from_raw_parts(owner, data, storage_length) -> &CStr`.
2. `owner` must be a shared safe borrow and is the sole origin of the returned view. `data` must be
   a raw `u8` or opaque `void` pointer, and the third argument must be `usize`. The built-ins do not
   infer provenance from `data`.
3. The unsafe caller proves that the pointer remains within the owner, the byte extent is valid,
   `str` bytes are UTF-8, and `CStr` storage is nonempty, has no interior NUL, and ends in one NUL.
   Safe wrappers may expose the result only when their private representation establishes those
   invariants.
4. MIR uses distinct `str-from-raw-parts` and `cstr-from-raw-parts` operations. Each operation
   retains its source span, containing unsafe scope, owner borrow type, result type, and three
   already-evaluated operands. The verifier rejects missing or forged unsafe provenance.
5. Generated C constructs the private `fy_str` value without allocation or copying. A zero-length
   String whose owned data pointer is null receives a private non-null empty sentinel. `&CStr`
   remains the existing immutable byte pointer transport; its storage-length operand is still
   evaluated before construction.
6. Moving an owner as a by-value resource argument while a retained derived view has a later use is
   `E2105`. The diagnostic identifies the move, owner declaration, loan, and last use.
7. `examples/owned_text/owned_text.fy` encapsulates both unsafe built-ins behind safe
   `string_as_str` and `cstring_as_cstr` wrappers. This does not stabilize `std::text`, either
   resource layout, the private C descriptor, a foreign safe-borrow ABI, or a general user-defined
   raw-parts protocol.

## Consequences

- Safe callers can observe owned String bytes through existing `&str` metadata operations and pass
  owned CString storage through internal `&CStr` calls without receiving a raw pointer.
- No new allocation, scan, copy, terminator search, or public ABI is introduced.
- Unsafe representation code remains confined to the example support module. A user-defined
  resource cannot opt into safe view reconstruction without the exact compiler-known built-in and
  an explicit owner token.
- The interpreter remains closed over these unsafe pointer-derived operations. C emission and
  native execution provide the executable evidence for this slice.

## Verification

- semantic tests and fixtures prove exact `Param(0)` propagation, unsafe-scope rejection, and
  `E2105` owner-move rejection;
- MIR tests require both operation shape and matching unsafe provenance;
- C-backend tests require the private UTF-8 descriptor, empty sentinel, and CStr pointer transport;
- the WSL-free Windows host smoke compiles and executes the checked-in safe owned-text example with
  local x64 MSVC at low and optimized profiles and returns 42.
