# Fairy Hash warm compact UI, 2026-11-11

Owner: [example README](../../examples/win32_hash/README.md),
`examples/win32_hash/src/{ui,application}.fy`, `native/window.c`, `native/app.rc`
and `fairy.windows.json`. This implements the selected second warm compact
mockup direction; generated mockup images are references, not shipped assets.

## Result and ownership

The default client area is 600 × 280 logical pixels, with a 520 × 276 minimum.
The warm ivory canvas has a filename/path/byte summary, two-line monospaced
SHA-256 result and bottom Copy/Save actions. The native Open/Copy buttons use
Common Controls v6, activated by an inline XML manifest in the declared RC
resource. Save is a small owner-drawn native BUTTON with enabled/pressed/focus
states. Its name and keyboard/dialog behavior remain native. Document decoration
uses GDI lines, and fonts are installed Segoe UI/Consolas with Windows font
fallback. The high-contrast palette uses system colors.

Fairy owns filename extraction, byte labels, digest line breaks (32 + 32),
status/tone selection, Copy availability/action and DPI/client-size layout.
The C bridge owns HWNDs, drawing/fonts, events, dialogs and CF_UNICODETEXT
transfer. It receives the original 64 hex digits from Fairy for Copy; C does
not reconstruct or hash the displayed text. CLI JSON and report bytes retain
the original uninterrupted digest. A busy clipboard produces a retryable error.
Copy/Save disable during hashing and after failure. Existing cancellation,
source/hard-link protection and cooperative close processing remain in place.

The full path remains in a read-only horizontally scrolling edit; only the
filename static ellipsizes visually. Minimize skips zero-size layout instead
of treating it as a fatal move failure. Window minimum/initial dimensions
account for the native nonclient area at the window DPI.

## Executed validation

Go integration:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

PASS, **44.310s**, one top-level test, two profile subtests, zero skips.
The existing authored compiler/companion/runtime pack in
`.cache/windows-gui-fairy-cli-20261109/low/` also passed `check`,
`build --locked` and `build --locked --release` on this example. The example
lock was refreshed for the changed Fairy sources; compiler/SDK pins did not change.

Each of the four compiler/profile images passed:

- `verify_cli.py`: 19 reference SHA-256, save/error and source-protection cases.
- `verify_gui.py`: real Windows Open/Save dialogs, three reference files,
  Unicode paths longer than 127 characters, cancellation preservation,
  overwrite confirmation, protected input and closing a 64 MiB job.
- Actual side-by-side Common Controls loading observed in the app's modules.
- Exact two-line digest and filename/byte summary; Copy's actual Unicode
  clipboard data matches the independent 64-digit reference. The verifier
  retains/restores the OLE clipboard data object. A deliberately locked
  clipboard fails cleanly, then the same action succeeds after releasing it.
- Nonoverlapping, in-bounds native controls at minimum and expanded sizes;
  exactly two native digest lines after resize; minimize/restore preserves
  the process and usable layout.

`verify_gui.py --screenshot <path>` optionally captures the real `sample.txt`
result via Windows/GDI and the standard-library-only `capture_window.py`.
Both authored profiles were captured and visually inspected; final output is
`.cache/windows-hash-theme-20261111/release.png`. All verifier processes exited.
`git diff --check` passed.

## Measured size

Comparison with the same compiler/profile before this UI work:

| Compiler | Before development | After development | Before release | After release |
|---|---:|---:|---:|---:|
| Go frontend | 241,152 | 254,464 | 152,064 | 160,256 |
| Reused authored CLI | 272,896 | 290,304 | 164,864 | **175,104** |

The authored release changes from **161 KiB to 171 KiB**, an increase of
**10,240 bytes (10 KiB, 6.21%)** for theme, layout, localization and Copy together.
PE section growth is `.text` +7,680, `.rdata` +1,536 and `.rsrc` +1,024 bytes.
The image remains GUI-subsystem, static CRT, with no overlay or COFF symbols.
Imports are Windows USER32, GDI32, COMDLG32, COMCTL32 and KERNEL32 only.
No bundled UI framework, font, bitmap, browser runtime or Python dependency
is introduced in the distributable. Python is only for optional verification.

Authored release SHA-256:
`9814534d4e81bc9325d0d9accad11be30710209977e958ad2f6e0d1154ab5af7`.
Receipts, source/image/compiler hashes, retained images and screenshots are in
`.cache/windows-hash-theme-20261111/summary.json` and its sibling files.
The final authored release is also `examples/win32_hash/build/win32-hash.exe`.

## Limits and changes

Early verification encountered clipboard contention; OLE snapshot/restore now
uses bounded retry and correct void return metadata for uninitialization.
A screenshot-only same-digest selection race was fixed by waiting for the
actual selected path, not only identical digest text. An abandoned first
verification child held the original EXE open and prevented an authored build
from replacing it; its identity/path were checked and it was closed normally
before a successful rebuild. Failed attempts are not counted as passes.

GUI checks use native control messages, not physical keyboard/mouse injection.
Physical cross-monitor DPI transitions, a manual high-contrast session and
other dialog locales were not certified. System-font fallback can vary with
installed fonts. The unchanged SHA implementation's unit suite, unchanged
`win32_minimal` full suite, Linux, hosted CI and a fresh compiler bootstrap
were omitted for this UI-only change; both frontends' focused CLI/GUI execution
was used instead. Existing cooperative I/O/snapshot limits still apply.

Implementation commits: `13abc483` (native visual styles), `62997c68`
(compact layout/localization/Copy and verification). Only example behavior,
private bridge functions, focused regression and example documentation changed.
Public compiler/language/SDK APIs, DB, repository hygiene, dependency versions
and CI runners/workflows are unchanged. Local commits only; no push, release,
publication or deployment.

## Follow-up: stale filename ellipsis after maximizing

The first compact release could show the old narrow-window `...` followed by
the rest of the filename after maximizing. The stored filename was intact;
the expanded native STATIC did not clear all of its old pixels. The bridge
now invalidates the entire filename control, with background erasure, after
moving/resizing it. Native `SS_ENDELLIPSIS` and the full stored filename remain.
No Fairy source, hashing/report behavior or public API changes are needed.

The regression reads existing visible pixels using `GetWindowDC`/`BitBlt`.
`PrintWindow` requests a fresh draw and masked the issue during the first
investigation. The verifier checks that narrow-to-maximized text looks exactly
like a fresh selection at the same maximized size, over two maximize/restore
cycles. It waits for right/left layout margins to settle and selects a distinct
temporary file between equivalent renders to avoid stale readiness observations.
The old retained authored release fails this regression; the fixed Go release
passes it along with existing dialog/hash/copy/resize/close checks. Evidence is
in `.cache/windows-hash-ellipsis-20261111/` (`before-gui.json`, `fixed-gui.json`
and the maximized screenshots). No Windows style contract was redefined;
[native static styles](https://learn.microsoft.com/en-us/windows/win32/controls/static-control-styles)
still determine ellipsis and text behavior.

Final verification: the Go development/release integration passed in 48.865s
(two profile subtests, zero skips), including 19 CLI cases per profile and the
new visible-pixel regression. The reused authored CLI's `build --locked
--release` passed; that image passed the 19 CLI cases and the complete GUI
verifier, including two maximize/restore comparisons. The updated release is
still **175,104 bytes (171 KiB)**, SHA-256
`4a44311f167b2577e14f4c58a17103f094fb932d8ea46b327b479812191d3093`.
No test application processes remain. Authored development, a fresh compiler
bootstrap, physical input/cross-monitor DPI/high-contrast and hosted/Linux
checks were omitted for this small C paint correction. The earlier execution
record and its hashes describe the previous version, not this rebuilt EXE.
No API, DB, repository hygiene or CI runner/workflow changes; no push.

## Follow-up: bounded maximized layout, tooltips and late paint observations

The form is centered and capped at 720 × 280 logical pixels in large windows.
The digest stays 70 logical pixels high. Native wrapping tooltips provide the
complete filename/path only when clipped, including Unicode and literal `&`.
Callback storage outlives the notification and supports text beyond its inline
80-character buffer. Standard Common Controls tracking messages exercise the
existing registered tools; the probe restores the original TOOLINFO afterward.
Optional `--desktop-hover` separately requires actual pointer movement.
Implementation commits: `dd9a8f4a`, `665712c1`, `009aac7b`.

The user subsequently observed solid black document-icon and byte-count areas
late in verification, including after source-overwrite refusal. The earlier
explanation attributing everything to closing-window animation was insufficient.
The original image was not reproduced in the new baseline run, and the user
also reported that recent runs no longer showed it. This is **repaint hardening,
not a confirmed root-cause fix**; no GDI leak or compositor defect is claimed.

`63459740` replaces per-control immediate repaint with one full owner/child
erase-and-redraw after Fairy finishes submitting layout. It extends the earlier
filename-only invalidation to all controls and uncovered parent areas. Native
Open/Save return also redraws the owner on acceptance or cancellation, preserving
the dialog error before the redraw. Layout arithmetic remains in Fairy; C only
applies rectangles and performs Win32 painting.

The verifier observes background pixels on icon 211 and byte label 209 at seven
checkpoints: initial, two maximize/restore cycles, source protection, minimum and
expanded sizes, and minimize/restore. It exposes the owned window once before
the sequence rather than revealing it anew at each sample, which could repair
stale paint. Occlusion produces an explicit diagnostic. These are sampled
background checks, not certification of every pixel or every transient frame.
The visible capture path now reads the composed desktop with `GetDC(NULL)` and
`BitBlt`, without requesting application paint. A parent's `GetWindowDC` capture
could contain stale child/nonclient backing pixels. Filename comparisons wait
for consecutive matching frames after native-dialog composition, without
sending any repaint request. `PrintWindow` remains a separate diagnostic capture.
Restore sampling allows up to two seconds for the existing DWM animation to
settle against the initial canvas color, without requesting a new application
paint. An early strict sample saw desktop pixels in the moving canvas corner;
that animation observation was not treated as a reproduction of black controls.

Go development/release integration passed in **69.927s**, two profile subtests,
zero skips, including 19 CLI cases and the complete GUI verifier per profile.
The native/Fairy sources are unchanged since that run. The final desktop capture
method is additionally checked on the authored release below. The reused authored
CLI passed `check` and `build --locked --release`; its example source lock was
refreshed without changing compiler or SDK pins.

The rebuilt authored release is **178,688 bytes (174.5 KiB)**, unchanged from the
preceding layout/tooltip image and 3,584 bytes above the original compact UI.
SHA-256: `359ee46ab4749761b08a5ad2e97032d76730dab0c4558b462b0ce9ad4d881561`.
It passes all 19 CLI reference cases. Logs, retained image, size report and
optional diagnostic captures are under `.cache/windows-hash-paint-20261111/`.
The usable release is `examples/win32_hash/build/win32-hash.exe`.
The final authored-release GUI run passes all seven composed-background samples,
two settled filename comparisons, ten tooltip text comparisons, native dialogs,
source protection, Copy/Save, resize/restore and close-during-hash. Earlier
capture-settling failures are not counted as passes. The final expanded visible
capture was also inspected; no black document/byte regions were present.

The unchanged hash unit suite, minimal-GUI suite, fresh compiler bootstrap,
physical hover/cross-monitor DPI/high-contrast, Linux and hosted CI were omitted
for this repaint follow-up. No external helper was terminated. No public
compiler/language/SDK API, DB, dependency, repository-policy or CI runner changes;
local commits only, no push/publication/deployment.
