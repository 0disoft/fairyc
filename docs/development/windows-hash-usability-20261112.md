# Fairy Hash keyboard, accessibility and display checks, 2026-11-12

Scope: the Hash utility, separate from the minimal Win32 example. Production
application policy remains in Fairy; C performs native Windows operations.

## Environment and first actual input

One monitor is connected: 1920×1080 at 125%. A physical transition between
different monitor scales cannot be certified on this inventory. No virtual
monitor/driver or persistent display scaling was installed/changed.

The retained authored release from the progress work was tested through actual
SendInput: Tab focused native path control 201, Shift+Tab returned to Open 202.
The new `native_keyboard.py` validates process ownership, requires stable
foreground, refuses held user keys/buttons and releases only its own partially
injected keys after an incomplete send. It observes focus with GetGUIThreadInfo;
it does not inject WM_KEYDOWN or use a private application diagnostic protocol.
Receipts: `.cache/windows-hash-usability-20261112/baseline-keys.json`.

Earlier foreground wheel-capture activity was observed, but this actual-key
probe acquired foreground without terminating any external process.

## Native accessibility names

The native IUIAutomation baseline confirmed missing names on path Edit 201 and
progress bar 213. An early managed UIA client returned generic Pane roles and
could not register its client-side proxy assembly; those observations were not
used to judge application roles. `accessibility_probe.py` instead calls the
public Windows SDK native COM interfaces, with no accessibility actions or
private application protocol.

Fairy now supplies stable names for path (`파일 경로`), digest (`SHA-256 해시`)
and progress (`전체 진행률`). C applies standard IAccPropServices annotations,
clears them before destroying the HWNDs and balances COM initialization. Windows
still supplies native Edit/Button/ProgressBar providers, read-only Value patterns,
0–100 RangeValue and access keys. Only system OLE32 is added to the link inputs;
there is no shipped framework or custom accessibility server.

The authored release passed `check`, `build --locked --release` and
`verify_gui.py --usability accessibility`: initial/ready/busy names, roles,
read-only values matching the real Unicode path/digest, 100% completed range,
disabled partial-result actions and enabled Alt+X Cancel. Actual screen-reader
voice output was not tested. Receipts are under
`.cache/windows-hash-usability-20261112/`.

## Actual keyboard limitation

Full physical-key runs repeatedly lost foreground after early Tab steps to an
ego window or no foreground window. Subsequent input was aborted, not redirected.
The user authorized termination of `krasis-input.exe` PID 34880. Both process
termination and WM_CLOSE were denied by Windows (access denied); the user then
said manual termination was currently difficult. No other helper/process was
terminated. Whole Tab loops, Enter and Alt shortcuts therefore remain unverified
through SendInput in this environment. The CLI includes these cases for rerun
after the foreground interference is removed; they are not counted as passes.

## Reproducible usability CLI

`verify_gui.py --usability accessibility|display|keyboard|all` reports the distinct
kind `actual-win32-hash-usability`, not the full dialog/visual/job suite. Native
tab-order assertions now pass in both directions in three application states:

| Starting state/control | Forward order | Reverse order |
|---|---|---|
| Initial / Open 202 | 201, 203, 202 | 203, 201, 202 |
| Completed / Open 202 | 201, 203, 208, 204, 202 | 204, 208, 203, 201, 202 |
| Busy / Cancel 212 | 201, 203, 212 | 203, 201, 212 |

These are actual HWND `GetNextDlgTabItem` results and explicitly report
`nativeOrderIsPhysicalInput:false`; they do not substitute for the blocked
SendInput suite. The keyboard CLI contains actual Enter on Open/Copy/Save/Cancel,
Alt+O/C/S/X, complete Tab/Shift+Tab loops and Escape from a native Save dialog.
It inserts test paths through standard dialog controls; typed text/IME is outside
this slice. Ctrl-based shortcuts were not verified here.

The Go integration selects comma-separated modes with
`FAIRYC_HASH_USABILITY=accessibility,display`; it rejects simultaneous job-only.
High contrast requires the separate `FAIRYC_HASH_HIGH_CONTRAST=1` opt-in and a
display/all mode. Ordinary CI/default checks do not acquire a new physical-input
or global-settings requirement. Invalid standalone CLI option combinations
(contrast without display, usability plus job-only) fail with JSON/exit 1: 2/2.

## Actual display and high contrast

The authored release passed both `--usability display` and
`--usability display --high-contrast`. A physical SetWindowPos move on the one
connected monitor observed OS DPI **120** at **125%**, PMv2 awareness, digest
height **88 physical pixels**, and LOGFONTW height **-18** for the 14-logical-pixel
control font. Layout checks follow the current window DPI rather than retaining
the startup digest height. No fabricated WM_DPICHANGED was used.

There are no connected monitors with different DPI: the report explicitly sets
`physicalDifferentDpiTransition:false` and gives the skip reason. Moving between
different physical monitor scales remains unverified, including future monitor
combinations and rapid transitions.

The OS high-contrast flag was actually enabled. Live system COLOR_WINDOW and the
PrintWindow-rendered application canvas both measured **2105376 / 0x202020**;
COLOR_WINDOWTEXT was **16777215 / 0xFFFFFF**, with **65 matching text pixels** in
the SHA-256 label. These are actual native-window rendering observations, not an
unobstructed desktop screenshot or a general contrast-ratio certification.
Flags, system colors and the application's original warm palette were verified
after restoration. A separate child watchdog holds a copy of the restoration
snapshot and restores it on controller exit or after 45 seconds; normal success
restores settings and terminates/waits for the watchdog immediately. The final
run also retained the same default scheme string. Earlier Windows theme changes
canonicalized an empty default scheme name; flags/colors, rather than exact
scheme-name spelling, define restoration success. No persistent scale setting,
display driver, virtual monitor or new user preference was installed.

## Final executed checks and size

The existing authored CLI was reused from
`.cache/windows-gui-fairy-cli-20261109/low/fairyc.exe`. Its `check` and
`build --locked --release` passed for the example. Final authored-release checks:

```powershell
python -B examples/win32_hash/verify_cli.py
python -B examples/win32_hash/verify_gui.py --usability accessibility
python -B examples/win32_hash/verify_gui.py --usability display
python -B examples/win32_hash/verify_gui.py --usability display --high-contrast
python -B examples/win32_minimal/size_report.py examples/win32_hash/build/win32-hash.exe
```

CLI passes all **19 cases**. Accessibility passes initial/completed/busy native
properties and the six tab-order observations. Display/high-contrast restoration
passes; screen-reader voice and the physical different-DPI transition remain
untested. Receipts are `authored-cli.json`, `accessibility.json`, `dpi.json`,
`display.json` and `release-size.json` under the ignored
`.cache/windows-hash-usability-20261112/` directory.

Go frontend development/release integration also passed:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_USABILITY='accessibility,display'
Remove-Item Env:FAIRYC_HASH_JOB_ONLY -ErrorAction SilentlyContinue
Remove-Item Env:FAIRYC_HASH_HIGH_CONTRAST -ErrorAction SilentlyContinue
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

PASS **32.072s**, with development **276,480 bytes** and release **172,032 bytes**.
Each profile passes the 19 CLI cases plus native accessibility and current-DPI
checks, and logs the physical transition omission. Its separate build profiles
are not the authored release below. High contrast was tested separately on the
authored release; this Go run did not toggle OS settings or send keyboard input.
Receipt: `go-integration.log` in the same cache directory.

Final authored release: **187,392 bytes (183 KiB)**, **+1,024 bytes (1 KiB)** versus
the preceding 186,368-byte progress build. SHA-256:
`f4c94b3303ea54914c341c5e163a5f847d54a2d7aa736302340fd8d1d13bb8af`.
Imports: USER32, GDI32, COMDLG32, COMCTL32, **OLE32**, KERNEL32. No framework,
font, bitmap or verification runtime is bundled.

No public compiler/language/SDK API, DB, dependency policy, CI workflow or runner
was changed. Changes are confined to the example's private native bridge,
Fairy-supplied accessibility text, focused Go/Python verification and execution
documentation. Cache/build output stays ignored; no normative source-of-truth
document was changed. Unchanged full visual/mouse, minimal GUI, general compiler,
fresh bootstrap and hosted CI suites were not rerun. No other process was killed
after the denied PID 34880 attempt. Local commits only; no push or deployment.
