# Fairy Hash portable release preparation — 2026-11-13

## Result

Local portable package: `examples/win32_hash/dist/fairy-hash-0.1.0-windows-x64.zip`
and its `.zip.sha256` sidecar. Package version remains **0.1.0** from `fairy.toml`.
No version bump, signing, upload or publication was performed.

The product icon is an editable SVG document/# mark, exported to a 6,306-byte ICO
with 16, 20, 24, 32, 40, 48, 64, 128 and 256px PNG frames. The standard-library
renderer is a development tool for this simple vector asset, not a dependency
of the app. Normal Fairy builds use the checked-in ICO/RC without Python.

`prepare_resources.py` derives fixed file/product/assembly versions from the
manifest and supplies ProductName, CompanyName, FileDescription, InternalName,
OriginalFilename and string versions. Current values include `Fairy Hash`,
`Rodisoft`, `FairyHash.exe`, and `0.1.0`. The Common Controls v6 manifest remains.
This uses Windows' [VERSIONINFO format](https://learn.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource).

The native bridge sets the embedded icon for both large and small window-icon
slots at actual DPI, refreshes them on DPI acceptance and destroys its owned
handles on replacement/teardown. It avoids LR_SHARED for differently-sized loads,
consistent with [LoadImage's caching guidance](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-loadimagew).
The existing opt-in Go desktop fixture now copies the ICO alongside the RC.

## Package

ZIP entries are exactly:

- `FairyHash.exe`
- `README.txt`: Korean instructions, version, shortcuts and CLI usage; UTF-8 BOM
- `LICENSE.txt`: unchanged root Apache License 2.0
- `SHA256SUMS.txt`: hashes of the preceding three entries

The sidecar hashes the complete ZIP. It is an integrity checksum, not a signature.
The current packaging template is explicitly unsigned; it rejects a selected EXE
with a certificate table rather than falsely describing it as unsigned. Future
code signing needs its own validation and corresponding distribution wording.

Packaging uses a temporary archive, validates its contents, extracts the EXE into
an owned temporary Korean/space-containing path, runs the real CLI, and only then
replaces the final archive. Fixed ZIP entry timestamps/order make identical input
payloads produce identical archives; compiler-wide reproducibility is not claimed.
The temporary directory is removed after verification. No tests/scripts, source,
PDB, Python, UI framework, font or installer is bundled. EXE resources are embedded.

## CLI validation

Compiler: existing `.cache/windows-hash-save-20261112/fairyc-go.exe`.

```powershell
# From examples/win32_hash
python -B make_icon.py
python -B prepare_resources.py
& '../../.cache/windows-hash-save-20261112/fairyc-go.exe' build --locked --release
# From repository root
python -B examples/win32_hash/verify_resources.py
python -B examples/win32_hash/verify_cli.py
python -B examples/win32_hash/package_release.py --exe examples/win32_hash/build/win32-hash.exe
python -B examples/win32_hash/package_release.py --compiler .cache/windows-hash-save-20261112/fairyc-go.exe
```

Passed:

- Native resource compilation and locked release build.
- Actual x64 PE / Windows subsystem / fixed and string version inspection.
- All nine embedded frames byte-identical to the ICO; Windows LoadImage decoded
  each requested size and GetIconInfo confirmed its dimensions. Handles released.
- Themed assembly version matches `0.1.0.0`.
- Current EXE has no certificate table (unsigned).
- CLI 19 cases on both the built and packaged EXE: reference hashes, UTF-8 report
  creation/replacement, missing paths, failed destinations and source/hard-link
  protection; no GUI invocation.
- Exact ZIP entries, CRCs, file SHA-256 values, original license and instructions.
- Sidecar matches archive; a changed README with a valid ZIP CRC but stale
  checksum is rejected before running the EXE.
- Rebuilding through the packaging command produced the same EXE and ZIP hashes
  as selecting the retained release.
- Python syntax parsing and `git diff --check`.

An initial C compile failed because the Windows headers define `small` as a
macro; the local icon variable was renamed before the passing build. Failed
builds were not counted as passes. The generated 256px icon was visually reviewed
as a local PNG asset, without launching an app window.

Receipts: `.cache/windows-hash-package-20261113/resources.json`,
`cli-resources.json`, `package.json`, `package-build.json`, `checksums.json`.

## Size and hashes

| Artifact | Bytes | KiB |
| --- | ---: | ---: |
| Previous EXE, same compiler/profile | 188,416 | 184 |
| Current EXE | 196,096 | 191.5 |
| Increase | 7,680 | 7.5 |
| Icon source ICO | 6,306 | 6.16 |
| Portable ZIP | 105,640 | 103.16 |

EXE SHA-256: `57cf4a51bcd0b8a049f0cc30975912b9d6c3a7fd58e010c652a37f0aedc6bce4`.

ZIP SHA-256: `11e8f96a9e099a9cc0c803b94e0725b980ed8e9327a73397650fd5ab5560fef1`.

## Not exercised

Per the user's constraint, no GUI, physical input, clipboard, Narrator, screen,
high-contrast or cross-monitor test was run. Resource-only mapping does not run
the EXE entry point; actual title-bar/taskbar appearance and DPI replacement
remain unobserved at runtime. No general compiler/bootstrap or hosted CI run was
needed. The unchanged hash algorithm's larger differential suite was not rerun.

No public compiler/language/SDK API, DB, normative source-of-truth, dependency
policy, CI workflow or hosted runner changed. `.gitattributes` now marks ICO as
binary. Generated distribution artifacts remain ignored; unrelated `docs-site/`
is preserved. Only local source/documentation commits were made.
