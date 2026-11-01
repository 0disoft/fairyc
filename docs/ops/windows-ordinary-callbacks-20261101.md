# Windows ordinary synchronous callback receipt, 2026-11-01

- Source implementation and core tests: `c45fc974` (local commit).
- Native probe and this receipt: the commit containing this file.
- Host: native Windows amd64, MSVC 14.51.36231.
- Owners: [ADR 0490](../adr/0490-ordinary-synchronous-callbacks.md),
  [C ABI v0](../abi/c-abi-v0.md#functions-and-callbacks).

## Executed evidence

| Check | Final observed result |
| --- | --- |
| Focused compiler/HIR callback tests | Pass: original callback, generalized signature, JSON/MIR codec round trips, both-target emission, partition prototypes, rejected signature/effect/qualifier/null/indirect/arity/aggregate forms and forged MIR target/type/escape mutations. |
| Semantic, MIR, interface and package artifacts | Full scoped package runs pass. Stable callbacks retain their exact expanded signature after fresh package/function/local numbering and stable bytes match. |
| Callback workspace cache | Process-memory and independent-session disk cold/warm checks pass with a multi-value callback from another package. |
| C backend | Full package passes (43.236s); the later new partition execution test passes separately (2.355s). |
| Windows native callback matrix | Four passes: MSVC C11/C17 x low/optimized; final matrix run 2.327s for the package. |
| Scalar observations | 15 exact Windows C-scalar identities and 12 Fairy scalar identities; three C-controlled values per family, including signed extrema, unsigned high bits, 64-bit values above 32-bit width, signed zero and finite fractional floats. C compares returned object representations. |
| Call shapes | 0-8 i32 value arguments plus typed user data, void return, an eight-value mixed double/float/int/i64 signature, const and mutable C-int pointers, exact T** output slot and pointer return. |
| Mutation and nested crossing | Every arity and mixed callback calls back into an independent C helper. Fairy observes the final counter; C verifies argument positions, returned values, pointer identity, slot publication and write-through. |
| Objects and images | Separate caller.obj/callee.obj, final PE and AMD64 unwind inspection pass; every image exits 42 with empty stdout/stderr. |
| Partition execution | Separate generated sources share one adapter. Repeated calls through different partitions retain callback pointer identity at low/optimized and both programs exit 42. |
| Vet, formatting and diff | Scoped go vet, gofmt and git diff --check pass. |

Each matrix image performs 112 actual C-to-Fairy callback invocations: 81 scalar,
27 arity, three mixed-argument and one pointer/slot invocation. The independent C
callee and generated Fairy caller are separate translation units; no public export,
C cast of a mismatched function pointer or substituted Fairy implementation is used.
These are source-level same-toolchain observations, not an independent register oracle.

An intermediate boundary-value probe omitted `<limits.h>` and failed C compilation.
The fixture header was corrected; all four final combinations above were rerun and pass.
The production callback implementation needed no change for that fixture issue.

## Reproduction

From the repository root on Windows:

```powershell
go test -mod=readonly ./internal/compiler ./internal/hir -run Callback -count=1
go test -mod=readonly ./internal/semantic ./internal/mir ./internal/moduleinterface ./internal/packageartifact ./internal/cbackend -count=1
go test -mod=readonly ./internal/workspacecompiler -run Callback -count=1
go test -mod=readonly ./internal/foreignstructprobe -run OrdinaryCallbacks -count=1 -v
go test -mod=readonly ./internal/cbackend -run '^TestPartitionedOrdinaryCallback$' -count=1 -v
go vet -mod=readonly ./internal/hir ./internal/semantic ./internal/mir ./internal/moduleinterface ./internal/packageartifact ./internal/cbackend ./internal/compiler ./internal/workspacecompiler ./internal/foreignstructprobe
git diff --check
```

## Omissions and remaining risk

TestOrdinaryCallbacksLinux compiles and skips under Windows. The POSIX source emits
28 scalar families including ssize_t; native GCC/Clang in both compiler directions and
profiles remains deferred to the batched Linux queue. No WSL provisioning was attempted.
Full go test ./..., hosted CI, push, publication and release were not run.

Only the declared bounded family is admitted. Aggregates, callback storage, indirect calls,
captures and declared effects remain closed. Retention, asynchronous/cross-thread delivery,
foreign unwinding and longjmp remain excluded. C provider adherence to synchronous lifetime
and raw-data validity is an unsafe integration obligation, not proved by these tests.

The language/C ABI subset and source-of-truth documents change. Grammar, SDK/CLI API, DB,
MIR opcodes and codec envelope, runtime, dependencies, repository hygiene and CI/runner
configuration are unchanged. No remote mutation was performed.
