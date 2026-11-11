# Fairy Hash

A small file SHA-256 utility. File reads, hashing, UTF-8 JSON formatting and
prepared report publication are written in Fairy C. The private SHA algorithm
is reused from `lexer_candidate`, with a SHA-256 length-overflow bound rather
than that compiler example's 96 MiB source limit. Reads use a reusable 64 KiB
buffer, independent of input size.

On Windows 10 1703+ x64 (PMv2) with MSVC/Windows SDK, launch without arguments for the native
GUI. **Open file...** selects a file using the actual Windows dialog. Fairy
calculates/displays its SHA-256 and byte count; **Save report...** opens the
native save dialog and writes the same UTF-8 JSON as the CLI. Cancelling either
dialog preserves the previous result. Save is disabled until a hash completes.
The input path and digest are read-only text controls with native copy support.
Fairy computes layout from DPI/client width. A bounded message poll between
64 KiB reads permits repaint/resize and closing during a large hash; this is
cooperative processing on one UI thread, not a background worker.

The C bridge only creates native controls, forwards events, opens dialogs and
converts UI text. It never reads file contents, hashes data or writes a report.
There is no private automation socket/stdio protocol or bundled UI framework.

```powershell
fairyc build --release
.\build\win32-hash.exe
.\build\win32-hash.exe --hash C:\data\input.bin C:\data\result.json | Out-String
python -B verify_cli.py
python -B verify_gui.py
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
overwrite confirmation, input protection and closing a 64 MiB job. It does not
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
