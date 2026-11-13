# @PRODUCT@

Fairy C + native Win32 desktop starter. Windows 10 1703+ x64, the Fairy C
compiler and MSVC/Windows SDK are required to build. Python 3.11+ is used for
resource generation and packaging; the distributed EXE needs no Python or UI
framework. This directory can be moved outside the compiler repository.
The Go Stage 0 CLI creates it with `fairyc new NAME --uid UID --template win32`;
that creation step uses embedded assets and needs no Python or repository.

```powershell
fairyc check
fairyc test --locked --file tests/domain.fy
fairyc test --locked --file tests/jobs.fy
python -B package_release.py --compiler C:\tools\fairyc.exe
python -B verify_cli.py
# Launch manually when ready:
.\build\@PACKAGE@.exe
.\build\@PACKAGE@.exe --process "한글 입력" | Out-String
.\build\@PACKAGE@.exe --scan C:\data\input.bin | Out-String
.\build\@PACKAGE@.exe --scan C:\data\input.bin --cancel-after 65536 | Out-String
```

Edit `src/domain.fy` for product behavior. Both GUI and CLI call `transform`;
replace the example CLI verifier as the behavior changes. `src/application.fy`
owns event/action policy; `src/ui.fy` owns labels and DPI-based layout. Control
IDs are 101 label, 102 input, 103 action, 104 read-only result, 105 live status.
The native bridge creates controls and transports events/UTF-8 text; it does
not implement product logic. Native keyboard handling defers to the copied
minimal-example IME guard during composition. Tab/Shift+Tab/Enter/Esc policy
is in Fairy C. This starter supports one IME-attached editable input.

**파일 읽기 (Alt+O)** demonstrates a real async file job: choose a file, count
bytes/LF newlines in copied chunks, show percentage/elapsed time and cancel while
keeping the window. Close abandons the resource without joining native I/O.
Input/action/file-open controls disable while busy; Cancel (Alt+X or Esc) stays
available. `file_job.fy` pumps events without blocking and updates progress at
most every 100ms. Completion is published only after final source validation.
Controls 106/107/108 are file-open/Cancel/progress. Client size is 560x350 logical
pixels, minimum 400x330. Actual visible/keyboard behavior is not verified by CLI.

`--scan FILE [--cancel-after BYTES]` emits newline-delimited JSON progress and a
terminal result, with done/total/percent/elapsedMs/LF counts/errorCode. Exit 0 is
verified completion, 2 is cancellation, 1 is failure. Cancel threshold is checked
at chunk boundaries (up to 64 KiB), and zero cancels before waiting for open/read.
Active/cancelled work stays below 100%; an empty file reaches 100% only when
verified. `--process TEXT` remains the original text example. `verify_cli.py`
checks 14 cases, including 9 async/cancellation/error cases, without a window.

Shared sources `jobs.fy`, `reader.fy`, `writer.fy`, `progress.fy` and native
reader/writer transport are copied from `modules/win32_jobs` during generation.
`jobs::poll` returns an owned copied chunk; consume it in Fairy C, then present
progress. Native I/O can read ahead without borrowing Fairy memory. Cancellation
marks the state and leaving the Reader resource scope requests cancellation;
late completion cannot revive the cancelled job. Retired reads are bounded to
two contexts; a driver that ignores cancellation can delay their availability.
The writer is a low-level prepared-publication transport, not automatic source
protection; the caller must authorize commit, check identity and await cleanup.

The native shell wires per-monitor DPI v2, system high-contrast colors, Segoe UI,
accessible input/result names, polite completion announcements, resource-driven
icons/title and owned handle cleanup. Desktop behavior is not certified by the
CLI test: no GUI, keyboard, IME, Narrator or physical monitor test is run during
creation/packaging. Test those manually when convenient. Long-running work must
use a worker rather than blocking `transform`; the hash example's asynchronous
reader/writer is a separate, more advanced reference.

Edit `app.toml` for title, company (blank allowed), description, EXE/ZIP names and
ICO path. Version lives in `fairy.toml`. After identity/icon edits run
`python -B prepare_resources.py` and rebuild, or use the packaging command.
Replace `native/app.ico` with your own multi-resolution icon (16/32/48/256px).
The default icon and IME transport originate from Fairy C's examples.

The bundled `win32_app/` package is build tooling, shared in the repository with
Fairy Hash; generated copies are standalone snapshots. Regenerating a starter
does not update an existing project. `LICENSE` covers the supplied source;
review licensing and replace the example product text before publishing.
The portable ZIP includes only the EXE, guide, license and checksums. Packaging
checks the actual extracted CLI and PE resources, keeps version unchanged and
does not sign, publish, upload or automate desktop input.
