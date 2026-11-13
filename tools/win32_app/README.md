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

Create a standalone app from the repository root (parent directory must exist):

```powershell
python -B tools/new_win32_app.py C:\projects\my-tool --name my-tool --uid example.my-tool --title "내 도구"
# Optional: --company "회사 이름" --icon C:\assets\app.ico
cd C:\projects\my-tool
fairyc check
fairyc test --locked --file tests/domain.fy
python -B package_release.py --compiler C:\tools\fairyc.exe
```

The source template is `templates/win32_desktop/`. The generator bundles the
shared Python package, a snapshot of the existing minimal-example IME transport,
an editable app configuration, the default multi-resolution icon and license.
It refuses existing destinations and publishes only after settings/resources
are prepared. Generated projects own their snapshots; this is not an updater.
`fairyc new` remains unchanged; this generator is invoked with Python.

To validate creation, an independent native build, extracted CLI/resources and
overwrite/invalid-setting rejection without interacting with the desktop:

```powershell
python -B tools/verify_win32_starter.py --compiler C:\tools\fairyc.exe
```

No GUI, Narrator, clipboard, mouse, keyboard or monitor automation is run. New
native UI wiring needs a separate manual check; CLI receipts do not certify it.
