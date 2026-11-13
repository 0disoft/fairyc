# Fairy Hash inputs, checksum files and desktop starter — 2026-11-13

## Delivered in order

1. File drop and single-path GUI startup (`67601c54`).
2. Single-entry `.sha256` import/export (`2b109be7`).
3. Shared Win32 build tooling (`d2b17de7`) and a standalone hash-free starter.

The work kept `main`, version 0.1.0, and the existing Fairy C implementation.
No desktop interaction, GUI launch (including hidden windows), clipboard access,
keyboard/mouse input, Narrator or monitor configuration was used for validation.
Concurrent `docs-site/` edits/commits were preserved and excluded from our commits.

## Input paths

An idle window accepts one Shell file-drop payload. The bridge uses
[WM_DROPFILES](https://learn.microsoft.com/en-us/windows/win32/shell/wm-dropfiles)
and [DragQueryFileW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-dragqueryfilew)
to convert one bounded UTF-16 filename to UTF-8, then frees the payload exactly
once with DragFinish. Multiple/empty/oversized/invalid paths are rejected.
Pending drops are freed when entering busy state or closing. Drop registration
is disabled during computation, checksum loading and report saving.

`FairyHash.exe FILE` passes the initial UTF-8 path into the same Fairy C flow as
Open/drop, without a file dialog. An Explorer drop onto the EXE uses this path.
Unknown leading `--` options are rejected; filenames beginning `--` can use a
qualified path. No association/registry change or installer was introduced.

`verify_drop.py` compiles the production decoder into a console-only fixture and
passes real HDROP structures to Shell APIs. All **6 cases** passed: Unicode,
multi-file, empty, small output buffer, malformed UTF-16 and excessive length.
This verifies payload decoding, not an actual desktop drag, WM_DROPFILES dispatch
or visible startup. The new GUI paths are compiled, not exercised.

## Checksum files

Fairy C owns format parsing/formatting in `checksum.fy`. Supported UTF-8 records:
bare hash, GNU `hash  filename` / `hash *filename`, BSD `SHA256 (filename) = hash`.
UTF-8 BOM, CRLF, case-insensitive hex and outer ASCII whitespace are accepted.
The limit is 64 KiB and one entry; escaped GNU records, UTF-16, embedded NUL or
newlines and multiple entries are rejected. Embedded filenames are informational
and never opened automatically. Error 13 means malformed/oversized content.

- GUI **체크섬 열기 (Alt+L)** loads the expected field, keeping the current actual
  digest. The existing native reader transports bytes asynchronously while Fairy
  handles Cancel/Close/resize, bounds content and checks source observations.
  Failed/cancelled loads preserve the previous expected text/result. Busy state
  disables expected entry and checksum loading.
- GUI Save uses GNU UTF-8 `hash  basename\n` when the selected destination ends
  in `.sha256` (case insensitive); otherwise JSON is retained. The dialog defaults
  to JSON, so the checksum suffix must be typed explicitly. The actual hash is
  exported even if an expected value mismatches. Safe async publication and
  source/hard-link protection are reused.
- `--checksum FILE [DEST]` emits/saves the standard text. `--compare-file FILE
  CHECKSUM [REPORT]` validates the checksum before hashing and emits the existing
  comparison JSON with exit codes 0/2/1. Original `--hash` / `--compare` remain.

Passed: one pure checksum-format test, **22 actual checksum CLI cases**, previous
**19 hash/save/error + 16 comparison** cases. The portable ZIP's extracted EXE
also passed all **57 cases** and resource/icon/checksum verification. Tests cover
format variants, match/mismatch, malformed/oversized/multiple records, early
validation, previous-report preservation, source/hard-link protection and no
leaked temporary reports. Updated tab-order/accessibility fixtures were parsed
but not executed; real GUI loading/cancel/save behavior remains unverified.

## Reusable desktop path

`tools/win32_app/` contains shared identity validation, resource generation, PE
inspection and portable packaging, with no Hash domain dependency. Fairy Hash's
thin wrappers retain its defaults and CLI validation. Refactoring reproduced the
same Hash ZIP bytes and exercised a temporary custom-branded build (Unicode/
quoted title, empty publisher, alternate icon/name, **57 packaged CLI cases**).

`python -B tools/new_win32_app.py DEST --name NAME --uid UID --title TITLE`
generates a standalone project from `templates/win32_desktop/`. Existing targets
are never overwritten; input settings/resources are checked in a temporary
sibling before publication. App identity/icon/company are configurable, with
blank company by default. It bundles a snapshot of shared build tooling, the
default icon/license and the existing minimal-example IME guard, compiled with
diagnostics disabled. The starter has no SHA algorithm, hash business logic,
async workers or private desktop automation protocol.

`domain.fy` owns the example transformation shared by GUI and `--process TEXT`;
`application.fy` owns action/keyboard policy; `ui.fy` owns native labels and
layout. `native/window.c` transports events/text and owns native handles. Wiring
includes one IME-attached edit, button, read-only result, status, per-monitor DPI
v2, themed controls, system high contrast, accessible names and polite status
notification. Real keyboard/IME/DPI/reader announcements are not certified.
Long-running product work must be moved off this starter's UI thread; the Hash
reader/writer remains a separate reference. The Python helper does not extend
the compiler's `fairyc new` contract or register a public SDK API.

`verify_win32_starter.py` generated **outside the repository**, then passed:

- Frontend check, locked pure navigation-policy test and locked release build.
- Five extracted EXE CLI cases including Korean/quotes/empty input/unknown mode.
- Resource title/version, blank company, nine icon sizes and portable ZIP checks.
- Six rejected creation cases: existing destination/repeated overwrite attempt,
  reserved name, invalid UID, control characters and missing icon. Existing
  project bytes remained unchanged; failed creation published no destination.

The semantic test interpreter cannot execute the example Buffer-based transform;
that behavior is tested through the actual native CLI. Initial integration found
and corrected a missing shared Path import, a Windows `small` macro collision
and overlapping package-receipt fields before the final successful checks.

## Commands and artifacts

Compiler: `.cache/windows-hash-save-20261112/fairyc-go.exe` (SDK stage0.87,
language/20). Relevant checks, not a general compiler regression/release suite:

```powershell
# examples/win32_hash
fairyc-go.exe check
fairyc-go.exe build --locked --release
fairyc-go.exe test --locked --file tests/checksum.fy
python -B verify_drop.py
python -B verify_compare.py
python -B verify_cli.py --exe build/win32-hash.exe
python -B verify_checksum.py
python -B package_release.py --exe build/win32-hash.exe
python -B verify_app_settings.py --compiler <compiler>
# repository root
python -B tools/verify_win32_starter.py --compiler <compiler>
```

Python syntax parsing and scoped `git diff --check` passed. No broad bootstrap,
unchanged SHA performance/differential suite or hosted CI/runner was run.

| Artifact | Before these steps | Final |
| --- | ---: | ---: |
| Fairy Hash EXE | 202,752 bytes (198 KiB) | 213,504 bytes (208.5 KiB) |
| Fairy Hash ZIP | 109,014 bytes | 113,818 bytes |
| Generated starter EXE | — | 137,728 bytes (134.5 KiB) |
| Generated test starter ZIP | — | 76,608 bytes |

Hash EXE SHA-256: `0b6d7d591123c28b2c97060bd31c8b6c1f5e8f81634ec571ca4316eef7f66349`.
Hash ZIP SHA-256: `742030f8f403bb3dfe13c9a1789b5dbd2f20eaefa9c3d25e04ce3457d2b734ca`.
The Hash ZIP stays at `examples/win32_hash/dist/fairy-hash-0.1.0-windows-x64.zip`.
The retained development starter is under
`.cache/windows-desktop-starter-20261113/standalone`; the independently generated
test project was removed. Its receipt is
`.cache/windows-desktop-starter-20261113/starter.json`. Files remain local/ignored.
The final Hash package receipt is
`.cache/windows-desktop-starter-20261113/hash-package.json`.

Only private example CLI commands and repository helper/template files changed.
No language/compiler/SDK API, DB, normative language contract, dependencies,
CI workflow or runner changed. Guides and this development record document the
behavior/limits. New generated projects begin at 0.1.0; the repository/example
version was not bumped. Local commits and unsigned packages only; no push,
release, signing or publication.
