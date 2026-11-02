# Windows zero-parameter inline arithmetic traps, 2026-11-03

This extends the [parameterized inline-trap receipt](windows-candidate-inline-traps-20261103.md)
to three literal-only i32 bodies. Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
Linux execution remains in the maintainer's deferred combined queue.

- Go correction: `b80b01f8`.
- Final fixtures and successful division control: `45a84f45`.
- Fairy production emitter: unchanged from `6c71738b`.

## Reproduction and correction

The Go reference native-test replay initially failed with MSVC C2220/C4702 before execution.
The smaller partitioned C probe independently reproduced optimized caller-continuation failures
for `10/0` and `-2147483648/-1`. Both low builds compiled and executed their expected traps.
Constant addition overflow already used the Go backend's proven non-returning path.
The initial small probe's exit expectations were corrected to the existing runtime category
numbers; those assertion failures are not counted as compiler defects.

Go fix `b80b01f8` changes `internal/cbackend/terminators.go`. A marked zero-parameter inline
i32 division guard reads the existing private volatile divisor under `_MSC_VER`. It retains
both the zero and signed-minimum/negative-one checks, their existing failure edge and location.
The original guards remain in the other preprocessor branch. The operation's original divisor
materialization and the addition-overflow proof are unchanged. No new storage or runtime helper,
warning suppression, forced no-inline attribute or source-language restriction is introduced.

The existing Fairy-authored CLI passed the extended optimized replay without a production
emitter change. A speculative literal-storage barrier was removed before final validation.
Its current C/image digests are checked by the existing host harness before artifact reuse;
the CLI processes still compile each fixture's actual current-source packet through the companion.
This is not a rebuilt Fairy compiler or a new performance measurement.

## Fixture and observations

`internal/cli/candidate_inline_trap_fixture_test.go` retains the three parameterized cases and
adds these exports in `libs/answer/src/answer.fy`:

| Expression in zero-parameter export | Trap | Provider line |
| --- | --- | --- |
| `10/0` | `division-by-zero` | 8 |
| `-2147483648/-1` | `signed-division-overflow` | 9 |
| `2147483647+1` | `integer-overflow` | 10 |

Both ordinary callers and generic bool-marker wrappers call each export. Every `run` must
exit 5 with empty stdout and the exact Go-reference provider diagnostic, including column.
Each case runs cold-development and locked-warm-release; warm cache generation hashes must
stay unchanged. Each inspected release packet has one reachable inline definition in the
shared header and none in body units. Windows `test` remains a non-cache command.

Each caller family also runs interpreter, native-development and native-release `test --json`.
Reports must exactly match the Go oracle: six categorized failures followed by one passing
sentinel that calls zero-parameter `40+2` and parameterized `add(40)`. This checks continuation
after traps and avoids accepting a fix that breaks successful constant expressions.

The partitioned Go C probe `TestPartitionedZeroParameterInlineTraps` also covers successful
`40+2` and `84/2` beside the three traps, at low and optimized: ten native executions.

## Validation

The final `TestCandidateInlineTrapCLI` replay passed in 578.403 s (whole test 578.19 s).
The low subtest passed in 418.56 s and optimized in 115.62 s. These are test-run times on a
shared development host, not comparable compiler performance measurements.

| Final external CLI observations | low | optimized |
| --- | --- | --- |
| Ordinary cold/warm run pairs | 6/6, exact Go diagnostics | 6/6, exact Go diagnostics |
| Generic cold/warm run pairs | 6/6, exact Go diagnostics | 6/6, exact Go diagnostics |
| Ordinary interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Generic interpreter/native/native-release reports | 3/3, exact Go JSON | 3/3, exact Go JSON |
| Independent external CLI processes | 30 | 30 |

The 60 external CLI processes comprise 48 run commands and 12 report commands. The reports
contain 84 rows: 72 categorized trap failures and 12 passing sentinels. Every inspected packet
passed shared-header placement checks; all locked warm cache generation snapshots remained
unchanged. The test recomputed its current Go reference once before both host profiles; those
18 in-process reference calls are separate from the external Fairy CLI count.

Other executed checks:

- Focused backend division, checked-operation and inline checks passed in 3.768 s.
- The added successful zero-parameter division control passed in both profiles in 0.940 s.
- The earlier expanded standalone Go reference passed in 105.365 s; final CLI validation
  recomputes the Go oracle with the successful zero-parameter sentinel added.
- The unchanged Fairy optimized baseline passed in 199.798 s, including 30 external CLI
  processes and six exact Go-matching reports, before adding that final sentinel assertion.
- Scoped `go vet -mod=readonly ./internal/cbackend` and `./internal/cli` passed.
- `git diff --check` passed. All 98 local Markdown path links in the six changed documents
  resolve; link anchors were not checked.

Native Linux is deferred because this laptop has no usable WSL. C17, the full Go suite,
sanitizers, hosted CI, distribution and broader arithmetic/constant-call coverage are omitted
from this bounded Windows regression. Volatile read-back cost is not benchmarked.
Public API, SDK ABI, MIR, cache wire format, DB, dependencies, repository hygiene and CI/runner
configuration are unchanged. No push, deployment or hosted workflow dispatch was performed.

## Reproduction commands

```powershell
go test -mod=readonly ./internal/cbackend -run 'Test.*(Division|Checked|Inline)' -count=1 -timeout 3m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to replay both host profiles.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineTrapCLI$' -count=1 -timeout 15m -v
go vet -mod=readonly ./internal/cbackend ./internal/cli
git diff --check
```
