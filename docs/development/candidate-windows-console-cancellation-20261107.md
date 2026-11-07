# Windows console cancellation, 2026-11-07

Two disjoint workers implement the native console adapter and developer workspace
preparation. The coordinator owns Fairy lifetime/main integration, closed symbol
selection, runtime pin, actual CLI comparisons, current-source checks and commits.
Only the native/CLI part is accepted here; preparation/native build has its own
receipt. The complete implementation objective remains active.

## Native event and registration

```powershell
go test ./internal/candidatehost -run '^TestWindowsConsoleCancellationNative$' -count=1 -v -timeout 4m
```

Worker **5.033s PASS**, Windows MSVC C11 `/W4 /WX`, `STRICT=1`, low/optimized.
Each profile executes five actual isolated children: headless, Ctrl-C, Ctrl-Break,
callback/close race and failed-unregister poison. Each control kind runs 16 times
per profile; each race child performs 512 bind/close cycles with a concurrent
actual handler thread. Collision 170, missing console 6, non-inheritance,
initially clear/latched event, closing the original token before registration,
rebind, duplicate release, original unregister error, repeated-close refusal and
stable process handle counts pass. Close/logoff/shutdown refusal is checked by
direct handler calls; no session shutdown or user-console signal is generated.
Log: `%TEMP%/fairyc-windows-console-cancel.log`.

Authoritative `windows_process.c` SHA-256:
`f110f5cbec81139286434ccc742a6408bcd2bb5ad67852263e181cbdd9633ee9`.
Failed unregister without a cause is normalized to ERROR_INVALID_STATE;
actual native error values remain unchanged. Registration is unpublished and
its duplicate released on failure; future binds are poisoned to avoid handler
accumulation. Console attachment changes while registered remain unsupported.

## Actual Fairy owners and developer main

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidate(WindowsConsoleCancellation|StateCommandConsoleCancellation|StateCommandWindows)$' -count=1 -v -timeout 15m
```

**159.482s PASS**, log `.cache/windows-console-cli-final-03.log`.
The sealed Fairy owner test executes four low/optimized attached/headless
processes, each with 33 binding lifetimes. Exact none-token admission, competing
binding refusal, actual Ctrl-C/Break delivery, inactive headless owner, explicit
consuming close, latched token and stable handle counts pass.

Actual unchanged `tools/dev/fairyc-state/main.fy` is compiled for both profiles.
Four further processes observe actual Ctrl-C/Break before source I/O and return
exit 4 with cancellation diagnostics and no store files. A scheduling-only
observer wraps local-event creation and image-path discovery; its inverse
reproduces exact original support bytes. It changes no source, graph result,
provider, compiler result or storage response. All generated signals remain
inside hidden isolated child consoles. Only test setup clears Ctrl-C ignore.

The existing actual Fairy/Go command comparison reruns on the new main import
closure: **48 paired command observations / 96 processes**, plus admission and
pre-store refusal controls. Both profiles preserve exact JSON, durable bytes,
exit classes and rollback for the six graph fixtures. No Go executor replaces
the authored native execution.

## Runtime selection and current-source integration

The final native source is packaged as 23 authoritative assets, 169,685 bytes,
SHA-256 `c04b6a17dc7c698446d3d148d7bdf6d2d951e9fe0c692f58f639feb17a298159`.
Both closed selectors admit precisely the three new operations. The private
Go selector test passes in 0.055s. The unchanged runtime-packaging probe closure
passes both profiles in 10.22s inside `.cache/windows-console-cli-final.log`:
34 complete emitted-C comparisons against live Go selection and two unknown
operation refusals. That combined command subsequently fails on the then-unfixed
Console field assignment, so it is not counted as an overall passing command.

Fresh complete isolated `check` and `check --locked` pass in **13.923s / 14.400s**,
**755 source files / 9,500 MIR functions**, empty stderr. All **1,225 authored
sources**, including no excluded pending module, match the checked snapshot
`.cache/state-store-all-593d1ea1`; only its successful lock is copied back.
Record `.cache/state-store-all-check.json`, log
`.cache/windows-console-source-check-final.log`. The tools-side actual main is
separately compiled and executed above. Scoped vet, gofmt and diff hygiene pass.

Initial attempts preserve actual failures in `.cache/windows-console-cli.log`,
`.cache/windows-console-cli-final.log` and `.cache/windows-console-owner-final.log`:
the admitted subset needs a checked switch return, pointer fields cannot be
mutated directly, and test-only system imports need explicit test import lists.
The implementation uses a scalar bound flag; no language contract was widened
and no arbitrary system-import bypass was added to the product.

Native Linux stays batched. Developer native build/distribution, forced command
conflict/late-cancellation/broken-output boundaries and final whole-source
Go/Fairy/self-build/clean-machine/hosted acceptance remain separate. No public
CLI/SDK/native ABI, DB/provider, dependency, repository-policy or runner change.
Private adapters and orchestration change as specified by ADR 0544. No push,
CI dispatch, release or deployment.
