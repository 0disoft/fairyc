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
