# Win32 jobs source bundle

Reusable borrowed-buffer-free I/O transport and Fairy C job policy, extracted
from Fairy Hash. This is a vendored source bundle, not a new public SDK package.
Generated apps own copies; there is no worker that invokes Fairy C on a foreign
thread. Native readers/writers copy paths/data and keep their storage alive until
OS work completes. No callback or HWND is retained by a worker.

`reader.fy` / `native/reader.c` expose bounded 64 KiB reads and protected metadata.
`protect`/`sample` are synchronous CLI guard helpers; do not use them on a GUI
thread when the filesystem may be slow. The asynchronous Reader is separate.
`jobs.fy` sequences open/chunks/EOF/final-name validation, counts delivered bytes,
keeps incomplete work below 100%, and makes completed/cancelled/failed states
terminal. Call `poll`, consume a returned chunk in Fairy, update your UI, and
`reader::wait` while pending. `cancel` marks the state; leave the Reader resource
scope to request asynchronous OS cancellation and drop ownership without joining.
A completed chunk belongs to the caller, so the next native read can overlap
Fairy processing. There is no atomic filesystem snapshot guarantee.

`writer.fy` / `native/writer.c` transport one prepared publication operation at a
time. The caller owns source-identity checks, exclusive temporary naming, commit
authorization and completion/cleanup waits. Once commit starts, do not treat a
late Cancel as rollback. Use `publication.fy` for the shared staging/commit policy.
`progress.fy` formats elapsed time and calculates percentages without done*100
overflow. Reader retirement is bounded to two contexts; writer retirement to one.

Canonical files live here. `python -B tools/sync_win32_jobs.py` refreshes the
Hash reference snapshots; `--check` detects drift. Template generation copies
these same files into an independent app, and the compiler's template bundle
must be refreshed after source changes. Real GUI behavior is validated separately
from native/headless CLI checks.

`publication.fy` now owns check → exclusive sibling staging → write → flush →
identity recheck → explicit authorization → rename → cleanup. Poll returns 1
only when the caller must authorize/cancel; 2/4/8 are terminal success/failure/
cancellation after cleanup. `begin(identity,true)` protects an input file and
hard-link aliases. `unprotected()` is for generated data with no input file.
Hold source protection separately when required; metadata is not an atomic
snapshot. Cancellation before authorization preserves the old destination;
once authorized, cancellation cannot undo commit. Temporary collisions are
retried up to 32 names (`.fairy-save-N.tmp`) without touching unowned files.
Payloads are copied, bounded to 1 MiB, and may be empty. Hash's `saving.fy`
now only pumps its own window and presents status around this shared policy.
