# Fairy Windows cancellation and capture, 2026-11-07

Status: explicit token/capture implementation verified on Windows low/optimized.
Graph build/run/CAS and actual developer command plumbing remain separate.

The parallel worker owns the initial token and then the closed native adapters,
selectors and runtime pin. The coordinator owns capture integration, process
tree/handle oracles, runtime-byte comparison, full source/lock and final commit.
Token/capture execute actual Fairy and the production Windows C host. Independent
Go Windows event/process handles provide the observation and signaling oracle.

## Native observations

The combined command passes all five selected roots in **52.949s**, without
selected skips:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateWindows(CaptureCancellation|CancelNative|CancelSealedOwner|CaptureAuthoredRegression|CaptureOutputLimit)$'
```

`CancelNative` passes both profiles in 6.05s: real Fairy owners and independent
Go owners interoperate across processes; create collision preserves clear and
signaled state, open never creates, signaling remains latched, final-owner drop
removes the object and later fresh creation starts clear. Native name/type errors
retain their original OS cause. Invalid names, suffix boundaries, none and
pre-I/O refusal are covered. Three separate sealed-owner probes refuse external
construction/read/write with the actual private-field diagnostic (0.95s).

`CaptureCancellation` passes **three cases/profile, six total**, in 6.47s.
Already-canceled requests have PID zero and create no child marker; invalid
options retain precedence. For live cancellation, a real C parent starts a leaf;
the leaf flushes both PIDs to disk. Go opens and observes both as live before
signaling the shared event. Fairy returns kind 12 with the actual parent PID;
both retained process handles subsequently observe termination. The handle
count returns to the stable pre-cancellation baseline. Every caller performs two
normal completed launches first and requires the second to preserve its count;
the real cancellation must preserve that same baseline too.

The five original authored capture tests pass both profiles in **14.00s**,
covering trap receipts, binary stdin/output, overflow, timeout job-tree cleanup
and handle counts. The unchanged output matrix passes **25 cases/profile, 50
total**, in **25.36s**, including shared/minimum/exact/one-over limits, unchanged
64KiB defaults and exact 64MiB capture. No broad compiler matrix is repeated.
Log: `.cache/windows-cancel-integration-final.log`.

## Closed host and packaging

The worker's full scoped candidatehost command passes in **13.894s**, including
17 new selector subcases, exact source composition and unknown-name refusal:

```powershell
go test -mod=readonly ./internal/candidatehost -count=1 -timeout=60s
```

The coordinator extends the existing actual Fairy packager matrix with all
three cancellation names and their combination. **13 complete C sources/profile,
26 comparisons**, match live Go scalar/SDK/host bytes exactly. Unknown reset
operations refuse with no output in both profiles. The command passes in
**11.712s** (`.cache/windows-cancel-runtime-pack.log`):

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateHTTPSRuntimePackaging$'
```

The regenerated pack contains **23 assets / 165,116 bytes**, SHA256
`4ecca0968f1931220c219639a43b4a75474a16aa8d940439e12bc0f770d57a7f`.
The generator's subsequent non-update invocation confirms the checked-in pin.
Product imports use only closed private adapters; four explicit Kernel32 imports
in the small native test helper serve only independent error/handle-count oracles.
The helper's new optional import list leaves existing callers unchanged.

## Preparation failures and final source

An initial Go build fails on the typed `WAIT_TIMEOUT` constant; explicit uint32
conversion fixes test code. Initial tiny-module preparation incorrectly requires
a partition header and a probe imports a nonexistent console module. Both test
harness errors are corrected. Direct system cancellation imports then fail the
closed companion contract; product code is moved to the real private adapter and
both selectors/pin are updated, rather than bypassing production admission.

The first live handle baseline precedes the process's first CreateProcessW and
observes **53 -> 60** handles in both profiles. Generated C is inspected and
contains all input/pipe and job-owner cleanup calls. The final fixture explicitly
warms with two ordinary executions and verifies stable counts on the second
normal execution and subsequent cancellation. Earlier exit-1 observations remain
in `.cache/windows-cancel-integration-first.log`, `-second.log`, `-third.log`,
`windows-cancel-handle-diagnostic.log` and `windows-cancel-cleanup-diagnostic.log`;
none is counted as a passing matrix or a proved per-call leak.

Final isolated `check` / `check --locked` pass in **16.253s / 14.285s**,
**750 source files / 9,445 MIR functions**, empty stderr. All **1,220 authored
files** match `.cache/state-store-all-77d76f2b`; no pending source is excluded.
Only its successful lock is copied back. Record: `.cache/state-store-all-check.json`.
Scoped Go vet, gofmt and diff hygiene pass.

## Remaining work

This does not yet expose token parameters through the graph builder/runner/CAS
wrappers, connect an authored state command, install a console handler, certify
token-observation fault injection or run Linux. Complete-source Go/Fairy,
consecutive self-build, distribution/clean-machine and hosted acceptance remain
open. Public CLI/SDK ABI, DB/provider choice, dependencies and CI/runners are
unchanged. Three compiler-private C ABI adapters are added explicitly. No push,
dispatch, release or deployment.
