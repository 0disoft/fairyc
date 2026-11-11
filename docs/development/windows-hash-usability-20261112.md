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
