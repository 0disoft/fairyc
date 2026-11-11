# Fairy Hash

A small file SHA-256 utility. File reads, hashing, UTF-8 JSON formatting and
prepared report publication are written in Fairy C. The private SHA algorithm
is reused from `lexer_candidate`, with a SHA-256 length-overflow bound rather
than that compiler example's 96 MiB source limit. Reads use a reusable 64 KiB
buffer, independent of input size.

On Windows x64 with MSVC/Windows SDK, launch without arguments for the native
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
.\build\win32-hash.exe --hash C:\data\input.bin C:\data\result.json
python -B verify_cli.py
fairyc test --locked --file tests/sha256.fy
```

The last path is optional. CLI stdout is the same UTF-8 JSON written to disk.
Failure returns exit 1 and `{"ok":false,"errorCode":N}`. Save writes/syncs/closes
an exclusively created sibling, then publishes with `fs::replace_file`; it
does not truncate the source, its hard-link aliases or existing temporary names.
This is not a power-loss durability or concurrent-file snapshot guarantee.
