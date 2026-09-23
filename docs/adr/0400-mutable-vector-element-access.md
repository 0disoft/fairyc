# ADR 0400: Mutable vector element access

- Status: Accepted
- Date: 2026-09-23
- Depends on: ADRs 0388–0394

## Decision

SDK `0.0.0-stage0.63` adds `std::vec::at_mut<T>(ref mut Vec<T>, usize) -> ref mut T` on the existing Windows vector path. The language revision remains `fairyc-language/17`; no user-defined generic aggregate or new runtime ABI is admitted. The exact SDK-owned source is the only caller of the private `storage_borrow_mut` operation.

The function checks the index, obtains an initialized mutable slot from an exact private C ABI, checks the non-null slot and returns a reference tied to the exclusive vector owner. It neither allocates, copies nor changes element ownership. An out-of-range index traps with `assertion-failed`. The owner cannot be moved, resized or borrowed again while the returned reference is live. Direct scalar field assignment on a named `ref mut` element is supported; general scalar dereference assignment, nested resources and iteration remain separate.

The C backend accepts the new mutable slot symbol only with `ptr mut void (ptr mut void, usize, usize)` on the supported vector targets and only through authenticated private storage emission. Ordinary C emission still rejects private storage operations and runtime symbols. The runtime validates the descriptor, element size and initialized index before returning a pointer. Existing immutable `at` retains its shared signature and behavior.

## Evidence and compatibility

`TestWindowsVectorMutableElement` runs a real vector of user records in low and optimized MSVC profiles, writes through the returned reference and reads the updated value. It checks alias and owner-move rejection while the mutable reference is live and the out-of-range trap. A separate field-store regression covers the case where a reference's last use is a field assignment. Backend and C runtime tests cover the exact private signature and unchanged vector execution. The SDK source digest and lock identity change; existing workspaces refresh their locks when moving to `0.0.0-stage0.63`.
