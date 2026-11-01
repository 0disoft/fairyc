# ADR 0489: Exact C-scalar raw pointers and output slots

- Status: Accepted for prototype
- Date: 2026-11-01
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0047 raw scalar pointers, ADR 0484 exact C-scalar arrays and ADR 0488 record fields
- Supersedes: only ADR 0488's blanket closure of new pointer forms, for the exact subset below

## Context

Foreign interfaces commonly pass a scalar by address, receive one through an
output pointer, or fill a `T **` slot. The existing executable pointer slice
admits Fairy `u8`, `i32`, `i64`, `u32` and `usize`, opaque `void`, and admitted
`@repr(C)` records. Exact `c::*` scalars stop at by-value transport, so a caller
must substitute a representation-equal Fairy scalar or erase the type to
`void *`. Both choices lose the provider's scalar identity, target width,
signedness or availability.

ADR 0488 admitted record-element arrays and kept new pointer forms closed for
that slice. Its record-array proof and closure of other pointer families remain
intact. This decision opens only the exact C-scalar forms below.

## Decision

For each exact `c::T` among the sixteen existing `CScalarType` identities that
the selected target supplies, admit:

- `ptr c::T`, emitted as `const <C T> *`;
- `ptr mut c::T`, emitted as `<C T> *`.

Windows MSVC admits fifteen of the sixteen identities because it has no
`c::ssize_t`. `c::long` uses the target's layout: four bytes under that LLP64
profile and eight bytes under the LP64 Linux profile. A POSIX
`ptr c::ssize_t` keeps the existing `<sys/types.h>` provider requirement.
Fairy scalars and representation-equal C scalars remain distinct pointer
targets; no conversion between `ptr c::int` and `ptr i32` is introduced.

The only new nested pointer form is the mutable output slot
`ptr mut ptr mut c::T`, emitted as `<C T> * *`. The existing exact
`ptr mut ptr mut void` -> `void * *` slot remains unchanged. Readonly and mixed
nested forms such as `ptr ptr c::T`, `ptr ptr mut c::T` and
`ptr mut ptr c::T` remain closed, as do depth-three pointers and nested
pointers to records, arrays or Fairy scalars. Existing single-level pointer
admissions are unchanged.

Admitted values support typed null, pointer-value copy, internal and foreign
arguments and results, `@repr(C)` fields, and equality only between operands of
the same exact pointer type, including that type's typed null. Mutability and
nominal scalar identity are part of that type. No cross-type comparison or
implicit conversion is added.

Inside an explicit unsafe boundary, the existing same-element address casts
apply to these C-scalar pointees: `ref c::T -> ptr c::T`,
`ref mut c::T -> ptr c::T`, and `ref mut c::T -> ptr mut c::T`, for direct
local places and borrowed C-scalar parameters. One new mutable pointer-slot
borrow is admitted: `ref mut ptr mut c::T -> ptr mut ptr mut c::T`. No readonly
nested borrow variant is admitted, including `ref ptr mut c::T` or
`ref mut ptr c::T`.

A borrow or local-address cast lends the existing place or slot only for a
synchronous C call. It does not extend the storage lifetime, transfer
ownership, pin the storage, or guarantee that C retains the pointer. If a
provider writes or returns a pointer, the provider's ownership and lifetime
contract determines its validity; the compiler adds no retention or escape
guarantee for it.

Raw dereference, pointer arithmetic, pointer retyping, integer-pointer
conversion, mutability strengthening, safe-reference construction from a raw
pointer, and C-scalar literals or arithmetic remain closed. No pointer array or
pointer-based array transport is added. The interpreter remains fail-closed on
foreign calls.

The C ABI and the admitted language subset extend. Source grammar, MIR opcodes
and codec format, runtime, SDK surface, serialized formats, dependencies, CLI
and CI configuration do not change.

## Evidence

Implementation and core compiler/interface/package tests are local commit
`3ea7a217`. The native probe and Windows receipt are in the commit containing
`docs/ops/windows-c-scalar-pointers-20261101.md`.

- `go test -mod=readonly ./internal/compiler -run '^TestCScalarPointer' -count=1`
  passes. All sixteen exact `CScalarType` identities reach verified MIR and
  survive MIR codec round trips; Windows rejects `c::ssize_t`; a local-only
  POSIX `ptr c::ssize_t` keeps `<sys/types.h>` and its size assertion; ten
  unsafe/type negatives reject; forged raw-address and raw-borrow MIR
  mutations for provenance, pointee and strengthening reject.
- `go test -mod=readonly ./internal/cbackend -count=1` passes in 30.217s.
- `go test -mod=readonly ./internal/moduleinterface ./internal/packageartifact -count=1`
  passes. Exact target, qualifier and depth identities remain distinct; all
  sixteen times three pointer shapes survive context-descriptor
  encode/decode/rebind.
- `go test -mod=readonly ./internal/foreignstructprobe -run '^TestCScalarPointers' -count=1 -v`
  passes in 3.770s. Emission inventories are 15 Windows and 16 POSIX scalar
  targets, each with const, mutable and `T **` shapes.
- Scoped `go vet -mod=readonly ./internal/cbackend ./internal/compiler
  ./internal/moduleinterface ./internal/packageartifact ./internal/foreignstructprobe`
  exits 0.
- The broader compiler filter `(CScalarPointer|RawPointer|RawAddress|Opaque|Borrow.*Pointer|IntegerTransportEmissionBoundaries)`
  also passes in 0.517s. The guarded `TestCScalarPointersLinux` compiles and
  skips under Windows.
- MSVC 14.51.36231 passes C11/C17 at low/optimized: four executions, 15
  scalar types times 3 value rotations = 45 cases each. Each case covers all
  three shapes, null round trips, C write-through of a borrowed scalar, slot
  fill/clear/refill, `@repr(C)` pointer fields and exact pointer identity.
  Separate caller/callee objects, final PE and AMD64 unwind inspection, exit
  42 and empty streams pass.

The fixture-only C bridge calls actual generated Fairy identity functions for
the const, mutable and `T **` shapes. It adds no public export or callback
surface.

The [Windows receipt](../ops/windows-c-scalar-pointers-20261101.md) records the
commands and boundaries. Native Linux execution, a full compiler-package run, a
full `go test ./...`, hosted CI, push and publication remain unexecuted or
deferred.
