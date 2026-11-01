# Windows exact C-scalar pointer and output-slot receipt, 2026-11-01

- Source implementation and core tests: `3ea7a217` (local commit).
- Native probe and this receipt: the commit containing this file.
- Host: native Windows amd64, MSVC 14.51.36231.
- Contract owners: [ADR 0489](../adr/0489-c-scalar-pointer-output-slots.md) and
  [C ABI v0](../abi/c-abi-v0.md#exact-c-scalar-raw-pointers).
- Current scope: item 14's exact C-scalar pointer admission. The old `void **` slot and every
  earlier pointer admission are unchanged. ADR 0488's record-array proof remains historical.

## Final executed evidence

| Check | Final observed result |
| --- | --- |
| Compiler focused tests | `go test -mod=readonly ./internal/compiler -run '^TestCScalarPointer' -count=1` passes. All sixteen exact C scalar identities reach verified MIR and MIR codec round trips; Windows rejects `c::ssize_t`; a local-only POSIX `ptr c::ssize_t` keeps the `<sys/types.h>` provider and size assertion; ten unsafe/type negatives reject. |
| Broader compiler filter | `(CScalarPointer|RawPointer|RawAddress|Opaque|Borrow.*Pointer|IntegerTransportEmissionBoundaries)` passes in 0.517s. |
| Forged MIR | Raw-address and raw-borrow mutations for provenance, pointee and mutability strengthening reject at verification and C emission. |
| C backend | `go test -mod=readonly ./internal/cbackend -count=1` passes in 30.217s. |
| Interface and cache identity | `go test -mod=readonly ./internal/moduleinterface ./internal/packageartifact -count=1` passes. Exact target, qualifier and depth identities remain distinct; all 16 x 3 pointer shapes survive context-descriptor encode/decode/rebind. |
| Probe package | `go test -mod=readonly ./internal/foreignstructprobe -run '^TestCScalarPointers' -count=1 -v` passes in 3.770s. |
| Scoped vet | `go vet -mod=readonly ./internal/cbackend ./internal/compiler ./internal/moduleinterface ./internal/packageartifact ./internal/foreignstructprobe` exits 0. |
| Emission inventory | 15 Windows and 16 POSIX scalar targets, each with exact `const C *`, mutable `C *` and `C **` shapes. |
| Windows native matrix | Four passes: MSVC C11/C17 at low/optimized. Each execution covers 15 scalar types x 3 value rotations = 45 cases, with all three shapes per case and null round trips. |
| Pointer behavior | C write-through of a borrowed scalar is read back by Fairy; slot fill, null clear and borrowed refill pass; `@repr(C)` pointer fields and exact pointer identity pass. |
| Object and image inspection | Separate `caller.obj` and `callee.obj` links pass final PE and AMD64 unwind inspection. Every execution exits 42 with empty stdout and stderr. |

The native fixture builds a standalone `GenerateCScalarPointers` unit and binds fixture-only C
bridges to actual generated Fairy identity functions for `const`, mutable and `T **` shapes.
It is not a public export or a callback surface.

## Reproduction

From the repository root on Windows:

```powershell
go test -mod=readonly ./internal/compiler -run '^TestCScalarPointer' -count=1
go test -mod=readonly ./internal/compiler -run '(CScalarPointer|RawPointer|RawAddress|Opaque|Borrow.*Pointer|IntegerTransportEmissionBoundaries)' -count=1
go test -mod=readonly ./internal/cbackend -count=1
go test -mod=readonly ./internal/moduleinterface ./internal/packageartifact -count=1
go test -mod=readonly ./internal/foreignstructprobe -run '^TestCScalarPointers' -count=1 -v
go vet -mod=readonly ./internal/cbackend ./internal/compiler ./internal/moduleinterface ./internal/packageartifact ./internal/foreignstructprobe
git diff --check
```

## Boundaries and omissions

Native Linux execution remains deferred. The guarded `TestCScalarPointersLinux` Clang/GCC
cross-helper compiles and skips correctly under Windows; its native run has not happened. The
16-target POSIX inventory is emission-only evidence on Windows.

Scoped `go vet` passes with exit 0. A full compiler-package run, full `go test ./...`, hosted
CI, push and publication were not run.

The language and C ABI subset extend. Source grammar, MIR opcodes and codec format, runtime, SDK,
serialized formats, dependencies, CLI and CI configuration are unchanged. No public export,
callback signature, pointer dereference, pointer arithmetic, pointer retyping, integer-pointer
conversion, mutability strengthening, safe reference from a raw pointer or new C-scalar
literal/arithmetic is admitted. Other qualifiers, depth-three pointers and nested record, array or
Fairy-scalar pointers remain closed. A C-produced output pointer's validity is the provider's
obligation.
