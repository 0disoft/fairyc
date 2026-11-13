# Fairy Hash app identity settings — 2026-11-13

## Result

`examples/win32_hash/app.toml` now supplies product name/window title, company,
description, distributed executable name, package prefix, assembly name and icon
path. `fairy.toml` remains the only version source; version **0.1.0** is unchanged.
The existing `Rodisoft` default is editable, including an empty company string.
No new publisher was selected or authenticated by this work.

`app_settings.py` validates those fields before resource generation. Strings
reject control characters and fit bounded UTF-16 lengths. Product names and
descriptions support Unicode, quotes and ampersands. Resource strings use the
[Windows RC doubled-quote convention](https://learn.microsoft.com/windows/win32/menurc/stringtable-resource).
EXE names cannot contain directories/spaces or reserved DOS device names; package
names are lowercase ASCII slugs; icons use existing local relative ICO paths.
Unknown/missing settings fail rather than silently falling back to a hardcoded
identity.

`prepare_resources.py` now generates VERSIONINFO, icon binding, assembly name and
string resource 101 from that configuration. The native window uses LoadStringW
for its initial title. There is no new generated C header or runtime config file
to ship. Normal builds consume the checked-in RC/ICO; after changing identity,
run preparation or use the packaging command's existing prepare/build path.

Package creation and verification derive the EXE filename, ZIP prefix and guide
content from the same settings. The internal compiler output name remains
`build/win32-hash.exe`. End-user checksums name the configured executable.
The guide uses quoted PowerShell invocation syntax. Package contents stay four
files: executable, guide, original license and checksums.

## Executed checks

```powershell
# From examples/win32_hash
python -B prepare_resources.py
& '../../.cache/windows-hash-save-20261112/fairyc-go.exe' build --locked --release
python -B verify_resources.py
# From repository root
python -B examples/win32_hash/package_release.py --exe examples/win32_hash/build/win32-hash.exe
python -B examples/win32_hash/verify_app_settings.py --compiler .cache/windows-hash-save-20261112/fairyc-go.exe
```

Passed:

- Default locked release build, version fields, all nine Windows-decoded icon
  sizes, manifest identity and the configured window-title string resource.
- Default ZIP contents/checksums and 19 real CLI hash/report/error cases on its
  extracted EXE, under a Korean/space-containing temporary directory.
- A separately built temporary copy with title `검증 "해시" Ω`, blank company,
  quoted Korean description, different icon bytes/path, `BrandHash.exe` and
  `brand-hash-0.1.0-windows-x64.zip`. Actual PE strings, title resource, icon bytes,
  guide, ZIP names, checksums and 19 packaged CLI cases all match that identity.
- An invalid `../escape.exe` setting is rejected before changing the prepared RC
  or previously created ZIP. Original source settings, manifest, RC and ICO stay
  byte-identical after the custom-copy test. Temporary output is removed.
- Python syntax parsing and `git diff --check`.

Receipts: `.cache/windows-hash-app-config-20261113/default.json` and `custom.json`.

## Artifact

Current default package:
`examples/win32_hash/dist/fairy-hash-0.1.0-windows-x64.zip` plus `.zip.sha256`.

| Artifact | Before | After |
| --- | ---: | ---: |
| EXE, same compiler/release profile | 196,096 bytes (191.5 KiB) | 196,608 bytes (192 KiB) |
| ZIP | 105,640 bytes | 105,706 bytes |

The EXE increases by **512 bytes** for resource-driven title loading/string data.
EXE SHA-256: `3eaf7c38badea3c1a5305adb20178f1b46e9a134a0880510d330a735cb2fc797`.
ZIP SHA-256: `b845469f76dc78accc2e0ee90a906765699f4ff482e963654384893da7b01d85`.

No GUI, Narrator, clipboard, mouse, keyboard, high-contrast or physical DPI test
was run. Title data/loading is verified through resource-only mapping; the
actual visible title bar remains untested in this slice. General compiler,
bootstrap, unchanged SHA differential and hosted CI suites were not rerun.

No public compiler/language/SDK API, DB, normative source-of-truth, dependency,
CI workflow or runner changed. The concurrent docs-site work was untouched.
Only local commits and an unsigned local ZIP were produced; no push/publication.
