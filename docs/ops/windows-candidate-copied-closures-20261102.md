# Windows Fairy candidate copied closure receipt, 2026-11-02

- Sources: `0ccb40ef` (syntax/AST), `69200ca5` (private typed calls), and the
  `88b42e0c` source/artifact integration commit, followed by `e0f82ea5`
  (shadowed-closure iteration admission).
- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Scope: [candidate copied closures v1](../architecture/candidate-copied-closures-v1.md).

## Executed evidence

| Probe | Result and scope |
| --- | --- |
| Syntax | Four original function ASTs equal Go frontend JSON byte for byte in low/optimized; copied source preserves the written identity. |
| Admission and typing | Both native profiles pass snapshot/call typing, a closure in a generic function, 24 syntax/lowering rejections and five ordinary semantic rejections. Escapes, mutation, implicit/resource captures, helper-prefix access, labels, budgets and wrong capture/argument/result types reject. |
| Shadowed closure iteration | A named byte view can iterate when an ordinary local hides the only active closure. The Go interpreter returns 42 and the Fairy low/optimized type replay passes. Active, unshadowed closures still reject view/vector for-in. This admission-only follow-up reuses the earlier C pipeline receipt; the complete pipeline was not rerun for it. |
| Go identity oracle | Both native profiles independently match Go `PreparePackages` root IDs, specialization IDs, parameter bindings, dependencies and public template closure hashes for four cross-package graphs. Written closure body and capture initializer edits invalidate dependent identity; an unrelated private edit preserves it. One replay also shifts package-owner ordinals. |
| Restored C execution | Two native Fairy profiles encode/decode/restore against current source and emit C11; low/optimized MSVC consumers execute all four outputs with exit 42 and empty streams. Checks cover snapshots after mutation, multiple closures, zero/four captures, four parameters/eight helper arguments, nested calls/shadowing, while/classic-for, ordered extern-C argument observers and i32/i64 generic callers. The complete replay passes in 198.78 seconds. |
| Go reference | `go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^TestCopiedClosure' -count=1` passes, including snapshots, ordered arguments, detached generic syntax and workspace memory/fresh-session disk checks. |
| Candidate source | Isolated bootstrap `check` and `check --locked` pass: two instances, 588 source files, 7,385 MIR functions. |
| Hygiene | `go vet -mod=readonly ./internal/cli` and `git diff --check` pass. No generated/cache output is staged. |

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosure(Syntax|Calls|Identity)$' -count=1 -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
go test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosurePipeline$' -count=1 -timeout 10m -v
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^TestCopiedClosure' -count=1
go vet -mod=readonly ./internal/cli
git diff --check
```

The complete host uses 95 private C parts and a 16 MiB native stack. It compiles
original source with Fairy, encodes/decodes the complete artifact, restores it
against current source, and emits C. Go supplies independent identity expectations
and builds the host; it does not supply a pre-expanded executable AST.

## Deferred and unchanged

Native Linux stays in the combined queue. These Windows-only replay harnesses
need a GNU companion path before Linux execution can be claimed. Candidate C17,
interpreter execution, full candidate CLI build, fresh-session persistent cache,
full `go test ./...`, hosted CI, self-hosting and clean-machine distribution were
not run. Go cache evidence does not certify Fairy candidate persistent caching.

Source-of-truth changes: ADR 0496, the architecture page, this receipt, the ordered
work/Linux queue and package/CLI/SDK index. No public Go/CLI API, SDK ABI, DB,
dependency, cache wire version, repository policy, runner or CI configuration
changes. No push, publication, deployment or workflow dispatch.
