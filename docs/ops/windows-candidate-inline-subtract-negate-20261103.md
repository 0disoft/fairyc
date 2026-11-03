# Windows inline subtraction and unary negation, 2026-11-03

This extends the [remainder/multiplication receipt](windows-candidate-inline-remainder-multiply-20261103.md)
with six named i32 cases. Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
Native Linux remains deferred at the maintainer's request because this laptop has no usable WSL.

- Go C-only proof correction and partitioned regressions: `d544d328`.
- CLI fixture: `b478d64a`, using the unchanged shared trap runner.

## Failure and correction

The initial partitioned probe passed 31 of 32 build/run cases. Only zero-parameter
`2147483647-(-1)` under optimized MSVC failed before execution: C2220/C4702 rejected the
caller's impossible assertion/return continuation. Low compiled and executed its expected trap.
The same optimized failure was isolated with the generated header, body and verified MIR.

The Go C-only assertion/failure analysis previously discarded the exact value produced by
unary negation. Its transfer rule now retains representable i32 literal negation, including
the unannotated i32 operands emitted by the normal lowerer. The resulting `-1` feeds the
existing subtraction-overflow proof, reachable-block selection and non-returning inference.
The original subtraction trap edge and source span remain authoritative.

`-INT32_MIN`, unknown locals, bool and other widths do not become new literal facts. Existing
indirect-write and unknown-operation invalidation remains intact. Nine analysis controls cover
explicit/implicit i32, safe results, minimum rejection, unknown input, wrong types and clobbering.
This adds no volatile storage/read-back, runtime helper, warning suppression, forced no-inline
attribute, source form, verified MIR change, ABI change or cache format field. The new transfer
is part of the general Go C-only proof analysis, rather than a source-level inline restriction.

## Fixture and controls

The CLI fixture is `internal/cli/candidate_inline_subtract_negate_test.go`.

| Call or zero-parameter body | Expected category | Provider line |
| --- | --- | --- |
| `subtract(-2147483648,1)` | `integer-overflow` | 8 |
| `subtract(2147483647,-1)` | `integer-overflow` | 8 |
| `negate(-2147483648)` | `integer-overflow` | 9 |
| `-2147483648-1` | `integer-overflow` | 10 |
| `2147483647-(-1)` | `integer-overflow` | 11 |
| `-(-2147483648)` | `integer-overflow` | 12 |

All providers are public, safe nongeneric i32 inline functions. Ordinary and generic bool-marker
callers invoke the same providers. Each case checks cold-development and locked-warm-release
`run`, exact Go diagnostics including provider column, empty stdout, exit 5 and unchanged warm
cache generation hashes. Actual release packets require one reachable inline definition in the
shared header and none in body units. Windows `test` retains its non-cache entry.

Each caller family also compares interpreter/native-development/native-release JSON reports
with Go. Six categorized failures must be followed by a passing sentinel. Its ten direct
provider assertions observe parameterized and zero-parameter subtraction preserving INT32_MIN
and INT32_MAX, plus successful negation of -42, INT32_MAX and 0. The partitioned Go C probe
asserts the same eight trap/control expressions in both shapes and profiles: 32 native executions.
It observes each exact boundary result before returning 42 for the successful controls.

## Final validation

`TestCandidateInlineSubtractNegateCLI` passed in 596.289 s (whole test 596.15 s).
The low subtest passed in 411.50 s and optimized in 139.21 s. These are test-run durations
on a shared development host, not compiler performance measurements.

| External CLI observations | low | optimized |
| --- | --- | --- |
| Ordinary cold/warm run pairs | 6/6, exact Go diagnostic | 6/6, exact Go diagnostic |
| Generic cold/warm run pairs | 6/6, exact Go diagnostic | 6/6, exact Go diagnostic |
| Ordinary interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Generic interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Independent external CLI processes | 30 | 30 |

The 60 external CLI processes comprise 48 run commands and 12 report commands. The reports
contain 84 rows: 72 categorized trap failures and 12 passing sentinels. Every inspected packet
passed shared-header placement checks and all locked warm cache generation snapshots remained
unchanged. The current Go oracle ran once before both host profiles; its 18 in-process calls
are separate from the external Fairy CLI count. The Fairy emitter needed no additional change.

The fixed partitioned probe plus focused assertion, constant and non-returning analysis tests
passed in 13.382 s. Scoped `go vet -mod=readonly ./internal/cbackend ./internal/cli` and
`git diff --check` passed. Earlier failed attempts are not counted as passing observations.
All 105 local Markdown path links in the six changed documents resolve; anchors were not checked.

The existing Fairy emitter passed the named full-CLI matrix. The private host harness
checks exact current C/image digests before artifact reuse, and each fixture's actual packet
is built by the C-only companion. This is not a self-hosting or distribution acceptance claim.

Linux, C17, full `go test ./...`, sanitizers and hosted CI are outside this bounded Windows
regression. It does not establish every arithmetic width, boundary, spelling or nested expression.
Public API, SDK ABI, DB, dependencies, repository hygiene and CI/runner configuration are unchanged.
No push, deployment or workflow dispatch is part of this work.

## Reproduction

```powershell
go test -mod=readonly ./internal/cbackend -run 'Test.*(Assert|Constant|NonReturning)|^TestPartitionedInlineSubtractNegate$' -count=1 -timeout 3m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to replay both host profiles.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineSubtractNegateCLI$' -count=1 -timeout 20m -v
go vet -mod=readonly ./internal/cbackend ./internal/cli
git diff --check
```

The outer Go replay allows 20 minutes to avoid repeating a partially completed host matrix on a
busy laptop. Product compile/execution deadlines and repository/CI timeout settings are unchanged.
