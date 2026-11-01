# Windows zero-field and nested owning resource receipt, 2026-11-01

- Zero-field implementation/core/native probe: `24561cbd` (local commit).
- Nested owning implementation, native probes and receipt: commit containing this file.
- Host: Windows amd64, MSVC 14.51.36231.
- Owners: [ADR 0491](../adr/0491-zero-field-resources.md),
  [ADR 0492](../adr/0492-nested-owning-tag-cleanup.md), language values/resources and MIR v0.

## Executed evidence

| Scope | Final result |
| --- | --- |
| Zero-field core | Verified MIR and codec round trip, initialized private size/alignment 1/1 on both emission targets, moves via locals/parameters/returns, borrow and direct resource parent. Interpreter observes Token,Empty,Parent,Token,Empty,Token,Empty exactly once each. |
| Zero-field native | MSVC low/optimized pass. Independent C observer requires exact 1,9,1,1 cleanup, including nonempty zero-field cleanup and inactive Option. Silent zero-field cleanup uses no C event. Both images exit 42 with empty streams. |
| Nested owning core | Option/Option, Option/Result and Result/Option/Result; all active branches, outer/inner none, local and return/parameter moves, matching into owned inner Result and live reassignment. Codec round trip and eight exact interpreter resource events pass. |
| Native combined matrix | Four passes, C11/C17 x low/optimized. Every image checks 27 exact cleanup events using independently compiled generated caller and C observer objects, exits 42 and has empty stdout/stderr. PE and AMD64 unwind inspection pass. |
| Parent mutation | Active Box cleanup records 10 and changes its first Leaf from 1 to 99. Reverse field cleanup records 2,7,99. Inactive outer/inner tags produce no child cleanup. All 27 expected events must be consumed before success. |
| Trace equivalence | Two low/optimized runs match interpreter resource type plus source-binding site identity/order for the eight-event nested-tag fixture. Existing four-event nested-parent trace also passes both profiles. Production C has no trace hooks. |
| Boundaries | Eight owning tag layers emit; nine reject. Use-after-move, resource partial move and using a MaybeLive root reject. Owning tag fields and unsupported f64 sibling representations remain rejected at emission. Extra zero-field constructor fields, missing definitions and invalid cleanup reject. |
| Forged extraction | Clearing Option/Result payload move metadata, retyping the payload or substituting a scalar container all fail MIR verification and C emission. Exact payload/container identity is checked; shared inert type-graph facts are memoized. |
| Cache | Cross-package owned nested tags retain resource definitions and obligations through process-memory and fresh-session disk cold/warm restoration, with C emission after warm hits. |
| Scoped packages | Full semantic, MIR, interpreter, module-interface, package-artifact and C-backend tests pass. Focused compiler/workspace cache checks pass. Scoped vet, gofmt and git diff --check pass. |

The private byte is initialized storage, not an ownership flag or source-visible field.
The C observer validates byte mutation only through memcpy to the first nested i32 field.
No external C ABI representation for resource values is introduced.

## Reproduction

```powershell
go test -mod=readonly ./internal/compiler -run '^(TestZeroFieldResource|TestNestedOwningResource|TestPrivateVectorNestedResourceDropTrace)' -count=1 -v
go test -mod=readonly ./internal/workspacecompiler -run '^TestNestedOwningResourceCache' -count=1 -v
go test -mod=readonly ./internal/semantic ./internal/mir ./internal/interpreter ./internal/moduleinterface ./internal/packageartifact ./internal/cbackend -count=1
go vet -mod=readonly ./internal/cbackend ./internal/mir ./internal/interpreter ./internal/compiler ./internal/workspacecompiler
git diff --check
```

Focused filters were run in separate steps during development. This combined filter replays
those final observations; repeated matrix runs are not additional distinct coverage.

## Omissions and remaining limits

Native Linux stays deferred by maintainer request. TestNestedOwningResourceLinux compiles
and skips on Windows; no WSL provisioning or Linux sanitizer run was attempted. Later native
GCC/Clang low/optimized and ASan/UBSan are listed in the ordered queue. Linux-profile C emission
is not a native pass. Full repository go test ./..., hosted CI, push and publication were omitted.

This bounded slice admits zero-field resources and nested builtin owning tags only. Owning
user enums/arrays, tag-bearing resource fields, deeper owning transport, unsupported payload
leaves and public resource/tagged ABI remain closed. Trap/abort is not a normal cleanup edge.
Language and MIR source-of-truth docs and the private C backend extend; source grammar,
SDK/CLI and foreign ABI, runtime, dependencies, DB, repository hygiene and CI/runner policy do
not change. No remote mutation was performed.
