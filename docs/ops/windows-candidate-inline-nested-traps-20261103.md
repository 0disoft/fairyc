# Windows nested inline first-failure ordering, 2026-11-03

This extends the [subtraction/negation receipt](windows-candidate-inline-subtract-negate-20261103.md)
with one representative competing-failure expression and a successful nested-expression control.
The [existing language order](../language/0004-results-traps-and-control-flow.md#evaluation-order)
evaluates operands left to right; the first evaluated trap stops the expression.
Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231. Linux remains in the maintainer's deferred batch.

- Go checked-storage correction and partitioned probe: `2acd75c7`.
- CLI fixture and optional exact-column checks: `d469171d`.

## Reproduced build failure

Before the correction, the partitioned probe passed 14 of 16 build/run cases. Both profiles of
the zero-parameter inner-overflow case failed compilation with MSVC C2220, C4101 and C4189.
An earlier proven overflow removed the division and its dynamic trap path, but the private
divisor, pending category and pending location declarations remained unused. The standalone Go
reference also reproduced this before program execution; it was not an observed wrong trap order.

The Go backend now collects divisor types from executable blocks and removes unreachable dynamic
trap targets while preparing executable requirements. All verified MIR still undergoes capability
validation first. Existing reachability proofs, operation ordering, trap categories and source spans
are unchanged. No warning level, runtime helper, volatile barrier, MIR/ABI/cache format or source
admission is changed.

## Representative cases

The shared parameterized provider returns `(a+1)/b+(2147483647+1)`. The zero-parameter providers
substitute the same literals. Expected columns below are independently counted from the authored
provider source, rather than copied from either compiler's diagnostic.

| Call or zero-parameter body | First failure | Provider line:column |
| --- | --- | --- |
| `nested(2147483647,0)` | Inner `a+1` overflow before division/right addition | `8:48` |
| `nested(9,0)` | Division by zero before right addition | `8:47` |
| `nested(9,2)` | Right `2147483647+1` overflow after successful division | `8:56` |
| `(2147483647+1)/0+(2147483647+1)` | First addition overflow | `9:41` |
| `(9+1)/0+(2147483647+1)` | Division by zero | `10:43` |
| `(9+1)/2+(2147483647+1)` | Right addition overflow | `11:49` |

Ordinary and generic bool-marker callers invoke the same public safe nongeneric inline exports.
Each case checks cold-development and locked-warm-release `run`, exit 5, empty stdout, exact Go
diagnostic bytes, the independent expected column and unchanged warm cache generation hashes.
Actual release packets require one reachable inline definition in the shared header and none in
body units. Existing fixture groups retain their old positive-column check when no explicit column
list is supplied; this follow-up does not repeat their full CLI observations.

Each caller family also compares interpreter/native-development/native-release JSON reports with
Go. Six categorized failures must be followed by a passing sentinel. Its direct provider calls
observe `-(-((a+1)/b))+(39-2)` with `(9,2)` and its zero-parameter equivalent both returning 42.
Windows `test` remains a non-cache command. The separate partitioned C probe compiles and executes
the three failure stages and successful control in both shapes/profiles: 16 native executions.

## Final validation

`TestCandidateInlineNestedTrapCLI` passed in 684.398 s (whole test 684.08 s).
The low subtest passed in 482.00 s and optimized in 128.72 s. These are test-run durations
on a shared development host, not compiler performance measurements.

| External CLI observations | low | optimized |
| --- | --- | --- |
| Ordinary cold/warm run pairs | 6/6, exact Go and independent columns | 6/6, exact Go and independent columns |
| Generic cold/warm run pairs | 6/6, exact Go and independent columns | 6/6, exact Go and independent columns |
| Ordinary interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Generic interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Independent external CLI processes | 30 | 30 |

The 60 external CLI processes comprise 48 run commands and 12 report commands. Reports contain
84 rows: 72 categorized trap failures and 12 passing sentinels. All inspected packets passed
shared-header placement checks and all locked warm cache generation snapshots stayed unchanged.
The current Go oracle ran once before both host profiles; its 18 in-process calls are separate
from the external Fairy CLI count. The existing Fairy emitter required no additional correction.

The fixed partitioned probe plus focused assertion, constant, non-returning and divisor-storage
checks passed in 9.122 s. Scoped `go vet -mod=readonly ./internal/cbackend ./internal/cli` and
`git diff --check` passed. Failed attempts are not counted as passing observations.
All 110 local Markdown path links in the six changed documents resolve; anchors were not checked.

The private host harness checks exact current C/image digests before reusing a Fairy CLI artifact;
each fixture's packet is produced by that CLI and compiled by the C-only companion. This does not
constitute self-hosting, distribution or all-expression acceptance.

Native Linux is deferred because this laptop has no usable WSL. C17, full `go test ./...`, sanitizers
and hosted CI are outside this bounded Windows regression. The cases do not prove all nestings,
operators, operand widths or interactions with effects/callbacks/cleanup. Public API, SDK ABI, DB,
dependencies, repository hygiene and CI/runner settings are unchanged. No push, deployment or
workflow dispatch was performed.

## Reproduction

```powershell
go test -mod=readonly ./internal/cbackend -run '^TestPartitionedInlineNestedTraps$|Test.*(Assert|Constant|NonReturning)|^TestEmit(MaterializesCheckedDivisionDivisor|CheckedUnsignedDivision)$' -count=1 -timeout 3m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to replay both host profiles.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineNestedTrapCLI$' -count=1 -timeout 20m -v
go vet -mod=readonly ./internal/cbackend ./internal/cli
git diff --check
```

The outer Go test budget does not change product build/execution deadlines or CI timeout settings.
