# Copied closure values through the actual CLI, 2026-11-08

Scope: the first CLI acceptance item in `fairy-handoff-20261108.md`, for
ADR 0551's bounded `fn i32(i32)` values. No compiler, ABI, dependency,
lockfile, workflow or runner change is needed for the accepted commands.

## Original source and command ownership

`internal/cli/candidate_closure_value_cli_test.go` uses the same original
workspace as `closure_value_cli_test.go`. It includes a separate provider
package, returned factories, four copied captures, a capture-free different
helper, returned ordinary structs, local and struct copies, snapshots,
separate environments, once-only argument evaluation and an unused trapping
body. The fixture is not rewritten into lifted helpers by the test.

The executable retains the Fairy-authored CLI main and its commands. The
existing private compiler-host builder compiles 210 bounded C parts with a
16 MiB stack reserve and a one-minute limit per native build command. The Go
companion consumes C packets emitted by Fairy; it does not compile the input
workspace with the Go frontend. Source/header/support/profile and EXE hashes
must match before an existing validation host can be reused.

The commands exercise `check`, locked check, `build`, direct execution of
each published EXE, `run`, JSON test listing, interpreter tests and native
tests with captured output. Native application commands use development and
release profiles. The test report must preserve ordered identities and the
two division-by-zero traps, one integer-overflow trap and one passing test.
The final harness also selects only the passing test, checks exit zero and
its JSON report, and verifies the source lock was not changed by locked
commands.

## Executed Windows checks

| Check | Result |
| --- | --- |
| Existing Go CLI `StableSource`, `WindowsNative`, `WindowsRunTraps`, `PersistentCache`, `Refusals` | PASS, package 23.502 s; both native profiles, interpreter, fresh-process cache/repair and 13 rejected source shapes |
| Fairy CLI low | PASS, 260.13 s including host preparation/build; 10 independent CLI processes and two directly executed build outputs |
| Fairy CLI optimized, final harness | PASS, 306.85 s including host preparation/build; 13 independent CLI processes and two directly executed build outputs; includes three successful exact selections and locked-byte check |
| Fairy CLI low, exact-selection follow-up | PASS; the same original source and exact-digest low host, interpreter plus development/release native selection, exit zero, schema/target/profile/streams and unchanged lock bytes |
| `go vet -mod=readonly ./internal/cli` | PASS on both new test sources |

The low run preceded the three extra exact-selection assertions and explicit
lock snapshot. Its full four-test reports already included the successful
value case in interpreter and both native modes. This receipt does not
claim that those later assertions were executed in that low run. The separate
follow-up above closes those specific low-profile observations without
repeating the unchanged full command/cache sequence. The initial ad-hoc
follow-up incorrectly expected interpreter schema v3; inspection of the
owning `stableTestReportSchema` constant established v1. The final check
requires v1 for interpreter and v4 for captured native reports.

The failed tests inside the JSON reports are intentional fixture programs;
the surrounding acceptance test passes only for their specified trap kinds,
ordering, empty captured streams and aggregate exit status 5. Success-only
selection must instead return zero.

Reproduce the Fairy acceptance from the repository root:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/closure-cli-host-20261108'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueCLI$' -count=1 -timeout=20m -v
```

`FAIRYC_CLOSURE_CLI_PROFILE=low` or `optimized` selects one producer profile.
The existing opt-in gate keeps this full CLI replay out of ordinary package
tests; no CI job or default runner scope was added. Local logs are ignored
`.cache/closure-cli-go-20261108.log`, `closure-cli-fairy-low-20261108.log`,
`closure-cli-fairy-optimized-20261108.log` and `closure-cli-vet-20261108.log`.
The low follow-up is `.cache/closure-cli-exact-low-20261108.log`; its disposable
workspace was populated directly from the existing raw Go fixture constants.

## Boundaries

Separate source/provider/identity mutations and corruption recovery through
the Fairy CLI are owned by `TestCandidateClosureValuePersistentCache` and
its follow-up receipt. The Go persistent-cache pass does not establish the
Fairy cache path. Go rejection coverage also does not close the historical
Go/Fairy diagnostic/status difference for invalid copied-closure inputs.

No whole Go suite, candidate self-host/fixed point, native Linux, clean-machine
distribution or hosted CI ran here. Those gates remain in the handoff; no
push, workflow dispatch, publication or deployment was performed. Normative
source-of-truth documents, public APIs, DB, repository hygiene and runner
configuration are unchanged.
