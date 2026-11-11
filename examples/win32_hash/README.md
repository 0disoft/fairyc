# Fairy Hash

A small file SHA-256 utility. File reads, hashing, UTF-8 JSON formatting and
prepared report publication are written in Fairy C. The private SHA algorithm
is reused from `lexer_candidate`, with a SHA-256 length-overflow bound rather
than that compiler example's 96 MiB source limit. Reads use a reusable 64 KiB
buffer, independent of input size.

On Windows 10 1703+ x64 (PMv2) with MSVC/Windows SDK, launch without arguments for the native
GUI. **파일 열기 (Alt+O)** selects a file using the actual Windows dialog. Fairy
calculates/displays its SHA-256 and byte count; **결과 저장 (Alt+S)** opens the
native save dialog and writes the same UTF-8 JSON as the CLI. Cancelling either
dialog preserves the previous result. Save is disabled until a hash completes.
The warm, compact interface uses a 600 × 280 logical-pixel client area (minimum
520 × 276), system fonts, a filename/path/size summary and a two-line monospaced
digest. In larger windows the form is centered and capped at 720 × 280 logical
pixels; the digest stays 70 logical pixels high with nearby Copy/Save actions.
**복사 (Alt+C)** copies the original 64 hex digits without line breaks.
While hashing, processed/total bytes, percentage, a thin native progress bar and
HH:MM:SS elapsed time update at most every 100ms. Time uses a monotonic clock.
Open becomes **취소 (Alt+X)**: cancelling keeps the window, shows the partial
progress/time and restores Open. Partial results cannot be copied or saved;
the next file starts a fresh job. Progress reaches 100% only after successful
final hash/size validation. Successful results retain elapsed time.
Copy and Save enable only after successful hashing; a busy clipboard reports a
retryable error. The input path and digest remain read-only text controls with
native selection/copy support. Long filenames ellipsize visually while the path
control retains the complete path and supports native horizontal scrolling.
When a filename or path does not fit, hovering shows its complete Unicode text
in a native, wrapping tooltip. It updates for each selected file and window
size; short text that already fits has no tooltip. Ampersands are literal.
Common Controls v6 comes from Windows via the embedded manifest; no UI framework,
font, bitmap or additional runtime ships with the EXE. High contrast uses system
colors. Fairy computes layout from DPI/client width and height, including resize
and minimize/restore. A bounded message poll between
64 KiB reads permits repaint/resize, cancellation and closing during a large hash; this is
cooperative processing on one UI thread, not a background worker.

The C bridge only creates native controls, forwards events, opens dialogs and
converts UI text, draws small native decorations and transfers clipboard data.
It never reads file contents, hashes data or writes a report. Fairy owns the
filename/byte labels, digest line breaks, status colors and Copy action policy.
There is no private automation socket/stdio protocol or bundled UI framework.

```powershell
fairyc build --release
.\build\win32-hash.exe
.\build\win32-hash.exe --hash C:\data\input.bin C:\data\result.json | Out-String
python -B verify_cli.py
python -B verify_gui.py
python -B verify_gui.py --screenshot build/fairy-hash.png
python -B verify_gui.py --desktop-hover
python -B verify_gui.py --job-mib 64
python -B verify_gui.py --job-only --job-mib 256
python -B verify_gui.py --usability accessibility
python -B verify_gui.py --usability display
python -B verify_gui.py --usability display --high-contrast
python -B verify_gui.py --usability keyboard
python -B benchmark_cli.py --mib 64
fairyc test --locked --file tests/sha256.fy
fairyc test --locked --file tests/progress.fy
```

The last path is optional. CLI stdout is the same UTF-8 JSON written to disk.
Failure returns exit 1 and `{"ok":false,"errorCode":N}`. Save writes/syncs/closes
an exclusively created sibling, then publishes with `fs::replace_file`; it
does not truncate the source, its hard-link aliases or existing temporary names.
This is not a power-loss durability or concurrent-file snapshot guarantee.

This is a GUI-subsystem executable. CLI JSON uses inherited stdout: a capturing
caller such as Python/Codex or a PowerShell pipeline can read it without a new
console window. No Python/runtime/framework is required to run the EXE itself.

`verify_cli.py` checks 19 real hash/save/error cases against Python SHA-256,
including binary/empty input, Unicode paths, replacement, hard-link protection
and exclusive temporary names. `verify_gui.py` opens the actual Windows
Open/Save dialogs and controls only its own process using native messages. It
checks three reference files, paths longer than 127 characters, cancellation,
overwrite confirmation, input protection and closing a 64 MiB job. It also checks
one completed 16 MiB job (or `--job-mib 16..256`), independent percentage/time
observations, GUI-thread ping/resize latency, two actual Cancel button clicks,
removal of cancelled inputs and successful reuse of the same window. The local
fixture bounds are 500ms for ping/cancellation and 1s for programmatic resize;
these are verification bounds, not guarantees for stalled devices or OS reads.
`benchmark_cli.py` measures process wall time without a GUI observer on one
materialized local-file fixture and checks Python's SHA-256 reference. Optional
`--baseline <exe>` and `--reverse` allow paired measurements in either order.
It includes process startup and filesystem caching; it does not measure disk
bandwidth. Benchmark/GUI fixtures and owned processes are cleaned up.
The GUI verifier additionally checks
actual side-by-side Common Controls loading, two-line digest display, exact
64-character clipboard data, a locked clipboard followed by successful retry,
nonoverlapping minimum/expanded layouts and minimize/restore. Long-filename
checks compare actual visible pixels after maximize with a fresh selection at
the same size, twice through maximize/restore. This catches stale ellipsis
pixels that text reads or a repainting `PrintWindow` capture can miss. The
verifier keeps only its owned app exposed for the measured sequence, then closes
it. Visible captures use the composed desktop rather than the parent's backing
DC, and filename comparisons wait for native dialog composition to settle.
Seven checkpoints cover initial paint, two maximize/restore cycles, source
overwrite refusal, two final sizes and restoration. They compare background
pixels of the document icon and byte label with the canvas without requesting
a repaint; occlusion is a failure rather than a successful pixel observation.
Fairy finishes each complete layout with one native full-child redraw. Returning
from an Open/Save dialog also erases and redraws all owner controls, including
on cancellation. This hardens repainting; the reported late black rectangles
were intermittent and their original cause has not been confirmed.
Default tooltip checks use standard Common Controls messages on the existing
registered tool and read the actual popup. `tooltip_probe.py` temporarily enables
tracking and restores the exact original TOOLINFO; text comes from the production
callback, with no injected text/path/hash or app-specific command. Tooltip checks
need no physical pointer movement. Visible paint checks require an exposed,
unlocked desktop. `--desktop-hover` additionally
requires actual pointer hover on an unobstructed desktop and restores the pointer
at completion. Default dialog/tooltip checks inject no mouse button or keyboard
events; `--usability keyboard` deliberately uses real SendInput. Results explicitly
report `tooltipInput` and `desktopHoverVerified`. Fixtures include 103-character
Unicode filenames and longer paths with literal ampersands, beyond the tooltip
notification's 80-character inline buffer.
`--job-only` exercises the actual native Open dialogs, hash job, percentage/time,
resize, two cancellations and window reuse without foreground activation or
pixel inspection. It reports `actual-win32-hash-job`, `pixelInspection:false`
and `physicalInput:false`; screenshots/desktop hover cannot be combined with
that mode. The Go integration can select it with `FAIRYC_HASH_JOB_ONLY=1` when
an input helper prevents foreground ownership. This does not certify the full
visual/clipboard/Save-dialog suite.
Clipboard checks
eagerly copy/restore available clipboard data handles (including text and
binary formats), without retaining a live OLE clipboard wrapper. Optional `--screenshot`
uses `capture_window.py` and Windows/GDI to capture the actual `sample.txt`
result to PNG, without extra Python packages. It does not
inject precomputed hashes/paths into a private app protocol. The verifier knows
classic and Windows 11 filename controls and Korean/English confirmation labels;
other dialog locales are not certified.

Errors: 1=arguments/path text, 2=read/close, 3=kind/observed size mismatch,
4=SHA length/state bound, 5=allocation/UI failure, 6=report save, 7=source/alias
overwrite refused, 8=GUI cancellation. Save failures remain interactive in the
GUI. Cancel-by-close is observed between reads; it cannot interrupt a stalled
OS read. Concurrent same-size writes can change the byte stream being hashed.

The [implementation and execution record](../../docs/development/windows-hash-utility-20261111.md)
contains exact compiler/profile sizes, verification commands and limits.
The [warm UI execution record](../../docs/development/windows-hash-warm-ui-20261111.md)
records the theme/layout/copy changes and their size difference.
The [progress execution record](../../docs/development/windows-hash-progress-20261111.md)
records large-job timings, responsiveness, cancellation and build size.

Usability checks emit `actual-win32-hash-usability` and close their owned window.
`accessibility` reads native Windows UI Automation roles, accessible names,
read-only values, progress range, action states and access keys. It also checks
native forward/reverse tab order in initial/completed/busy states; this is not
physical keyboard input or a screen-reader voice test. Path, digest and progress
names come from Fairy through standard IAccPropServices annotations. OLE32 is a
system DLL; the verifier and its Python runtime are not bundled with the app.

`display` enumerates real monitors, moves only its own window, verifies PMv2,
actual DPI, bounded digest geometry and font scaling, then restores its position.
It reports a skipped physical transition when no different-DPI monitors exist.
`--high-contrast` additionally toggles the real OS high-contrast setting, compares
system colors with PrintWindow-rendered canvas/text pixels and restores original
flags/colors and the app palette. A separate watchdog restores settings if the
controller exits or stalls. High contrast is opt-in; no monitor driver or
persistent scale preference is changed.

`keyboard` requires stable foreground and released user keys/buttons. It exercises
Tab/Shift+Tab loops, Enter on Open/Copy/Save/Cancel, Alt+O/C/S/X and Escape from a
native Save dialog. File paths are populated through native dialog controls;
these checks certify actual navigation/action keys, not typed path text or IME.
Foreground denial/loss is a failure, never a synthetic-key fallback or a pass.
`all` combines the three modes, including physical input; usability modes cannot
be combined with job-only, screenshots or desktop hover.

The existing opt-in Go desktop integration can select these checks instead of
the full visual dialog suite, while retaining 19 CLI cases in both profiles:

```powershell
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_USABILITY='accessibility,display'
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

Unset `FAIRYC_HASH_JOB_ONLY` for usability checks. Optional
`FAIRYC_HASH_HIGH_CONTRAST=1` adds the system contrast toggle only to display/all;
it is never enabled by default. Select `keyboard` or `all` only on a desktop where
the verifier can keep foreground. The [usability execution record](../../docs/development/windows-hash-usability-20261112.md)
distinguishes passes, physical-input failures and monitor limitations.
