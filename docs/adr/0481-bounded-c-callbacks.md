# ADR 0481: Bounded C callbacks

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md#functions-and-callbacks)

## Decision

The first source callback signature is `unsafe extern C fn i32(i32, ptr mut void)`.
It may appear only as the penultimate parameter of an unsafe C import, followed by a final
`ptr mut void` user-data parameter. A call supplies the name of an ordinary, non-capturing
`unsafe fn i32(i32, ptr mut void)` with no declared effects. The compiler adapts that function
to the exact C function-pointer signature; ordinary Fairy symbol names and calling conventions
remain private. No public C export is created.

The external declaration promises synchronous, same-thread invocation before the import
returns. C must not retain either the callback or call-bounded user data, dispatch asynchronously,
unwind, or `longjmp` across Fairy frames. These are obligations of the unsafe integration, not
properties the compiler can establish from a C declaration. Raw user-data validity remains the
caller's responsibility; existing borrow-to-pointer checks still apply. Null user data is allowed
when the callback does not dereference it. Callback pointers themselves are non-null.

Only a named function in this direct call position is admitted. Callback locals, fields, arrays,
return values, casts, indirect calls, safe-borrow captures, nullable callbacks, other signatures
and targets with declared effects remain closed. The pure interpreter rejects foreign calls.
Target maturity flags and Phase 2 status remain unchanged.

## Delivery

Implement in small commits: source typing and contract; verified MIR, package preservation and C
transport; native crossing evidence. Verify target rebinding and reject callback values that
escape the designated foreign argument. Reuse existing native runners without new CI jobs.
