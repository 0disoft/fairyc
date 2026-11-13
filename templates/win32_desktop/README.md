# @PRODUCT@

Fairy C + native Win32 desktop starter. Windows 10 1703+ x64, the Fairy C
compiler and MSVC/Windows SDK are required to build. Python 3.11+ is used for
resource generation and packaging; the distributed EXE needs no Python or UI
framework. This directory can be moved outside the compiler repository.

```powershell
fairyc check
fairyc test --locked --file tests/domain.fy
python -B package_release.py --compiler C:\tools\fairyc.exe
python -B verify_cli.py
# Launch manually when ready:
.\build\@PACKAGE@.exe
.\build\@PACKAGE@.exe --process "한글 입력" | Out-String
```

Edit `src/domain.fy` for product behavior. Both GUI and CLI call `transform`;
replace the example CLI verifier as the behavior changes. `src/application.fy`
owns event/action policy; `src/ui.fy` owns labels and DPI-based layout. Control
IDs are 101 label, 102 input, 103 action, 104 read-only result, 105 live status.
The native bridge creates controls and transports events/UTF-8 text; it does
not implement product logic. Native keyboard handling defers to the copied
minimal-example IME guard during composition. Tab/Shift+Tab/Enter/Esc policy
is in Fairy C. This starter supports one IME-attached editable input.

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
