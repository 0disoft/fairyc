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

The executable MIR opcode is `c-callback`. It names one internal function in `Symbol` and
produces a dedicated local consumed only as the designated foreign argument in that block.
The address is materialized after user-data evaluation because naming a static function has
no side effect. This also permits a nested user-data producer call without extending the callback
temporary across calls. Signature/origin, unique use and overwrite checks reject forged MIR.

Stable package bodies encode the target FunctionID in the new opcode's existing `Variant` field,
with `Symbol` cleared. Rebinding resolves it to a fresh symbol and clears `Variant`. Both process
and stable artifacts include callback targets in their dependency sets. The codec envelope and
all existing opcode wire names remain unchanged; old consumers reject the new opcode, and
compiler-build identities keep incompatible caches separate.

Local delivery passes type/semantic/interface, MIR/codec/interpreter, package-artifact,
C-backend and emission-fixture tests. Focused compiler tests cover both target emission profiles
and reject forged callback targets, null substitution, duplicate use and overwrite. Stable
package rebinding preserves bytes after fresh package/function/local numbering; unresolved
targets are rejected. Memory and independent-session disk hits retain a cross-package callback
target without lowering again.

The shared native unit executes six callback invocations per C11/C17 low/optimized combination
under MSVC 14.51.36231: pure handling with null data, updates to a Fairy stack counter through
an independent C helper, and nested C data production. All four combinations pass, alongside
the previous aggregate/array/enum/float/integer cases. Linux-profile C emission and affected-package
vet pass. Hosted GCC/Clang execution awaits an authorized batched push; the full repository suite
and full Windows smoke were not repeated. No SDK/CLI API, DB, workflow or runner changes.
