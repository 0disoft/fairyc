# Fairy Hash large-job progress and cancellation, 2026-11-11

Owner: `examples/win32_hash`. Layout, percentage/time formatting, hashing and
cancellation policy belong to Fairy; the private C bridge performs Win32 calls.

## Baseline

Retained authored release `359ee46ab4749761b08a5ad2e97032d76730dab0c4558b462b0ce9ad4d881561`,
178,688 bytes. A materialized local 64 MiB repeating 0..255 fixture was checked
against Python SHA-256 (`281e519df3077b557c6b03f5da83c4e8d397219259615dd7c3308f89cae8f2a6`).
CLI wall time, including process startup: **10.904s, 5.869 MiB/s**.

`verify_gui.py --job-mib 64` passed the complete GUI checks plus this large job.
Observed busy time after dialog acceptance: **10.207s, 6.270 MiB/s**; this excludes
dialog closing/initial processing and is not directly interchangeable with CLI
wall throughput. 242 real GUI-thread WM_NULL samples: p95 **10.115ms**, max
**13.099ms**. Native programmatic resize and Fairy layout: **51.310ms**.
These are one-run local measurements with filesystem caching, not disk bandwidth
or a statistical performance guarantee. The dialog driver chooses the actual
fixture; the application receives no injected digest or private test command.

The measured local-file responsiveness supports keeping bounded 64 KiB chunks.
A stalled synchronous OS read remains a limitation: messages and cancellation
are serviced after the current read/hash chunk returns. Physical resize dragging
and network-device stalls are not covered by the programmatic resize metric.

Receipts and retained baseline image: `.cache/windows-hash-progress-20261111/`.

## Progress display

Fairy reads the initial size from the open file handle, renders processed/total
bytes, an integer percentage, a native thin progress bar and HH:MM:SS elapsed
time. It uses monotonic GetTickCount64 through a clock-effect private bridge.
Updates are throttled to 100ms; the final display retains elapsed time. Active
work is capped at 99%, with 100% reserved for successful finalization and size
validation. A different final byte count is rejected as error 3 for GUI and CLI;
same-size concurrent modifications are still not a snapshot guarantee.
Percentage thresholds avoid multiplication overflow even at SHA's length bound.

The authored release passed check/build, all 19 CLI cases, the full GUI verifier
with a 64 MiB reference job, advancing/coherent percentage/time samples and final
100%, plus the focused percentage boundary/large-integer unit test (1/1).
An attempted Buffer-formatting interpreter test exceeded the authored semantic
interpreter subset and was replaced by actual GUI elapsed-time observations;
that failed attempt is not counted as a pass. Elapsed wall agreement is checked
within two seconds because the observer starts after native dialog acceptance.

## Cancel while keeping the window

Fairy swaps Open for an enabled Cancel (Alt+X) during computation. Native code
only latches its real button event; close has priority and pending resize/DPI
events survive cancellation. Fairy abandons the owned file/hash state, preserves
the window, leaves partial progress visible, disables Copy/Save and restores
Open/focus. A new job clears partial state and resets progress. WM_CLOSE still
terminates the window cooperatively.

The authored release passed 19 CLI cases and the complete GUI suite with a
64 MiB job, two independent 16 MiB cancellations and a successful `abc` job in
the same HWND afterward. Cancellation response was **22.718ms / 23.298ms**,
with the cancelled fixtures removed and no partial digest available. Native
control enable/show operations are sequential; the observer waits for the full
state transition instead of treating the first disabled Open query as readiness.
The initial premature observer check was not counted as a functional failure or
successful verification. Working and cancelled desktop captures were inspected.

Final GUI observed busy time: **11.507s** on 64 MiB (5.562 MiB/s including
observer pressure). 215 GUI pings: p95 **11.835ms**, max **16.135ms**; resize
**50.474ms**. Percentage/time snapshots agree with independently calculated
processed/total ratios and observed elapsed time. Progress snapshots are sampled
every 150ms so cross-process reads do not dominate the workload. This differs
from the baseline observer load; CLI paired measurements separately measure
whole-process throughput without GUI observers.

## Separate CLI orchestration and final native verification

Paired 64 MiB CLI checks initially exposed a slowdown in the combined GUI/CLI
orchestrator: baseline 12.184/12.496s versus new 15.036/15.736s in opposite
execution orders. The CLI now has a small filesystem-only orchestrator in
`hash_file.fy`; GUI progress/event handling remains in `application.fy`. Both
still call the same read, finish and SHA implementation. No compiler or SHA
algorithm was changed. A subsequent pair measured baseline 14.817s versus
separated CLI 11.613s. Host variation is visible across runs; this is evidence
for retaining the simpler CLI path, not proof of a compiler-level root cause
or a statistical speedup claim. `benchmark_cli.py --baseline <exe> --reverse`
allows the same independent-reference comparison to be repeated.
The final reverse pair measured separated CLI **12.044s / 5.314 MiB/s** and
baseline **10.392s / 6.159 MiB/s**. Across the two final pairs, current CLI was
11.613–12.044s and baseline 10.392–14.817s. The remaining host/order variation
prevents claiming a proven speedup or complete absence of regression. Final
authored CLI reference verification passed all 19 cases after the separation.

After this separation, KrasisSourceWheelCapture (PID 20528) held foreground
ownership. Two full-suite attempts failed at foreground activation before
testing new job behavior; these were not passes. No external helper was killed.
`--job-only` now uses actual native dialogs/control messages without activation,
physical input or desktop pixels. Its distinct report explicitly says
`pixelInspection:false` and cannot be combined with screenshots/desktop hover.
The previous successful full GUI/visual suite and inspected working/cancelled
captures describe the same UI implementation before the CLI separation; those
checks were not falsely rerun/certified in job-only mode.

Final authored-release job-only execution passed a materialized **256 MiB**
reference file, two independent cancellations and reuse of the same window:

| Observation | Result |
|---|---:|
| Observed busy duration / throughput | 73.584s / 3.479 MiB/s |
| GUI-thread ping p95 / maximum (1,204 samples) | 18.930ms / 85.529ms |
| Programmatic resize and Fairy layout | 48.669ms |
| Cancel response, two jobs | 23.570ms / 23.155ms |
| Coherent progress/time samples | 292 |

The independent SHA-256 is
`486cc817b95d853d3c357ff283b204c0144bd255e73fe2deb1389493b257e3c0`.
Progress reaches the next minute correctly and final duration agrees with
observed wall time. Throughput is still modest; preserving responsiveness does
not imply that the portable checked-arithmetic SHA implementation is optimized.

Final Go frontend development/release integration:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_JOB_ONLY='1'
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

PASS **50.477s**, two profile subtests, zero Go test skips. Both run the 19 CLI
cases and a 16 MiB native GUI job, two cancellations and reuse. Pixel/physical
inspection is explicitly omitted, rather than counted as passing. Go development
is 274,944 bytes; its 16 MiB job took 20.615s, ping max 158.671ms, cancellations
35.494/24.518ms. Go release is 170,496 bytes; its 16 MiB job took 0.708s, ping
max 4.297ms, cancellations 24.761/24.287ms. These frontend/profile/observer results
are separate measurements and are not interchangeable with CLI throughput.

The usable authored release `examples/win32_hash/build/win32-hash.exe` is
**186,368 bytes (182 KiB)**, **+7,680 bytes (7.5 KiB)** versus baseline, including
the separate CLI orchestrator. SHA-256:
`c7dd4aff5b76bfc6d3e4a9dcdba85c51751e89606931d554d2bff196581a9a70`.
It still imports only USER32, GDI32, COMDLG32, COMCTL32 and KERNEL32. There is
no bundled framework, font, image or verification runtime in the EXE.

The source lock was refreshed with the reused authored CLI; SDK/compiler pins,
public compiler/language/SDK APIs, DB, dependency policy and CI runners/workflows
are unchanged. Only examples, their private bridge, focused verification and
execution documentation changed. The focused percentage unit test passes 1/1;
the unchanged SHA unit suite/minimal-GUI suite, fresh compiler bootstrap, Linux,
hosted CI and physical DPI/high-contrast/dragging/stalled-device tests were not
run for this slice. Ordinary file chunks are responsive in the measured cases;
a stalled OS read still delays cancellation until it returns. Local commits
only, no push/publication/deployment.
