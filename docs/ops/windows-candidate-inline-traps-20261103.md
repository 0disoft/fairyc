# Windows candidate inline arithmetic traps: MSVC regression receipt, 2026-11-03

This receipt records a Windows native regression and its bounded fix for constant actual arguments
reaching checked traps in parameterized public `@inline` i32 arithmetic. It supplements the
[cross-package inline architecture](../architecture/candidate-inline-v1.md) and
[ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md) without changing the admitted
source forms or cache contract.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Go fix: `f838dc5c` (`fix: preserve runtime inputs in MSVC inline checked arithmetic`).
- Fixture/reference: `internal/cli/candidate_inline_trap_fixture_test.go`.
- Copied-host CLI: `internal/cli/candidate_inline_trap_cli_test.go`.
- Go source: `internal/cbackend/functions.go`.
- Fairy source: `module_unit_c_calls.inline_arguments` plus the body hook in
  `module_unit_c_function`; Fairy fix and native CLI evidence: `2e2f4978`.

## Failure and fix

MSVC native test compilation treated a constant-actual inline call as always trapping. It then
rejected the ordinary caller continuation with C4702 (`unreachable code`), promoted to C2220
(`warnings treated as errors`), before the program could execute. The fixture first removed a
redundant assertion after the trapping call, but the warning persisted. A generated-C diagnostic
isolated the same failure: the caller emitted an inline `divide(10,0)` call followed by a valid MIR
return, and MSVC diagnosed that return as unreachable.

The Go backend now reads each private i32 parameter through `volatile int32_t fy_inline_arg_N` and
writes it back before ordinary generated statements. The guard is emitted only for a function with
an `InlineBody` marker and nonzero parameters, and only under `#if defined(_MSC_VER)`. The Fairy
generator performs the same read-back once per parameter through its `inline_arguments` hook. Other
compilers preprocess the guard away. C inlining remains permitted; no warning suppression was added.

The public signature, MIR, language behavior, ABI and cache wire format are unchanged. The MSVC
barrier's overhead and effect on constant folding have not been benchmarked, so this receipt claims
neither a speedup nor complete handling of constant calls.

## Covered cases

| Caller expression | Required trap | Provider source |
| --- | --- | --- |
| `divide(10,0)` | `division-by-zero` | `libs/answer/src/answer.fy:5` |
| `divide(-2147483648,-1)` | `signed-division-overflow` | `libs/answer/src/answer.fy:5` |
| `add(2147483647)` | `integer-overflow` | `libs/answer/src/answer.fy:3` |

Each case runs through an unchanged nongeneric caller and a generic `bool`-marker wrapper. Both
call the same nongeneric public safe i32 inline `add` and `divide` exports. The reference requires
the exact provider path, line and positive column.

## Reference and native CLI evidence

The standalone Go reference passed in 25.27 s. Its six `run` calls each exited with program-trap
status 5, wrote empty stdout, and reported the exact provider location above. Its six `test` calls,
one interpreter/native/native-release command for each caller family, each produced the canonical
four rows: three categorized failures followed by the passing sentinel. The reference test has a
Windows amd64 guard so Linux defers the Windows native observation instead of claiming it.

| Native CLI result | low | optimized |
| --- | --- | --- |
| Ordinary program traps | PASS 6/6, exact Go | PASS 6/6, exact Go |
| Generic program traps | PASS 6/6, exact Go | PASS 6/6, exact Go |
| Ordinary test reports | PASS 3/3, exact Go | PASS 3/3, exact Go |
| Generic test reports | PASS 3/3, exact Go | PASS 3/3, exact Go |
| Whole test / CLI subtest | PASS, 352.62 s / 325.41 s | PASS, 364.63 s / 337.24 s |

Both profiles completed. The 36 external CLI processes comprise 24 cold-development and
locked-warm-release program-trap executions plus 12 report commands. Every run diagnostic matched
the Go reference as full bytes, and every report's outcome JSON matched exactly. The 12 reports
contain 48 rows: 36 categorized failures and 12 passing sentinels. These are test-run times, not
benchmarks.

Each native CLI test invocation also computes its own 12 in-process Go oracle calls;
the standalone reference's 12 calls are separate and are not Fairy CLI processes.

All native runs matched the reference diagnostic bytes and outcome JSON exactly. Locked
warm generations remained unchanged. Every inspected release packet contained one inline header
definition and no inline body in any body unit. There is no standalone direct-product or observer
image addition.

## Host and cache boundary

The tests reuse the authored `file_cli` host with its 169 private C parts, original main and
commands, and 16 MiB private stack reservation, plus the original C-only `fairyc-candidate-build`
companion. Source changed for this regression, so hosts rebuild; the prior 169-part source/image
digest is not reused as current evidence. Each case exercises the complete cache through a
cold development `run` and a locked warm release `run`. Windows `test` remains a non-cache entry.

## Validation and limits

Focused Go compiler, workspace and backend inline tests passed in 0.142 s, 3.084 s and 1.262 s.
`go vet -mod=readonly ./internal/cbackend ./internal/cli` and `git diff --check` passed.
Native Linux, full `go test ./...`, C17, a broad new operator or zero-parameter trap matrix,
sanitizers, hosted CI, distribution and self-hosting were skipped for this receipt.

The production private C emission now includes the MSVC parameter barrier. Public API, SDK ABI, DB,
dependencies, repository hygiene, CI and runner configuration are unchanged. Only the three cases
above are covered; zero-parameter always-trapping inline forms, all operators and broader targets
are not newly proven. No push or deployment is part of this evidence.

## Reproduction

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to run both profiles; set low or optimized to select one.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineTrapReference$' -count=1 -timeout 5m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineTrapCLI$' -count=1 -timeout 20m -v
go vet -mod=readonly ./internal/cbackend ./internal/cli
git diff --check
```
