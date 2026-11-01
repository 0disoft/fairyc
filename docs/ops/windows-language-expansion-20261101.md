# Windows bounded generics, interfaces and closures receipt, 2026-11-01

- Scalar implementation: `1e351f03`; static interfaces: `adb5d38c`.
- Closure syntax: `8f3f3401`; execution, final diagnostics and receipt: containing commit.
- Host: Windows amd64; MSVC 14.51.36231; native modes C11/C17, profiles low/optimized.
- Owners: ADRs [0494](../adr/0494-broader-scalar-generic-transport.md),
  [0495](../adr/0495-explicit-static-operation-interfaces.md) and
  [0496](../adr/0496-explicit-copied-local-closures.md).

## Observed behavior

| Slice | Windows evidence |
| --- | --- |
| Wider direct scalar generics | Seven new scalar families, two explicit generic forwarding functions and repr(C) independent C observer records. Exact negative signed boundaries, maximum u16/u64 and exact f32/f64 values survive both functions. Four native executions exit 42 with empty streams. Private, explicit-file and package-aware semantic/MIR paths and fourteen memory/fresh-session disk cases pass; original composite signature rejections and name-resolution precedence remain intact. |
| Static operation interfaces | Explicit i32 and i64 providers select different algorithms and return 42 in interpreter and four native executions. Missing/duplicate providers, unsafe/wrong signatures, body-bearing declarations and bad attributes reject. Cross-package public operations and algorithms restore memory/fresh-session disk hits. Provider-body changes miss owner/dependent; unrelated private changes retain the dependent key. Missing providers reject before cache reuse with source diagnostics. |
| Copied closures | The original mutable value changes after creation; repeated calls retain the old snapshot, while a later closure sees the new value. Zero and four captures, four explicit arguments/eight helper arguments, nested lexical shadowing and classic-for calls pass in interpreter and four native executions. Source syntax is not mutated. |
| Ordered closure arguments | A separately compiled C counter rejects reordering or duplicate evaluation and requires exactly two calls. All four native executions return 42 with empty streams. Captures are prior snapshot locals; each explicit argument remains a normal MIR argument. |
| Closure bounds and persistence | Return/copy, mutation/implicit capture, non-i32 captures, duplicates, nested declarations, annotations, reserved helper calls, oversized capture/parameter/body budgets and parameter-name collisions reject. Generic expansion detaches capture/signature/body AST edges. A generic closure executes in the interpreter. Ordinary/generic memory/disk cold/warm, fresh-session restoration and body edits pass with both-target C emission. Ordinary private closure-body edits retain the dependent key; public generic closure-body edits invalidate it. |
| Related regressions | Full syntax, AST, HIR, semantic, MIR, generic source/planner, public-interface and package-artifact tests pass. Diagnostic/formatter packages pass. Focused compiler/workspace tests, scoped Go vet, gofmt and diff checks pass. |

Sixteen distinct native executions are four fixtures x four mode/profile combinations.
Repeated development runs are not extra coverage. POSIX emission is not native Linux proof.

## Representative size and build observations

One final local run recorded the following. C byte counts exclude the separately compiled scalar
runtime/C observer. Build time includes writing/compiling/linking inputs, not Go bootstrap or
clean-machine setup. These are small fixture observations, not throughput or speedup guarantees.
No assembly-level inlining claim is made.

| Fixture | MIR functions | MSVC C bytes | PE bytes low / optimized | Native build range |
| --- | ---: | ---: | ---: | ---: |
| Seven wider scalar generic families | 15 | 15,664 | 111,616 / 109,056 | 0.281-0.346 s |
| Two explicit static operation providers | 7 | 4,033 | 109,568 / 107,520 | 0.270-0.313 s |
| Four copied closures and capture boundaries | 5 | 12,041 | 111,104 / 109,056 | 0.346-0.387 s |
| Ordered closure arguments | 2 | 3,107 | 109,056 / 108,544 | 0.284-0.375 s |

## Reproduce

```powershell
go test -mod=readonly ./internal/compiler -run '^(TestBroaderGenerics|TestStaticInterface|TestCopiedClosure|TestGenericSignature)' -count=1 -v
go test -mod=readonly ./internal/workspacecompiler -run '^(TestBroaderGenerics|TestStaticInterface|TestCopiedClosure)' -count=1 -v
go test -mod=readonly ./internal/syntax ./internal/ast ./internal/hir ./internal/semantic ./internal/mir ./internal/genericsource ./internal/genericplan ./internal/moduleinterface ./internal/packageartifact ./internal/diagnostic ./internal/formatter -count=1
go vet -mod=readonly ./internal/closureprep ./internal/semantic ./internal/genericsource ./internal/compiler ./internal/workspacecompiler
git diff --check
```

Package checks were run in development groups and consolidated above for replay. The native helper
selects GCC/Clang x low/optimized on Linux; no WSL or native Linux run was attempted. The ordered
queue retains commands for a later batch. Full repository tests, Linux sanitizers, hosted CI, push,
release and publication remain unexecuted.

This is a Go Stage 0 prototype extension, not arbitrary generics, multi-method/dynamic dispatch,
escaping/stored closures, general higher-order programming, or Fairy candidate/SDK promotion.
Checked arithmetic and planner limits remain in force. Interface effects follow selected ordinary
calls; no implementation-independent effect ceiling is promised. View/vector for-in together with
an active copied closure is separately closed.

Owning changes: ADRs 0494-0496 and older deferral/signature notes, closure grammar, lexing/typing
extension notes, E4502 explanation, source indexes and ordered queue. CLI flags/SDK API,
runtime/foreign ABI, dependencies, DB, repository hygiene and CI/runner settings are unchanged.
No remote mutation was performed.
