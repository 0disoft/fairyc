# Fairy Hash

A small file SHA-256 utility. Fairy C owns read sequencing, hashing, UTF-8 JSON
formatting and prepared report publication. The private SHA algorithm
is adapted from `lexer_candidate`, with a SHA-256 length-overflow bound rather
than that compiler example's 96 MiB source limit. Reads use a reusable 64 KiB
buffer, independent of input size.
The compression core uses bounded 32-bit words carried in 64-bit `usize`, explicit
32-bit masks and borrowed schedule/constant views. Arithmetic remains checked;
there is no native hashing library or C hash implementation.

On Windows 10 1703+ x64 (PMv2) with MSVC/Windows SDK, launch without arguments for the native
GUI. **파일 열기 (Alt+O)** selects a file using the actual Windows dialog. Fairy
also accepts one file dropped onto its idle window or passed as its sole command
argument (`FairyHash.exe "C:\data\input.bin"`), including an Explorer drop onto
the EXE. These inputs use the same Fairy C computation/cancellation flow.
Multiple-file drops are rejected; drops are disabled/discarded while busy.
The native drop transport consumes/frees each Shell payload and converts bounded
UTF-16 paths to UTF-8. `python -B verify_drop.py` checks six production decoder
cases without creating a window; actual desktop drops/startup are not exercised.
Fairy
calculates/displays its SHA-256 and byte count; **결과 저장 (Alt+S)** opens the
native save dialog and writes UTF-8 JSON. Cancelling either
dialog preserves the previous result. Save is disabled until a hash completes.
The warm, compact interface uses a 600 × 380 logical-pixel client area (minimum
520 × 376), system fonts, a filename/path/size summary and a two-line monospaced
digest. In larger windows the form is centered and capped at 720 × 380 logical
pixels; the digest stays 70 logical pixels high with nearby Copy/Save actions.
**복사 (Alt+C)** copies the original 64 hex digits without line breaks.
**예상 SHA-256 (Alt+E)** accepts a reference hash and displays a separate match,
mismatch, invalid-input or waiting-for-file message. ASCII hex is case insensitive;
only surrounding ASCII whitespace is trimmed. The field retains its text between
files and is disabled during hashing/saving. A new selection, failed computation
or cancellation cannot reuse a previous match. Valid expectations add
`expectedSha256` and `matches` to the saved GUI report; empty/invalid expectations
leave the ordinary hash report unchanged. Mismatch still permits copying/saving
the actual calculated hash. Matching, normalization and report formatting live in
Fairy C's `comparison.fy`; the C bridge only transports input/events and presents
native controls. The comparison status has a polite live-region annotation;
new match/mismatch states notify accessibility clients, without per-character
invalid-input announcements. This wiring is compiled, but its desktop keyboard,
layout and screen-reader behavior has not been rerun for this change.
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
font or additional runtime ships with the EXE. A small multi-resolution document/#
icon is embedded for Explorer, the title bar and taskbar. High contrast uses system
colors. Fairy computes layout from DPI/client width and height, including resize
and minimize/restore. GUI open/read/metadata system calls run on an owned native
reader thread; a separate cancellation thread also keeps the cancellation API
off the UI thread. Fairy hashes each 64 KiB chunk on the UI thread, processes
events and waits for readiness/messages for at most 16ms. It never waits for a
file operation or a cancellation routine to finish before responding to Cancel
or Close. Elapsed progress keeps updating while reads are pending.

`native/window.c` creates controls, forwards events, opens dialogs, converts UI
text, draws small decorations and transfers clipboard data. `native/reader.c`
performs Win32 file calls and transports copied chunks/metadata through events;
it never hashes data or writes a report. Worker buffers/path storage stay owned
until their workers finish; no Fairy borrow, callback or HWND escapes to them.
At most two reader contexts (each with reader/canceller threads) may remain live.
If both are waiting on unresponsive I/O, another selection gets a retry message;
the window remains usable. Late cancelled results cannot update a new job.
Fairy owns the
filename/byte labels, digest line breaks, status colors and Copy action policy.
There is no private automation socket/stdio protocol or bundled UI framework.

```powershell
fairyc build --release
.\build\win32-hash.exe
.\build\win32-hash.exe --hash C:\data\input.bin C:\data\result.json | Out-String
.\build\win32-hash.exe --compare C:\data\input.bin '<64-hex-expected-SHA256>' C:\data\compare.json | Out-String
python -B verify_cli.py
python -B verify_compare.py
python -B verify_sha.py
python -B verify_gui.py
python -B verify_gui.py --screenshot build/fairy-hash.png
python -B verify_gui.py --desktop-hover
python -B verify_gui.py --job-mib 64
python -B verify_gui.py --job-only --job-mib 256
python -B verify_reader.py
python -B verify_slow_io.py --compiler C:\tools\fairyc.exe
python -B verify_gui.py --usability accessibility
python -B verify_gui.py --usability display
python -B verify_gui.py --usability display --high-contrast
python -B verify_gui.py --usability keyboard
python -B verify_desktop.py
python -B verify_desktop.py --mode narrator
python -B benchmark_cli.py --mib 64
python -B benchmark_cli.py --baseline C:\data\previous.exe --pairs 3 --mib 64
python -B profile_costs.py --compiler <path-to-fairyc.exe> --mib 64
fairyc test --locked --file tests/sha256.fy
fairyc test --locked --file tests/progress.fy
fairyc test --locked --file tests/comparison.fy
```

The last path is optional. CLI stdout is the same UTF-8 JSON written to disk.
Failure returns exit 1 and `{"ok":false,"errorCode":N}`. Save writes/syncs/closes
an exclusively created sibling, then publishes with `fs::replace_file`; it
does not truncate the source, its hard-link aliases or existing temporary names.
This is not a power-loss durability or concurrent-file snapshot guarantee.

`--compare FILE EXPECTED_SHA256 [REPORT.json]` validates the expected value before
file I/O and emits the ordinary hash JSON plus normalized `expectedSha256` and
boolean `matches`. It exits **0 for match, 2 for mismatch, 1 for operational or
argument errors**. A mismatch is a successful calculation and can save a report.
Malformed expected values return `{"ok":false,"errorCode":12}` without touching
the destination. Existing `--hash` JSON and exit codes are unchanged.
`verify_compare.py` exercises 16 real CLI cases: matching, uppercase/whitespace,
mismatch, both report outcomes, invalid lengths/characters/Unicode, early validation,
existing-report preservation, missing input and source protection.

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
bandwidth. `--pairs 1..5` alternates run order and reports median times plus EXE
identity/size, checking every run against the same Python SHA-256 reference.
`verify_sha.py` adds 24 differential cases: padding/read boundaries, zero/FF data,
one million `a` bytes and seeded nonrepeating binary input. Benchmark/GUI fixtures
and owned processes are cleaned up.
`profile_costs.py` copies the example into an owned temporary workspace and builds
a separate profiling entry point with QPC clock calls. It attributes cached file
reads, in-memory Fairy SHA, file+SHA and actual HWND message/progress work, checking
all digests against Python. Component intervals exclude startup/dialogs/close;
they are instrumented wall time, not CPU samples. Its repeating fixture permits
the memory case to reuse one 64 KiB block. No profiler code or extra clock import
is added to the production EXE. Compare production EXEs with `benchmark_cli.py`.
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
`--job-only` exercises native Open/Save dialogs on empty/Unicode/binary files,
saved reports and source/hard-link protection, then the hash job, percentage/time,
resize, two cancellations and window reuse without foreground activation or
pixel inspection. It reports `actual-win32-hash-job`, `pixelInspection:false`
and `physicalInput:false`; screenshots/desktop hover cannot be combined with
that mode. The Go integration can select it with `FAIRYC_HASH_JOB_ONLY=1` when
an input helper prevents foreground ownership. This does not certify the full
visual/clipboard/tooltip suite.
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
overwrite refused, 8=GUI cancellation, 9=GUI reader capacity temporarily exhausted,
10=GUI save cleanup failure, 11=source changed or final source observation failed.
Save failures remain interactive in the GUI. Cancel/Close abandons the job
without waiting for its OS read to return; a pending worker may retain its file
handle/storage until the OS operation finishes. The synchronous CLI uses the
original `std::fs` read/save path, with an additional owned native source guard.

Both GUI and CLI retain read sharing without permitting ordinary write/delete
handles during calculation. Fairy compares size, volume/file ID, creation time,
last-write time and change time before/after the stream. After EOF it also checks
the selected path again while the original handle is protected. Access time is
ignored, since reading can update it. Metadata collection and the GUI's final
path lookup run on its reader worker, preserving cancellation/close behavior.
No second hash pass or per-chunk metadata query is added.

An observed change or failed final path check returns error 11. The GUI leaves
the digest blank, keeps Open available and disables Copy/Save, including when a
previous selection had a valid result. The CLI emits only its error JSON, without
a digest/report and without modifying an existing output. Selecting a stable
file again restores normal calculation/copy/save. This compares observations;
it is not a filesystem snapshot or continuous monitoring after completion.
Coarse/stale metadata or a backend bypassing sharing and restoring all observed
fields may escape detection. Real filesystem behavior governs those limits.

```powershell
python -B verify_source_changes.py --compiler C:\tools\fairyc.exe
```

The source-change verifier uses an isolated named-pipe-gated build. Production
sharing stays in effect for protected/CLI cases. Only `change-*` fixtures in
that test image allow write/delete sharing, so actual same-size overwrites,
restored last-write timestamps, grow/truncate and EOF path replacement can
exercise Fairy's comparisons. It reports this explicitly; production never
relaxes sharing. Nine cases include actual disabled Copy/Save clicks, unchanged
clipboard sequence, no Save dialog, CLI old-report protection and window reuse.
Test gates/sharing overrides are absent from production. See the
[source-change execution record](../../docs/development/windows-hash-source-changes-20261112.md).

GUI report saving also uses an owned native worker and a separate cancellation
thread. Fairy chooses destination validation, exclusive sibling names (32 bounded
attempts), write/sync sequencing, a second source-identity check and final publish.
`native/writer.c` owns copied paths/report bytes and executes one Win32 operation
per request. The completed hash stays visible; Open/Copy/Save are disabled during
saving. Cancel is available before publication and retains the computed result.
An owned temporary is marked with reversible `FileDispositionInfo` deletion;
cleanup closes that handle rather than deleting a path that another process
could reuse. Foreign temporary names are never truncated or removed.

Publication clears the temporary's deletion disposition and renames the same
open handle over the destination. Cancel is disabled before this boundary.
Close before publication requests cancellation; Close during publication allows
the complete result to settle. In both cases the window keeps pumping messages,
painting and resizing until worker cleanup finishes, then exits. There is at
most one save context, including cancelled cleanup; another save cannot overlap
it. A driver that ignores cancellation can delay cleanup/exit indefinitely,
without blocking the UI thread. Cleanup failure is reported as error 10 rather
than success. Forced termination or power loss during disposition/rename is not
covered by normal Close cleanup; no power-loss durability guarantee is made.
OS-native file dialogs remain synchronous. CLI report saving retains `std::fs`.

```powershell
python -B verify_writer.py
python -B verify_slow_save.py --compiler C:\tools\fairyc.exe
```

The writer probe uses production C with MSVC `/W4 /WX`, checking cancellation,
read-only-destination publication failure, complete replacement and zero live
contexts. The slow-save verifier builds an isolated test image with named-pipe
`ReadFile` gates around destination checking, creation, writing, synchronization,
publication and cleanup. It verifies 18 GUI cases including retained cancellation,
Close in three stages, late source-alias detection and foreign temporary names.
Disk-full errors are injected in that build; no real disk is filled. These gates
are real pending OS I/O on an owned thread, not certification of slow physical
disk/SMB behavior. They and their environment variable are absent from production.
`--slow-save` is an internal verifier mode, separate from `--slow-io` and job-only.
Optional `--gate-exe <path>` retains that test-only image for focused reruns.
See the [save execution record](../../docs/development/windows-hash-save-20261112.md).

The [implementation and execution record](../../docs/development/windows-hash-utility-20261111.md)
contains exact compiler/profile sizes, verification commands and limits.
The [warm UI execution record](../../docs/development/windows-hash-warm-ui-20261111.md)
records the theme/layout/copy changes and their size difference.
The [progress execution record](../../docs/development/windows-hash-progress-20261111.md)
records large-job timings, responsiveness, cancellation and build size.
The [SHA performance record](../../docs/development/windows-hash-performance-20261112.md)
records component costs and paired production-EXE speed/size measurements.
The [slow-I/O execution record](../../docs/development/windows-hash-slow-io-20261112.md)
records pending-read cancellation/close, resource lifetime and integration checks.

`verify_reader.py` compiles the unchanged native transport with MSVC and checks
chunk contents/metadata, an actual pending named-pipe ReadFile, cancellation and
zero live contexts after cleanup. `verify_slow_io.py` copies the app into a
temporary workspace and adds test-only named-pipe gates before open/read/metadata.
It observes GetThreadIOPendingFlag and the client's owned PID, then checks nine
GUI cases including elapsed time, cancellation, close, delayed completion and
the two-context limit. The probes use real OS pending I/O rather than a UI sleep;
they do not certify a particular disk/network driver. Test hooks are not linked
into the distributed EXE. `--slow-io` is an internal verifier mode for that build.
Go desktop integration selects it with `FAIRYC_HASH_SLOW_IO=1`, exclusive with
job-only/usability. Both modes retain CLI 19 and differential SHA 24 cases in
development/release; defaults and CI workflows do not enable desktop tests.

Usability checks emit `actual-win32-hash-usability` and close their owned window.
`accessibility` reads native Windows UI Automation roles, accessible names,
read-only values, progress range, action states and access keys. It also checks
native forward/reverse tab order in initial/completed/busy states; this is not
physical keyboard input or a screen-reader voice test. Path, digest and progress
names come from Fairy through standard IAccPropServices annotations. OLE32 is a
system DLL; the verifier and its Python runtime are not bundled with the app.
Status changes are live regions: Fairy requests polite completion/cancellation
announcements and assertive errors. Frequent percentage/time updates turn the
live setting off and do not emit live-region events. The accessibility probe
checks the UIA LiveSetting property and real, PID-filtered WinEvents, including
a deliberately busy clipboard error and silence during progress updates.

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
Tab/Shift+Tab and focused Enter are forwarded to Fairy before the multiline
digest Edit can consume them. Native file dialogs retain Windows key handling.

`verify_desktop.py` runs the same real-key checks on a temporary input desktop.
It briefly switches the screen, preserves/restores clipboard contents, contains
owned workers in a kill-on-close Job Object, and restores the original desktop
in `finally` with a separate restoration watchdog. Other input helpers need not
be terminated. Do not interact with the temporary desktop during the test.
`--mode narrator` attempts Windows Narrator on that desktop and reads its actual
last-spoken phrase through Narrator+Ctrl+X; it does not synthesize speech using
TTS. On this machine Windows rejects that UIAccess launch with error 740.
`speechHistoryTested:false` and `skipReason` explicitly mark the missing speech
test; a successful harness restoration is not a successful Narrator speech test.
No elevation prompt or alternative screen-reader installation is automated.
`narrator_session.py` separately verifies normal-desktop launch using
ShellExecuteExW with the `open` verb, without `runas`, and stops only the process
returned by that launch. This launch smoke passed; it does not certify speech.
The [normal-desktop Narrator record](../../docs/development/windows-hash-narrator-20261112.md)
records partial speech observations and the remaining verification boundary.

See the [keyboard/accessibility execution record](../../docs/development/windows-hash-input-accessibility-20261112.md)
for verified keys, live-region events, physical-display limits and image size.

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

## Portable release, without desktop automation

Python 3.11+ and the existing Fairy C/MSVC/Windows SDK toolchain can prepare a
local release. This keeps the current `fairy.toml` version; it does not bump,
sign, publish or upload it:

```powershell
python -B package_release.py --compiler C:\tools\fairyc.exe
# Or select an existing matching release image without rebuilding:
python -B package_release.py --exe build/win32-hash.exe
python -B verify_package.py dist/fairy-hash-0.1.0-windows-x64.zip
```

The ZIP contains only `FairyHash.exe`, a Korean UTF-8 BOM `README.txt`, the full
Apache 2.0 `LICENSE.txt`, and `SHA256SUMS.txt` for those three files. A sibling
`.zip.sha256` checks the archive. Checksums detect changed bytes; they do not
authenticate a publisher. The current template describes an unsigned build.
Artifacts stay under ignored `dist/`; no installer or Python is distributed.

Packaging synchronizes `native/app.rc` from `app.toml` and `fairy.toml`, builds with
`--locked --release`, then inspects x64/subsystem, VERSIONINFO, the themed manifest
and all nine embedded icon frames. It extracts its ZIP into a temporary path
containing Korean text and spaces and runs the actual hash/report CLI's 19 cases
and comparison CLI's 16 cases plus checksum import/export's 22 cases
before replacing the final ZIP. No window is launched, clipboard read, cursor
moved or key sent. For unchanged payload bytes the ZIP has stable entry order,
timestamps and compression settings; this is not a whole-toolchain reproducible
build claim. Failed verification leaves the previous ZIP intact.

App identity is configured in `app.toml`; no Python/C source edit is needed to
change the product/window title, company, description, distribution EXE name,
ZIP prefix, assembly identity or ICO path:

```toml
[app]
product_name = "Fairy Hash"
company_name = "Rodisoft"
description = "Fairy Hash - SHA-256 file utility"
executable_name = "FairyHash.exe"
package_name = "fairy-hash"
assembly_name = "Fairy.Hash"
icon = "native/app.ico"
```

`Rodisoft` is the existing editable default, not a required publisher name.
Set `company_name = ""` to leave that VERSIONINFO field blank. The app version
still comes only from `fairy.toml`; there is no second version to synchronize.
Window titles/metadata accept Unicode and quotes. EXE names use ASCII without
spaces/directories, ZIP prefixes use lowercase ASCII slugs, and the ICO must be
an existing relative path within the project. Use at least 16/32/48/256px frames.

After editing the settings, rebuild and package:

```powershell
python -B prepare_resources.py
fairyc build --locked --release
python -B verify_resources.py
python -B package_release.py --exe build/win32-hash.exe
```

Or use `package_release.py --compiler C:\tools\fairyc.exe` to prepare resources,
build and package in one command. The internal compiler output remains
`build/win32-hash.exe`; the ZIP contains the configured distribution filename.
Its guide and checksums use that filename too. Settings are embedded at build
time: end users do not need `app.toml` alongside the EXE.

`verify_app_settings.py --compiler C:\tools\fairyc.exe` makes a temporary branded
copy, builds it and checks the generated PE and actual packaged CLI. It covers
a quoted Korean title/description, blank company, alternate ICO path/content,
EXE and ZIP renaming, and invalid-path rejection without changing the original
settings. It launches no GUI or desktop input.

The example's editable SVG is `native/app.svg`; `make_icon.py` regenerates its
16/20/24/32/40/48/64/128/256px ICO using only the Python standard library. That
small renderer supports the example's filled rounded rectangles/polygons, not
arbitrary SVG. Directly replacing the ICO does not require running the renderer.
Normal builds use the checked-in ICO and RC without Python. Resource files and
included assets are native build inputs, outside the Fairy frontend lock.

Window code loads separate owned large/small icons at the current DPI and
refreshes them after a DPI change. It releases old handles after replacement and
at teardown. `verify_resources.py` maps the EXE as resource data and checks actual
Windows icon decoding without executing its entry point; visible window wiring
was not exercised in this release work.

See the [release execution record](../../docs/development/windows-hash-package-20261113.md)
for sizes, checksums and the exact validation boundary.
The [app-settings execution record](../../docs/development/windows-hash-app-settings-20261113.md)
records the configurable build and custom-brand verification.

The [comparison execution record](../../docs/development/windows-hash-comparison-20261113.md)
records the shared comparison policy, CLI checks, updated package and desktop-test boundary.

## Single-entry `.sha256` files

**체크섬 열기 (Alt+L)** loads one expected hash into the comparison field.
Accepted UTF-8 formats are a bare 64-digit hash, GNU `hash  filename` /
`hash *filename`, and BSD `SHA256 (filename) = hash`. UTF-8 BOM, CRLF and outer
ASCII whitespace are supported. Escaped GNU records, UTF-16, multiple entries,
NUL/newline inside a record and files over 64 KiB are rejected (error 13).
The embedded filename is informational: choose the actual input explicitly;
the app never opens paths from the checksum file. GUI reads use the existing
asynchronous transport, process cancel/close/resize and preserve previous input
on failure/cancellation. The GUI wiring has not been exercised on the desktop.

Save with a `.sha256` filename (case insensitive) to export the actual digest as
UTF-8 `hash  basename\n`; other save extensions retain JSON. The native save
dialog defaults to `sha256.json`; type the `.sha256` extension explicitly.
Saving uses the existing asynchronous safe publication and source protection.

```powershell
.\build\win32-hash.exe --checksum C:\data\input.bin C:\data\input.sha256 | Out-String
.\build\win32-hash.exe --compare-file C:\data\input.bin C:\data\input.sha256 C:\data\compare.json | Out-String
fairyc test --locked --file tests/checksum.fy
python -B verify_checksum.py
```

Both final destination paths are optional. `--checksum` writes standard checksum
text to stdout; `--compare-file` emits the same comparison JSON/0/2/1 exit codes
as `--compare`. Checksum contents are validated before hashing the actual input.
Malformed checksum files leave previous destination reports intact.
