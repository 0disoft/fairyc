# Win32 app build tooling

Shared by Fairy Hash and the standalone desktop generator. These are repository
build helpers, not a new compiler/SDK API or a runtime dependency.

- `settings.py`: bounded app identity, portable names and local ICO selection.
- `resources.py`: VERSIONINFO, Common Controls manifest, title/icon resources.
- `pe.py`: resource-only mapping, icon decoding and version/architecture checks.
- `portable.py`: guide/license/checksums, validated ZIP and atomic file publication.

The app directory supplies `fairy.toml`, `app.toml`, `native/app.rc`, its ICO and
`distribution/README.ko.txt`. Generated projects also carry `LICENSE`; in-repo
examples can use the repository license. Every shared operation receives the
app root explicitly. `portable.package` / `verify` accept an optional CLI-only
validator that raises on failure and returns extra receipt fields (e.g.
`dict(cliCases=5)`, without the common `ok`, `guiLaunched`, etc. fields).
Checksums detect changed bytes and do not authenticate a publisher. The current
packager supports unsigned images only.

Create a standalone app from any directory using the Go Stage 0 CLI:

```powershell
fairyc new my-tool --uid example.my-tool --template win32
cd my-tool
fairyc check
fairyc test --locked --file tests/domain.fy
fairyc build --locked --release
fairyc package --locked --json
```

Only creation is independent of Python/MSVC/Windows SDK. Native app build needs
Windows x64/MSVC/SDK; Python 3.11+ is used for the resource/packaging scripts.
Edit `app.toml` after creation for custom branding. The default console starter
is unchanged; `--template console` selects it explicitly. Existing destinations
are refused. The generated project's files are embedded in `fairyc`, so the
compiler EXE can be copied alone; it needs no repository checkout at runtime.

The optional Python repository helper also accepts initial title/company/icon
options and an arbitrary new destination (parent must exist):

```powershell
python -B tools/new_win32_app.py C:\projects\my-tool --name my-tool --uid example.my-tool --title "내 도구"
# Optional: --company "회사 이름" --icon C:\assets\app.ico
cd C:\projects\my-tool
fairyc check
fairyc test --locked --file tests/domain.fy
fairyc build --locked --release
fairyc package --locked --json
```

The source template is `templates/win32_desktop/`. The generator bundles the
shared Python package, a snapshot of the existing minimal-example IME transport,
an editable app configuration, the default multi-resolution icon and license.
It also bundles `modules/win32_jobs`: copied-buffer async reader/writer transport,
Fairy job/progress/cancel policy and a pure policy test. The GUI file-read example
and `--scan FILE [--cancel-after BYTES]` CLI share that controller; packaging runs
26 app CLI cases including 9 read and 12 save cases. See the generated README for lifecycle,
retirement bounds and writer commit responsibilities. No Fairy callback runs on
a foreign worker thread; product chunk processing stays on the Fairy caller.
It refuses existing destinations and publishes only after settings/resources
are prepared. Generated projects own their snapshots; this is not an updater.
The CLI bundle is refreshed from these same canonical inputs with
`python -B tools/bundle_win32_starter.py` (or `go generate ./internal/projectscaffold`).
Source/default/archive checksum tests reject stale bundle contents. The
repository helper is still available; it is not called by `fairyc new`.

To validate creation, an independent native build, extracted CLI/resources and
overwrite/invalid-setting rejection without interacting with the desktop:

```powershell
python -B tools/verify_win32_starter.py --compiler C:\tools\fairyc.exe
python -B tools/verify_win32_cli.py --compiler C:\tools\fairyc.exe
```

No GUI, Narrator, clipboard, mouse, keyboard or monitor automation is run. New
native UI wiring needs a separate manual check; CLI receipts do not certify it.
The CLI verifier copies only the compiler to an external temporary directory,
creates with empty PATH, verifies invalid/existing-target rejection and console
compatibility, then checks/builds/packages the generated app with normal build
tools. It runs only the app's CLI and resource inspection.
Canonical Hash worker snapshots are checked/refreshed using
`python -B tools/sync_win32_jobs.py --check` / `python -B tools/sync_win32_jobs.py`.

With ADR 0554-capable Go Stage 0 CLI, `fairyc build` prepares desktop resources
and `fairyc package [--locked] [--json]` builds release and validates the local
unsigned portable ZIP. Python helpers remain available for older compilers.
The CLI still requires Python 3.11+ and MSVC/SDK; generated EXEs do not.
