# Fairy Hash expected SHA-256 comparison — 2026-11-13

## Result

The utility now compares a calculated SHA-256 against an expected value. Fairy C
owns normalization, matching, comparison states and JSON formatting in
`examples/win32_hash/src/comparison.fy`. No C hash implementation, framework,
dependency or runtime was introduced. Version stays **0.1.0**.

Expected values must contain exactly 64 ASCII hexadecimal digits after trimming
surrounding ASCII whitespace (space, tab, LF, VT, FF, CR). Uppercase A–F is
normalized to lowercase. Interior spaces, wrong lengths and non-ASCII characters
are invalid. GUI states distinguish empty input, invalid input, no usable
calculated result, match and mismatch.

## CLI

```powershell
.\build\win32-hash.exe --compare C:\data\input.bin '<64-hex-expected-SHA256>' C:\data\compare.json | Out-String
```

The final report path is optional. Successful calculation emits the ordinary
hash JSON plus `expectedSha256` and `matches`. Exit codes are **0 match,
2 mismatch, 1 error**. Mismatch can save a report. Expected-input validation
precedes file I/O; malformed values emit `{"ok":false,"errorCode":12}` and
preserve any existing destination. Existing `--hash` output/exit codes remain
unchanged. Safe report publication and source/hard-link protection are retained.

## GUI wiring

The editable expected-hash control (214) follows the read-only digest in the tab
order; its mnemonic is Alt+E. A separate status (216) presents the comparison so
operational errors/progress in control 205 remain visible. Native code transports
UTF-8 input and EN_CHANGE notifications. A retained pending-input latch avoids
losing changes when a layout event is already pending. Fairy C maps Enter on
the input to refreshing comparison, without activating Open or a copy action.

Expected text persists between selections. Selecting another file invalidates
the previous match before computation; failed/cancelled jobs cannot display an
old match. Input is disabled during hashing/saving. Mismatch permits copying or
saving the actual calculated hash. Save rebuilds the report from the current
expectation: valid input adds comparison fields, empty/invalid input saves the
ordinary calculated-hash report. Copy remains the actual 64-digit hash.

Client size becomes 600 × 380 logical pixels, minimum 520 × 376; a maximized
window centers the form capped at 720 × 380. The digest stays 70 pixels high.
Native tab-order/accessibility probe expectations were updated for control 214
and busy-state disabling. The comparison status has a polite live-region
annotation; transitions to match/mismatch notify accessibility clients, while
invalid-input typing does not emit explicit live-region notifications.

These GUI paths are compiled and reviewed only. No visible or hidden GUI was
launched, and no desktop interaction was performed in this work. Actual input,
layout, Alt+E, focus, IME and screen-reader behavior remain unverified for the
new controls. Historical GUI receipts do not certify this revision.

## Executed checks

From `examples/win32_hash`, using
`../../.cache/windows-hash-save-20261112/fairyc-go.exe`:

```powershell
fairyc-go.exe check
fairyc-go.exe test --locked --file tests/comparison.fy
fairyc-go.exe build --locked --release
python -B verify_compare.py
python -B verify_cli.py --exe build/win32-hash.exe
python -B package_release.py --exe build/win32-hash.exe
```

Passed:

- Frontend check: 18 source files, 235 MIR functions; updated lock accepted by
  the locked release build and unit test.
- One pure Fairy C comparison test covering nine state/validation assertions.
- 16 actual CLI comparison cases: match, uppercase/boundary whitespace,
  mismatch, saved matching/mismatching reports, invalid lengths/characters/
  Unicode, validation before opening a missing file, preservation of an existing
  report, missing source and source overwrite protection. JSON and exit codes
  were compared with Python's SHA-256 reference; no temporary report leaked.
- Existing 19 hash/report/error CLI cases.
- The package verifier repeated those 19 + 16 cases against the EXE extracted
  under a temporary Korean/space-containing path. It checked four ZIP entries,
  CRC/checksums, guide/license bytes, PE resources, x64 GUI subsystem, manifest,
  configured title/version and nine embedded icon sizes. Temporary extraction
  was removed. `cliCases` stays 19; `comparisonCases` is reported separately.
- Python AST syntax checks for changed verifiers/probes; `git diff --check`.

Receipts: `.cache/windows-hash-comparison-20261113/compare-core.json`,
`hash-core.json` and `package.json`. The first two describe the initial CLI
slice; `package.json` describes the final packaged executable.

The custom-brand verifier now propagates the packaged comparison count; its
additional temporary branding build was not rerun. No broad compiler/bootstrap,
unchanged SHA differential, performance, GUI, keyboard, mouse, clipboard,
Narrator, high-contrast or monitor/DPI tests were run. No hosted CI was invoked.

## Artifacts and scope

| Artifact | Before comparison | After comparison |
| --- | ---: | ---: |
| Release EXE | 196,608 bytes (192 KiB) | 202,752 bytes (198 KiB) |
| Portable ZIP | 105,706 bytes | 109,014 bytes |

EXE growth: **6,144 bytes (6 KiB)**. ZIP growth: 3,308 bytes.

- EXE SHA-256: `3b01a85b6c2fc6b3f0e876b0353795d20b9c9c6eb73f8c779fcc042f5a18d060`.
- ZIP SHA-256: `12e4c74521091923b2aa7b0438525f193e8d0c9749ac05033b1431e175dfe30a`.
- Local ZIP: `examples/win32_hash/dist/fairy-hash-0.1.0-windows-x64.zip`
  and `.zip.sha256`.

Only the example's private CLI surface was extended. No compiler/language/SDK
API, DB, normative source-of-truth contract, CI workflow or runner changed.
Example documentation, distribution instructions and this execution record were
updated; generated build/package receipts remain ignored. The desktop integration
fixture now copies the new Fairy source; it was not executed. Concurrent
`docs-site/` work was untouched. Small local commits and an unsigned local ZIP
were produced; no push, release or publication.
