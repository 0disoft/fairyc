# Linux native validation audit - 2026-11-18

Scope: existing verification gates and the named pending Linux observations below.
This audit does not expand language, ABI, SDK or target-support contracts.
The local source baseline is `6fc6dc98bfc54aff412c0ffebdb3990866d5ef27`.

## Current blocker

The latest Frontend run returned by the repository run listing at audit time,
[43871201121](https://github.com/0disoft/fairyc/actions/runs/43871201121), tested
`801c5d8caa64d19aab8925763d6bf07f65f26418` on 2026-11-16.
Its [Ubuntu job](https://github.com/0disoft/fairyc/actions/runs/43871201121/job/133312783046)
failed in Vet with `overlay_transport_test.go:43:11: undefined: overlayOraclePort`.
The Linux Go tests and their evidence collector never ran. The separate C differential
job was skipped; it also depends on Windows jobs that failed independently.
An aggregate failed run is therefore not evidence that every Linux native case failed.

The same undefined-symbol error reproduces locally with Linux/amd64 selected for
`go vet -mod=readonly ./internal/contractstatenative`. The portable wire test references
an oracle declared in `overlay_runtime_windows_test.go`, which Go excludes on Linux.
Move that unchanged oracle to `overlay_oracle_test.go`, leaving Windows C probes in
their platform-specific file. Do not skip the portable test or weaken CI dependencies.
This is the single implementation change selected by this audit: the compile blocker
prevents the entire Linux Go test suite from starting.

## Pending observation inventory

| Area | Existing owner / evidence | Still unconfirmed |
| --- | --- | --- |
| Runtime and SDK focused matrix | [Coverage inventory](linux-runtime-coverage.md), [validation gate](../../VALIDATION.md), `.github/scripts/linux_phase2_evidence.py` | Current v5 requires 10 roots and 40 GCC/Clang low/optimized cases. Recorded v4/32-case evidence is historical; it does not establish the additional eight File Buffer cases or the current source. |
| Expanded C ABI and cleanup | [Ordered work](ordered-work-20261031.md), items 13-16; [ordinary callback probe](../../internal/foreignstructprobe/ordinary_callbacks_test.go) | The named record arrays, exact C-scalar pointers, ordinary callbacks, zero-field resources and nested owning tags retain Linux deferral in that queue. Older bounded F1-F3 receipts do not establish these expanded scopes. |
| Inline, generics, interfaces and closures | [Ordered work](ordered-work-20261031.md), items 17-18 | Named Windows prototype observations do not establish corresponding Linux native execution or fresh-process cache behavior. |
| Fairy GNU companion and self-build | [Ordered work](ordered-work-20261031.md), item 2 | Windows policy execution and both-target emission do not establish actual GNU compilation/linking, Linux process/filesystem behavior or consecutive native self-build. |
| Linux remote packages | [Linux fetch receipt](candidate-linux-fetch-cli-20261106.md) | Actual Linux compilation, trusted TLS download and consecutive full Linux self-build remain unverified in the owning receipt. |

These are bounded evidence gaps, not a claim that each feature is unimplemented.
The earlier runtime/SDK successes and [cleanup receipt](phase2-remaining-work.md#hosted-cleanup-receipt-2026-10-23)
retain their original source revisions and scope. No target maturity flags are promoted.

## Validation and remaining boundary

- Before the move: Linux-target package vet fails on the undefined oracle.
- After the move: `go vet -mod=readonly ./...` passes with `GOOS=linux` and
  `GOARCH=amd64`; the PowerShell invocation restores both previous environment values.
  This checks all current Linux-selected Go packages and tests, not just the failing package.
- `go test -mod=readonly ./internal/contractstatenative -run '^TestOverlayTransport'
  -count=1 -timeout 1m` passes on Windows. Its four test roots exercise input framing,
  decoding, corruption rejection and failure responses without published writes.
- Go formatting, diff whitespace checks and the audit's local link targets pass.
- Full Go test/native C suites and the unchanged evidence collector are not rerun.
  The change only relocates the shared Go oracle; executing unrelated Windows native
  suites would not supply the missing Linux observation.
- The Windows host's `wsl --list --quiet` returned
  `Wsl/CallMsi/Install/REGDB_E_CLASSNOTREG`; no usable local Linux execution environment
  was established. No WSL installation, remote environment setup, push or CI dispatch
  was performed.

Linux-target vet is a cross-platform compile/static check, not Linux execution.
Windows wire tests are host-neutral transport checks, not C ABI/native observations.
Closing the remaining matrix requires native Linux execution with the reference toolchain
and no missing or skipped required cases. Existing unrelated Windows failures can still
block the C differential job; this oracle relocation does not repair those failures.

## Windows dependency follow-up

The same audited run's Windows native smoke failed first while building
`examples/windows_target/main.fy` in the low profile, with CLI exit 4 and
`internal compiler invariant failed`. `TestWindowsFairyParserLiterals` independently
reproduces exit 4 locally. A direct smoke-source regression exposes the underlying
MIR error: `V1002`, initialization state exceeds the 1,048,576-entry snapshot budget.

The verifier retained ordinary locals after their final possible mention and copied
them into later branch/join snapshots. Retire only non-resource locals whose last
mention precedes a conservative lower bound on every successor's reachable blocks.
Later backedges and chains of backedges lower that bound; resources, drop flags and
Region state retain their existing checks. The snapshot limit is unchanged.
This fixes the selected verifier cause rather than disabling Windows dependencies.

Validation of this follow-up:

- The previously rejected smoke source now verifies and emits Windows C.
- Actual `experimental run-target` of that source returns 42 with MSVC 14.51.36231
  in both low and optimized profiles, with no unexpected program output.
- All `internal/mir` and `internal/mirverifyfixture` tests pass, including loop/budget
  regressions and verifier mutation rejection. Related MIR/backend vet passes.
- The existing `TestWindowsFairyParserLiterals` and
  `TestWindowsFairyParserRecordExpressions` pass their native probes after the fix
  (`go test -mod=readonly ./internal/cli -run
  '^(TestWindowsFairyParserLiterals|TestWindowsFairyParserRecordExpressions)$'
  -count=1 -timeout 3m`; package execution 163.808 seconds).
- A controlled 1,024-local retention test keeps exactly two entries: a consumed
  resource required for join checks and the future return value. Earlier mentions
  remain retained when a backedge can reach them. Generated graph checks independently
  compare the conservative bounds with actual reachability.

These local observations cover the selected source and verifier boundary, not all
Windows proof groups or hosted same-revision CI. Linux native execution and the
pending observation inventory above remain unconfirmed.
