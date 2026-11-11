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
Copy and Save enable only after successful hashing; a busy clipboard reports a
retryable error. The input path and digest remain read-only text controls with
native selection/copy support. Long filenames ellipsize visually while the path
control retains the complete path and supports native horizontal scrolling.
Common Controls v6 comes from Windows via the embedded manifest; no UI framework,
font, bitmap or additional runtime ships with the EXE. High contrast uses system
colors. Fairy computes layout from DPI/client width and height, including resize
and minimize/restore. A bounded message poll between
64 KiB reads permits repaint/resize and closing during a large hash; this is
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
fairyc test --locked --file tests/sha256.fy
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
actual side-by-side Common Controls loading, two-line digest display, exact
64-character clipboard data, a locked clipboard followed by successful retry,
nonoverlapping minimum/expanded layouts and minimize/restore. Long-filename
checks compare actual visible pixels after maximize with a fresh selection at
the same size, twice through maximize/restore. This catches stale ellipsis
pixels that text reads or a repainting `PrintWindow` capture can miss. The
verifier temporarily makes only its owned app topmost for those pixel reads.
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
