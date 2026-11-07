# Actual authored Windows state command, 2026-11-07

The coordinator owns actual `tools/dev/fairyc-state/main.fy` entry/orchestration,
producer-bound Outcome identity, actual Go command comparison and integration.
Two disjoint workers own the retained-root file reader and executed-result JSON.
Only authored Fairy and its actual emitted native program perform product work;
Go commands/compilation and Go storage supply independent test expectations.

## Real file input

```powershell
go test ./internal/cli -run '^TestCandidateStateCommandWindowsFile$' -count=1 -v -timeout 6m
```

Worker **5.834s PASS**, low **2.33s** / optimized **2.79s**; **26 scenarios/profile,
52 total**, containing **224 actual load calls** with stable process handle counts.
Cases cover empty/exact/above bounds, 64KiB chunk boundaries, Unicode/relative
paths, retained `C:\` parent spelling, missing/nonregular/reparse/ADS refusal,
presignaled cancellation, exact/one-short work and unchanged caller path bytes.
Full log: `%TEMP%/fairyc-state-command-windows-file.log`.

Reader errors retain stages 1 input/path work, 2 cancellation, 3 parent open,
4 file open, 5 metadata, 6 data/read/size, 7 file close and 8 parent close.
Codes distinguish invalid/change 1001, bounds/work 1002, allocation 1003,
cancellation 1009, token observation 1010 and original native error 0 + OS code.
The caller's 16MiB cumulative allowance includes path and sentinel, so not every
nominal maximum fits an already partly spent ledger. During-read cancellation,
concurrent mutation, 1010 and close failures were not deterministically forced.

## Executed-result ownership and JSON

The original Outcome retained response and publication but not the producer's
program identity. Its private ProgramIdentity record now retains 32 bytes and
allows `persistent::matches(outcome, prepared)` to refuse a different body even
when layout/UID agree. Report code consumes this proof before metadata emission.
An initial direct array resource field was refused by the existing language
subset; the ordinary-record wrapper follows the existing local-state pattern.
No language/SDK support was expanded for this implementation.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommand(Snapshot|Execute)Report$' -count=1 -v -timeout=7m
```

Worker **100.437s PASS**, execute **65.13s**, existing snapshot **35.13s**,
**14 actual execute + 16 snapshot** scenarios across low/optimized. Actual sealed
Built/Outcome and real local state cover commit, empty commit, reject, trap,
helper/borrow, every read codec, full-width/binary values and MAX no-op. Exact Go
JSON bytes, exact/one-short budgets, owned output after producer/source teardown
and different same-layout Prepared refusal pass. Existing snapshot API/bytes
remain. Log: `.cache/state-command-execute-report-native-01.log`.

Known acknowledged inconsistency is preserved in code, including the exact Go
PublicationReceiptError text; the concrete provider does not expose a fabricated
inconsistency and no such provider/result was injected for these observations.

## Existing persistence and cancellation

Because Outcome's private representation changed, prior native observations are
not reused as evidence for these new inputs. The coordinator reruns:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphWindows(Persistent|Cancellation|CancellationBoundary)$' -count=1 -v -timeout 15m
```

**211.246s PASS**: complete existing **20 persistence** scenarios (**62.63s**),
**52 cancellation/admission** checks plus none-token controls (**55.47s**) and
**six deterministic snapshot/post-run/CAS-entry** observations (**92.94s**).
Go reopen/rollback/unknown state, exact retained response and irrevocable
acknowledgment remain intact. Full log:
`.cache/state-command-outcome-owner-regression.log`.

## Actual process entry comparison

The test reads the actual authored main unchanged, builds it with its real
imports and installs the actual pinned packs adjacent to the test executable.
It independently builds/runs the current Go developer command. Both processes
receive the same source bytes/selected action/profile and independent real stores.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommandWindows$/low$' -count=1 -v -timeout 15m
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommandWindows$/optimized$' -count=1 -v -timeout 15m
```

Low **29.854s PASS**, optimized **49.561s PASS**.
Six graphs each execute init, snapshot, execute and final snapshot against both
actual commands: **48 paired command observations / 96 processes**, plus
asset-free admission and pre-store refusal controls. Every successful comparison requires identical stdout JSON and
durable state bytes, equal exit status and empty stderr. Reject and trap are
observed as the Go command's nonzero transaction exit classes with rollback.
Help and invalid argv work without distribution assets. Missing source, invalid
syntax, module/basename mismatch, private source ceiling and corrupt pin refuse
before creating any file in the store directory. Relative source paths and a
Unicode/spaced parent directory are used. Logs:
`.cache/state-command-windows-low.log`,
`.cache/state-command-windows-optimized.log`.

Initial orchestration scaffolding needed the existing switch return/type subset
and enum switch form. The initial reference fixture also used a non-identifier
basename; actual Go syntax correctly refused it. The command now explicitly
enforces the basename/module rule, with a mismatch refusal test. These initial
pre-integration failures are retained separately in attempt logs; no compiler
policy or Go reference acceptance was weakened to pass the comparisons.

## Integration and remaining scope

Fresh isolated full `check` / `check --locked` pass in **14.187s / 13.489s**,
**755 source files / 9,495 MIR functions**, empty stderr. All **1,225 authored
candidate sources**, with no excluded pending module, match the tested
`.cache/state-store-all-ed58b696` snapshot. Only its successful checked lock is
copied back. The actual tools-side main is separately compiled/executed by the
native command test above. Record `.cache/state-store-all-check.json`, log
`.cache/state-command-windows-check.log`. Scoped Go vet, gofmt and diff hygiene
pass. Linux native execution stays batched at the user's request.
The console-handler gap in this historical observation is subsequently closed
by ADR 0544 and `candidate-windows-console-cancellation-20261107.md`.
This receipt does not claim a supported installed distribution/build recipe, command-level forced conflict/
late cancellation/broken-output coverage, full-source Go/Fairy self-build parity,
clean-machine verification or hosted same-commit acceptance. The full queue is
still active. No push/CI dispatch/release/deployment; no public CLI/SDK/native ABI,
DB/provider, dependency, repository-policy or runner change.
