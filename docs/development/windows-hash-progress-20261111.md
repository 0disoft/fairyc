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
