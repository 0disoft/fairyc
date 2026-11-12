# Fairy Hash keyboard, accessibility and display verification — 2026-11-12

## Changes

The real keyboard probe exposed a multiline read-only digest Edit consuming Tab.
The native bridge now forwards unmodified Tab/Shift+Tab/Enter before Windows
dialog dispatch. Fairy chooses forward/reverse traversal and maps Enter to the
focused Open, Copy, Save or Cancel action. Enter on a read-only field has no
action. Native file-dialog input stays with Windows. The private bridge gains
focus-query/traversal functions; compiler/language/public SDK contracts do not
change.

Normal status changes use polite UI Automation live regions; errors use
assertive live regions. Fairy chooses urgency and silences frequent progress
updates. The native bridge sets IAccPropServices LiveSetting and raises
EVENT_OBJECT_LIVEREGIONCHANGED after updating text, clearing annotations at
teardown. This follows the [Microsoft Win32 live-region guidance](https://learn.microsoft.com/en-us/accessibility-tools-docs/items/win32/text_livesetting).
Native UIA roles, accessible names, read-only values and progress range remain.

The isolated-desktop verifier uses actual SendInput with foreground/PID checks.
It preserves the clipboard before switching, materializes/releases its previous
owner on the original desktop, and restores contents afterward. A Job Object
contains its processes; a separate watchdog and finally block restore the input
desktop. It does not terminate external input helpers. These Python probes and
their dependencies are not linked into the distributed EXE.

## Executed validations

Compiler: `.cache/windows-hash-save-20261112/fairyc-go.exe`, Windows x64 release.
Commands run from the repository root unless marked otherwise:

```powershell
# From examples/win32_hash
& '../../.cache/windows-hash-save-20261112/fairyc-go.exe' check --locked
& '../../.cache/windows-hash-save-20261112/fairyc-go.exe' build --locked --release
# From repository root
python -B examples/win32_hash/verify_cli.py
python -B examples/win32_hash/verify_gui.py --usability accessibility
python -B examples/win32_hash/verify_gui.py --usability display
python -B examples/win32_hash/verify_desktop.py
python -B examples/win32_hash/verify_desktop.py --mode narrator
python -B examples/win32_hash/verify_gui.py --job-only --job-mib 256
```

| Check | Result |
| --- | --- |
| Locked check/build | PASS; 17 Fairy source files, 228 MIR functions |
| CLI | PASS; 19 cases, including reports and protected source destinations |
| Real keys | PASS; 54 SendInput receipts, all with verified owned foreground |
| Tab/Shift+Tab | PASS; complete forward/reverse cycles in idle/completed/busy states |
| Enter | PASS; Open, Copy, owner-drawn Save and Cancel |
| Access keys | PASS; Alt+O/C/S/X; native dialog Alt+N/Enter and Save-dialog Escape |
| Action contents | PASS; clipboard digest, saved report and cancellation keeping window |
| UI Automation | PASS; roles, names, values, access keys, disabled actions, progress 0–100 |
| Live regions | PASS; polite completion, assertive busy-clipboard error, cancellation events |
| Progress silence | PASS; LiveSetting Off, no repeated live events during observed busy interval |
| Desktop cleanup | PASS; original desktop restored and owned process tree contained |
| Connected monitor | PASS; 1920×1080, 125%, DPI 120, PMv2; digest height 88px, font -18 |
| Cross-monitor DPI | SKIPPED; only one connected monitor, no different-DPI destination |
| Narrator speech | SKIPPED; launch attempted, CreateProcess rejected UIAccess Narrator with Win32 740 |
| 256 MiB GUI regression | PASS; correct digest, advancing progress, resize, cancel/reuse, three saved reports and source/hard-link protection |

The 256 MiB job took 5.787s (44.24 MiB/s). Observed UI ping p95/max was
1.71/12.70ms; resize 26.95ms; cancellation 21.74/22.55ms. These are one-run local
observations, not a performance comparison or guarantee.

Receipts are under `.cache/windows-hash-usability-final-20261112/`:
`keyboard-final-retry.json`, `accessibility-live.json`, `display-final.json`,
`narrator-final.json`, `cli-final.json`, and `jobs-256-final.json`.

Earlier failures are retained: initial real Tab stayed on digest control 203;
an isolated-copy attempt failed before releasing the prior clipboard owner;
one final keyboard attempt timed out accepting a native file dialog and its
retry passed. A build mistakenly run alongside that input test failed replacing
the running EXE; a sequential locked build passed and its image SHA stayed the
same. The default 16 MiB job did not provide enough advancing progress samples;
the documented 256 MiB run passed. These failed attempts are not passes. Native
file-dialog timing can still vary with shell/system load.

## Image and remaining limits

Release EXE: **188,416 bytes (184 KiB)**. Immediately before the live-region
addition with the same compiler it was 187,392 bytes: **+1,024 bytes (1 KiB)**.
SHA-256: `c9fa46080d54521a377b24f90892ab486c91b4cdacc8eb8320b9fc8fd34212da`.
No bundled UI framework, Python, screen reader, asset or extra runtime was added.

UIA inspection and WinEvent delivery do not certify actual screen-reader speech.
The Narrator probe uses the documented [Narrator+Ctrl+X last-phrase command](https://support.microsoft.com/en-us/accessibility/windows/narrator/appendix-b-narrator-keyboard-commands-and-touch-gestures)
when launch succeeds; here it never started, and no audio or speech history was
captured. No elevation or screen-reader install was attempted. A supported
interactive Narrator session or another installed reader remains necessary for
speech verification. A second monitor with different scaling remains necessary
for physical WM_DPICHANGED transition verification.

High contrast, IME, desktop-pixel/hover, full compiler/bootstrap and hosted CI
were not rerun for these key-routing/live-region changes. No DB, public API,
dependency policy, normative source-of-truth document, CI workflow or hosted
runner changed. Only the example and development documentation changed.
Unrelated untracked `docs-site/` was preserved. Local commits only; no push,
release or deployment.
