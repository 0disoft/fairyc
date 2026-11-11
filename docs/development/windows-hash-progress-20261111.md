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
