# Windows inline remainder and multiplication observations, 2026-11-03

This extends the [zero-parameter trap receipt](windows-candidate-inline-zero-traps-20261103.md)
with six named i32 cases. Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
Native Linux remains deferred at the maintainer's request because this laptop has no usable WSL.

- Shared fixture/runner refactor: `f8f3180a`.
- New fixtures and partitioned controls: `f520cbdd`.
- Production Go/Fairy compiler sources: unchanged from `ecca7ce5`.

The original division/addition provider, six cases and passing sentinel are retained as a
separate fixture. Its Go reference replay passed in
54.569 s after the refactor. The original 60-process Fairy CLI matrix was not repeated for this
test-only refactor; the unchanged production emitter evidence remains in the previous receipt.

## New scope

The new fixture is `internal/cli/candidate_inline_remainder_multiply_test.go`. Every provider
is public, safe, nongeneric and `@inline`; generic bool-marker callers wrap the same provider.

| Call or zero-parameter body | Expected category | Provider line |
| --- | --- | --- |
| `remainder(10,0)` | `division-by-zero` | 8 |
| `remainder(-2147483648,-1)` | `signed-division-overflow` | 8 |
| `multiply(46341,46341)` | `integer-overflow` | 9 |
| `10%0` | `division-by-zero` | 10 |
| `-2147483648%-1` | `signed-division-overflow` | 11 |
| `46341*46341` | `integer-overflow` | 12 |

Each caller family checks cold development and locked warm release `run` commands, exact Go
diagnostics including provider path/line/column, empty stdout and exit 5. Warm cache generation
hashes must stay unchanged. Inspected release packets require one reachable inline definition
in the shared header and none in body units. Windows `test` remains a non-cache command.

Each family also checks interpreter, native-development and native-release JSON reports. Six
categorized failures must be followed by one passing sentinel. Its direct provider calls check
`85%43 == 42`, `-85%43 == -42`, `6*7 == 42` and `-6*-7 == 42`; zero-parameter exports repeat
positive/negative remainder and positive product controls. Full report JSON must match Go.

The partitioned C probe `TestPartitionedInlineRemainderMultiply` separately compiles and executes
three traps and three successful controls with both parameterized and zero-parameter inline
providers, under low/optimized: 24 native executions. Negative remainders are normalized by
adding 84 so the observing executable returns 42. The final probe passed in 12.143 s.

## Final CLI replay

`TestCandidateInlineRemainderMultiplyCLI` passed in 736.657 s (whole test 736.47 s).
The low subtest passed in 483.33 s and optimized in 160.13 s. These are test-run durations
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
are separate from the external Fairy CLI count.

Both existing compilers pass the named matrix without another production correction. The existing
parameter read-back, zero-parameter divisor guard and constant-overflow handling are reused.
Current-source C/image digests are checked before reusing the private Fairy CLI host artifacts;
fixture packets and their compiled programs are produced by the actual CLI/companion path.

Scoped `go vet -mod=readonly ./internal/cbackend ./internal/cli` and `git diff --check` passed.
All 97 local Markdown path links in the five changed documents resolve; anchors were not checked.
Native Linux, C17, the full Go suite, sanitizers, hosted CI and distribution are omitted from
this bounded Windows matrix. It does not cover all operand widths, signed overflow boundaries,
operators, expression nestings or constant calls. Runtime cost of the existing volatile barriers
is not benchmarked. Public API, SDK ABI, MIR, cache wire format, DB, dependencies, repository
hygiene and CI/runner settings are unchanged. No push, deployment or workflow dispatch occurred.

## Reproduction

```powershell
go test -mod=readonly ./internal/cbackend -run '^TestPartitionedInlineRemainderMultiply$' -count=1 -timeout 3m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to replay both host profiles.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineRemainderMultiplyCLI$' -count=1 -timeout 15m -v
go vet -mod=readonly ./internal/cbackend ./internal/cli
git diff --check
```
