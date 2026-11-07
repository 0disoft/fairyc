# Fairy Windows graph cancellation, 2026-11-07

Status: actual builder, runner and persistent graph token plumbing implemented;
the verified scopes below do not complete the broader native-state/CLI queue.

The builder worker owns its authored module/new test. The execution-test worker
owns independent named-event/pre-I/O/default tests. The coordinator owns runner
and persistent integration, full old persistence regression, complete-source
checks, review, documentation and commit. No Go executor or fixture result
substitutes for actual Fairy/graph C/local-store product execution.

## API and behavior

`build_cancel` and both `run_cancel` functions borrow the explicit Windows Token
before the final mutable work argument. Existing `build`/`run` functions delegate
with none. Invalid options and overdrawn work still fail first. Valid pre-canceled
calls retain work, counters and durable bytes and perform no new stage I/O.

The builder passes the token to every real compiler/linker capture and checks
between source preparation, scratch stages, commands, image admission and final
return. The runner checks before request work, scratch and response admission,
then uses the actual cancellable child capture. Cancellation is distinct from
timeout, compiler/process failure and response rejection, with native causes
retained. Original deadlines, source/output bounds and scratch ownership remain.

Persistence checks before/after snapshot and detachment, forwards the token to
the actual runner, and checks successful commit results before copying final
mutations and before CAS. A post-run checkpoint failure retains the real
Admitted owner. New stage-9 causes preserve cancellation separately from storage,
runner and runtime causes. There is no poll after entering the one-attempt
provider CAS: a concurrent signal cannot hide known publication or cause replay.
Signals do not interrupt synchronous filesystem operations already in progress.

## Windows evidence

Builder worker command passes **142.398s**, actual low/optimized pre-cancellation,
none, existing builder behavior and five sealed-owner refusals:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphWindowsBuild(Cancellation|SealedOwner)?$' -count=1 -v -timeout=12m
```

The pre-canceled builder runs against an unusable host temporary location and
retains its starting work counter. Compiler/profile/stack/budget errors retain
their existing precedence. The none path produces real admitted COFF/PE bytes;
the existing builder matrix retains diagnostics/scratch checks. Log:
`.cache/windows-build-cancel-worker.log`.

The independent event integration runs the two profiles separately to avoid
repeating a valid low observation. Low passes **30.560s**, optimized **43.451s**:

```powershell
go test ./internal/cli -run '^TestCandidateStateGraphWindowsCancellation$/low$' -count=1 -v -timeout 12m
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateStateGraphWindowsCancellation$/optimized$'
```

Each profile performs **7 runner + 19 persistent cancellation/admission checks**,
**52 checks total**, on a real sealed Built and real local store. The independent
Go event is initially signaled. Work/counters remain unchanged, no admitted or
runner owner is fabricated, durable bytes remain identical and Go reopens
version 7. A separate default-API control builds/runs and performs real CAS while
that same external event remains signaled; independent Go reopen sees exactly
version 8/write count 1 and preserved unknown binary state. Both admitted owners
survive source/asset destruction and another native execution. Logs:
`%TEMP%/fairyc-graph-windows-cancel-low.log`,
`.cache/state-graph-cancel-optimized.log`.

The coordinator's complete existing **20-scenario low/optimized** persistence
matrix passes in **82.766s** without selected skips, preserving commit/reject/
trap/helper/borrow/map, no-op/MAX versions, real rename refusal, runtime write
limits, owned results and independent Go state comparison:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateStateGraphWindowsPersistent$'
```

Log: `.cache/state-graph-cancel-persistence-regression.log`.

The separate boundary test passes **72.343s**, low **14.00s** / optimized
**46.48s**, **three scenarios/profile**:

```powershell
go test ./internal/cli -run '^TestCandidateStateGraphWindowsCancellationBoundary$' -count=1 -v -timeout 12m
```

| Actual notification boundary | Snapshot/run/CAS calls | Observed result |
| --- | --- | --- |
| Snapshot lease acquired, second successful lease | 1/0/0 | Cause 9/1009, no Admitted, durable bytes identical, Go reopen version 7 |
| Runner scratch removed, second successful cleanup | 1/1/0 | Cause 9/1009, actual owned Admitted retained and semantically checked, durable bytes identical, Go reopen version 7 |
| CAS lease acquired, third successful lease | 1/1/1 | Event signaled before API return, acknowledged/consistent success, exact admitted writes and Go reopen version 8 |

The test renames exactly one real function definition in a native support copy
and appends a notification wrapper. Each wrapper calls the original exactly
once, preserves its returned result and GetLastError, and only signals the
independently Go-owned manual-reset event. It supplies no file bytes, metadata,
provider result or graph response. Reversing that rename/removing the observer
restores the original support byte-for-byte. The authored import closure,
emitted C header/units, graph runtime/frame assets and product source hashes
remain identical. No sleep or scheduling probability chooses the boundary.
The snapshot event is raised at lease acquisition, then observed after the
actual snapshot completes. The final pre-CAS token checkpoint is not itself
forced by this observer. Log:
`%TEMP%/fairyc-graph-windows-cancel-boundary.log`.

An initial boundary probe failed type checking with E1006 before native execution;
the fixture type was corrected before the complete passing run above. No product
source was changed to obtain the boundary observations.

## Source and remaining scope

Fresh isolated `check` / `check --locked` pass in **15.776s / 14.350s**,
**750 source files / 9,452 MIR functions**, empty stderr. All **1,220 authored
files** match `.cache/state-store-all-ebdcf5c0`; no pending source is excluded.
Only its successful lock is copied back. Record:
`.cache/state-store-all-check.json`. Scoped Go vet, gofmt and diff hygiene pass.

Runtime 1010 token-observation failure is structurally mapped, not forced by
invalid-handle injection. Already admitted reject/trap/error precedence is
retained; cancellation may win at the runner's pre-decode checkpoint. Windows
live job-tree cancellation is the ADR 0540
capture observation; these graph fixtures do not claim a separate delayed
compiler/graph kill. Token checkpoints do not install a console handler or
connect an actual authored developer command. Full current-source Go/Fairy,
self-build, distribution/clean-machine, combined Linux and same-commit hosted
acceptance remain open. Public CLI/SDK/native ABI, DB/provider, dependencies,
CI/runners and repository policy are unchanged. No push, dispatch or release.
