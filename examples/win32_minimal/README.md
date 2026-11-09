# Minimal Win32 GUI

From this folder on Windows x64 with MSVC and a Windows SDK installed:

```powershell
fairyc build --release
.\build\win32-minimal.exe
```

`fairyc run --release` also opens the window. Close it normally, or use
`fairyc run --release -- --smoke` to exercise creation, painting, the message
loop and a timer-driven close request automatically. Smoke mode also verifies real
edit notifications and button clicks, empty input, Unicode results, repeated
edits, the input length limit and rejection of a command without its control
handle. Closing passes through Fairy's `on_close`; smoke mode exits with code 5
if an event check fails or Fairy declines its final close request.

Type into **Message** and click **Apply**. The result displays the entered
Unicode text; an empty message shows a prompt. The status line identifies
`WM_COMMAND / EN_CHANGE` from editing and `WM_COMMAND / BN_CLICKED` from the
button. Enter runs Apply through Fairy and reports `WM_KEYDOWN: Enter`;
Esc asks Fairy to close. Input is bounded to 127 UTF-16 code units. Tab moves between the edit
and button controls, and the controls follow the window width when resized.

## CLI control for agents

Python 3 is needed only for this optional controller, not for the GUI executable.
From this folder:

```powershell
python -B control.py --verify
python -B control.py
```

`--verify` opens a real window, verifies the title and visibility, empty input,
Unicode including JSON quotes/backslashes, repeated clicks, length bounds and
invalid native requests, then closes the window. It prints one JSON result and
returns nonzero on failure. Every request has a five-second reply deadline.

Without `--verify`, the controller keeps its window open and accepts one JSON
object per stdin line. An LLM's existing command tool can keep this CLI session
open, write requests and read replies; no model/API key is built into the app.
The first reply is the initial window state. Example requests:

```json
{"command":"state"}
{"command":"input","text":"제로디 Win32 이벤트"}
{"command":"click"}
{"command":"close"}
```

State replies use `fairyc.win32-example-control/v1` and include `ok`, `pid`,
`visible`, `title`, `input`, `result`, `status`, `edits` and `clicks`. The input
command updates the native EDIT via `SetWindowTextW`, which generates
`EN_CHANGE`. Click sends the native BUTTON `BM_CLICK`, which generates
`BN_CLICKED`. Close sends `WM_CLOSE`, the same message used by the title-bar
close button. The ordinary WndProc forwards these events to Fairy and replies read actual
control text. This verifies programmatic control events; it does not certify
physical keyboard/mouse delivery, IME, screen rendering or accessibility.

### Enter and Esc through Fairy

On an idle, unlocked interactive desktop:

```powershell
python -B control.py --verify-keyboard
```

This sends real `SendInput` key-down/key-up pairs to the owned window. It checks
empty Enter, Unicode Enter, ignored Ctrl+Enter, Enter with Apply focused and
Esc closing with exit 0. Every Enter must apply exactly once; a subsequent state
read catches delayed/default-button duplication. No `BM_CLICK`, `WM_COMMAND`,
`WM_KEYDOWN` or `WM_CLOSE` message is fabricated to perform these key actions.
The existing desktop focus/held-input guards apply, including Return and Esc.

Interactive requests accept `{"command":"key","key":"enter"}` and
`{"command":"key","key":"escape"}`; optional `"focus":"input"` (default) or
`"focus":"apply"` selects the owned child control. `"key":"ctrl-enter"` tests
the ignored modified shortcut. The helper verifies actual native focus before
injecting. `arm-key<TAB>13|27<TAB>201|202` only arms observation and sets focus;
it cannot generate a key event or run an application action.

The pump intercepts Return/Esc `WM_KEYDOWN` before `IsDialogMessage` can run a
default button. Event 6 carries the native virtual key, modifiers and previous
key-state bit through scalar FFI. Fairy's `on_key` maps plain Enter to Apply and
plain Esc to `on_close`; modified/repeated key-downs choose no action. C has no
Enter-to-click or Esc-to-destroy fallback. Other keys retain normal Win32 dialog
handling. These tests do not certify IME composition or hardware auto-repeat.

State adds `keyCode`, `keyFlags`, `keyFocus`, `keyAction`, `keys`, `awaitingKey`
and `closed:false`. `keys` counts Fairy-processed key notifications, including
ignored ones; `clicks` counts Apply actions from both clicks and Enter. Flag bits
are Shift=1, Ctrl=2, Alt=4, Windows=8 and repeated key-down=16. `keyFocus` records
the native destination control ID (input=201, Apply=202); `keyAction` is Fairy's
decision (0=ignore, 1=Apply, 3=close request). The final key reply follows the
Fairy action and confirms the key/count/focus. An accepted Esc returns
`closed:true` after destruction; a declined Esc returns live state with
`closed:false`, and input/Enter remain usable in that session.

### DPI changes and scaled layout

The example sets Per-Monitor V2 awareness before creating any UI (Windows 10
1703 or later). Fairy uses 96-DPI base units and rounds `(value * dpi + 48) / 96`.
`on_size` scales margins, button size, vertical positions/heights, minimum size
and font height. The native bridge creates/caches the requested Segoe UI font,
selects it into all controls and releases replaced fonts.

`WM_DPICHANGED` copies the new DPI and suggested rectangle into native storage
and yields event 5 to Fairy. `on_dpi` supplies the new scaled minimum **before**
accepting the suggested window rectangle, then runs layout at the new DPI.
This ordering prevents an old high-DPI minimum from blocking a later shrink.
Pending DPI work precedes resize work, including during the modal drag loop.

```powershell
python -B control.py --verify-dpi
```

This is explicitly an **injected `WM_DPICHANGED` test on a real GUI**, not a
physical monitor transition or a display-setting change. It round-trips
96 / 144 / 192 / 96 / the initial OS DPI and checks actual window/control
rectangles, selected font heights, Fairy's event count, preserved text/click
state and rejected malformed requests. Result JSON uses
`verification:"injected-win32-dpi"`, includes the read-only monitor inventory
from `dpi_inventory.py`, and reports `physicalMonitorTransitionTested:false`.
It does not modify system display scaling.

Interactive JSON supports
`{"command":"dpi-test","dpi":144,"width":960,"height":540}`. DPI must be an
integer in 96..384, and suggested outer dimensions in 1..4096. This private
test command only exists in the opt-in automation session. State reports `dpi`
(last message DPI), `windowDpi` (actual `GetDpiForWindow`), `perMonitorV2`,
`dpiChanges` (Fairy-applied events), `dpiInjections`, `dpiSuggested`, `dpiApplied`
and actual selected `fonts` heights. `dpiApplied` records the accepted rectangle;
`window` independently measures the actual result. Injected message DPI can
differ from OS DPI by design; a successful injected check does not certify
physical cross-monitor or OS non-client scaling behavior.

### Resize and Fairy layout

```powershell
python -B control.py --verify-resize
```

This changes the real window through Windows
[SetWindowPos](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowpos),
then verifies Fairy's layout using actual control rectangles. It widens the
window, narrows to the minimum, changes height only, repeats the same size and
resizes again. Unicode input/result and edit/click counts must survive; invalid
sizes must leave all state unchanged, and Apply must still work afterward.

Interactive JSON accepts `{"command":"resize","width":800,"height":480}`.
Dimensions are the **outer window** size in the application's Win32 coordinate
space (physical pixels under PMv2), including its frame/title bar. The Fairy
minimum is 400 by 240 at 96 DPI and scales with DPI (500 by 300 at 120 DPI);
the example protocol caps each dimension at 4096. Booleans, non-integers,
unknown fields, malformed native dimensions and below-minimum sizes reject.

State replies additionally contain measured `window` and `client` rectangles,
the five actual child rectangles under `layout`, Fairy's mirrored `resizes`
counter and `minimum` size. Child positions are relative to the parent client
area. `resizes` includes the initial layout and counts processed size events;
coalesced notifications and unchanged-size requests need not each increment it.
Initial readiness and resize replies wait until Fairy has applied all controls.

Fairy owns initial window dimensions; `on_size` owns minimum dimensions, the
16-base-unit inset, input/result widths and the right-aligned fixed-base-width
Apply button. Vertical positions/heights stay fixed when only height changes,
and all base units scale when DPI changes.
C only reads native client dimensions, moves the requested control and applies
the supplied minimum tracking size. Size notifications are coalesced separately
from input/click/close events. The native message pump runs on a separate Win32
fiber on the same UI thread. `WM_SIZE` yields to Fairy even inside the standard
Windows modal border-drag loop, and resumes the native frame after Fairy has
applied its layout. There is no extra OS thread or retained Fairy callback.

### Live layout during real border dragging

On an idle, unlocked interactive desktop:

```powershell
python -B control.py --verify-drag
```

The controller hit-tests the owned window's bottom-right border as
`HTBOTTOMRIGHT`, presses the mouse button through `SendInput`, then moves
outward three times and inward once while keeping the button down. It reads
actual window/control rectangles at four intermediate sizes **before release**.
Every checkpoint must show `sizing:true`, `mouseDown:true`, increasing Fairy
`resizes` and `liveLayouts`, correct layout and unchanged text/edit/click state.
After releasing, it verifies `WM_EXITSIZEMOVE`, further input/click and clean
close. It never uses `SetWindowPos`, `SC_SIZE` or fabricated `WM_SIZE` messages
to perform this drag.

Interactive JSON `{"command":"drag-resize"}` performs the same bounded
four-step drag on the existing owned session and returns its checkpoints; it
leaves the session open. The dedicated `--verify-drag` mode also verifies setup,
post-drag controls and close. Native state adds `sizing`, `mouseDown`,
`sizeEntries`, `sizeExits` and `liveLayouts`. A live-layout completion is counted
only inside the native move/size loop while the left button is down.

The helper releases the injected button on success or failure and waits for the
native loop to exit before restoring the cursor. PID/foreground checks and
desktop limitations from the close-input section below also apply. Concurrent
user activity, unavailable foreground, outline-only window dragging or an
off-screen path fail rather than being reported as a successful live drag.
Desktop Go tests require `FAIRYC_RUN_DESKTOP_INTEGRATION=1`; default checks do
not inject desktop input.

### Title-bar close button and Alt+F4

On an idle, unlocked interactive Windows desktop, run:

```powershell
python -B control.py --verify-desktop-close
```

This opens two owned GUI processes sequentially, applies input, then closes one
by clicking its native title-bar close button and the other by Alt+F4. The
controller uses Windows [SendInput](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput)
for mouse/keyboard input. It does not send `WM_CLOSE` or `SC_CLOSE` to simulate
these two actions. A successful JSON receipt includes `systemClose:true`,
`closed:true`, `closeHandoff:true`, PID/HWND, input method, inserted event count
and process exit 0. `closeHandoff` confirms that Fairy resolved the close while
the native `WM_CLOSE` callback was still active, including caption tracking.
The mouse receipt also records the close-button center and `HTCLOSE` hit test.

Interactive JSON requests support `{"command":"close-button"}` and
`{"command":"alt-f4"}`. Either normally closes its session; start a new session
for the other action. If Fairy declines, the reply instead has `closed:false`
and the same session remains usable.

Desktop actions take foreground focus and briefly move the mouse. The optional
`desktop_input.py` helper locates only the spawned process's visible Fairy
window and retries activation for up to two seconds, temporarily attaching
input queues when needed. Access denied on an unrelated foreground queue is
recorded without preventing activation through the owned GUI thread; every
input still requires verified foreground ownership. Receipts include
`focusAttempts` and `attachErrors`. Activation must remain stable for 100 ms
before input. It queries `WM_GETTITLEBARINFOEX`
under per-monitor DPI awareness, checks the mouse target is unobscured and
hit-tests as `HTCLOSE`, then restores the cursor after the close reply if the
user has not moved it. Before activation and input, the helper waits up to two
seconds for held modifiers/buttons/F4 to be released; it does not release
user-held input. Persistently held input, unavailable foreground, blocked
input or missing native close observation fail the request. Keep the desktop
idle during these checks; concurrent user input can still race global input.
Locked/headless desktops and elevated targets are not supported. These are OS
injected input tests, not physical keyboard/mouse hardware or IME certification.

For mouse close and dragging, a read-only `WM_APP+1` probe waits for fresh
native hover and press delivery before proceeding. A hover counter distinguishes
the new movement from a prior notification. It observes notifications;
it does not fabricate input or close/resize messages. Receipts record
`hoverDelivered`, `pressDelivered`, `pressMessage` and `pressHit`; a stale image
without the probe fails with a rebuild instruction. The press hit must match
the target, and dragging requires `WM_NCLBUTTONDOWN` rather than a double-click.
Waits are bounded and injected mouse presses are released on failure.

The native `arm-close` request only arms observation and activates the owned
window. The subsequent `SC_CLOSE` is recorded by WndProc before Windows maps it
to `WM_CLOSE`, and Fairy decides whether to close. `WM_CLOSE` yields from the
message-pump fiber to Fairy immediately, before returning to Windows. Only an
armed desktop reply adds `systemClose` and `closeHandoff`; ordinary `close`
replies retain their existing shape.
The GUI needs no Python or extra runtime dependency for normal use.

The GUI enables this private example protocol only with `--automation-stdio`
and inherited pipe stdin/stdout. The Python controller owns that process and
closes it on stdin EOF or error. It opens no socket, contacts no service and
targets only its owned child process. Native requests use LF
lines `state`, `input<TAB>UTF-8`, `resize<TAB>width<TAB>height`, `click`, `close`,
`arm-close`, `arm-key<TAB>code<TAB>focus`, `dpi-test<TAB>dpi<TAB>width<TAB>height`;
replies are JSON lines.
The native buffer is bounded to 1023 bytes and validates UTF-8, NUL and input
length before mutation. The Python JSON request limit is 4096 bytes; multiline
input and unknown fields reject. There is one outstanding request at a time.
An alternate built example image can be selected with `--exe <path>`.

Fairy owns `main()`, the application event loop, edit/click/key/resize/DPI counts and the
`on_change` / `on_click` / `on_key` / `on_close` / `on_size` / `on_dpi` handlers in `src/main.fy`.
`on_change` chooses the edit
status; `on_click` decides empty-input behavior and builds the UTF-8 result
text. Editing leaves the last applied result intact until Apply is clicked.
`on_close` allows closing in this example; changing its decision to 0 declines
the request and leaves the window interactive.
C owns the Win32 window procedure, message pump,
control access, UTF-8/UTF-16 conversion and automation transport. The `.rc`
string table is compiled by SDK `rc.exe`. No UI
framework or browser runtime is bundled; the executable uses Windows system
DLLs and the static CRT. This prototype does not expose arbitrary Fairy
callbacks to Windows or generate Win32 SDK bindings.

The private bridge functions are `fairy_win32_open`, `fairy_win32_next`,
`fairy_win32_read_input`, `fairy_win32_present`, `fairy_win32_changed`,
`fairy_win32_resolve_close`, `fairy_win32_client_extent`, `fairy_win32_move`,
`fairy_win32_layout_done`, `fairy_win32_initialize_dpi`, `fairy_win32_dpi`,
`fairy_win32_font`, `fairy_win32_accept_dpi`, `fairy_win32_key`,
`fairy_win32_key_done` and `fairy_win32_close`.
`initialize_dpi` enables PMv2 and returns the system DPI. `open` takes Fairy's initial
window dimensions. `next` pumps Win32 messages and returns 1 for a click,
2 for `EN_CHANGE`, 3 for `WM_CLOSE`, 4 for `WM_SIZE`, 5 for `WM_DPICHANGED`,
6 for a Return/Esc key-down, or 0 on exit. `key` reads code/flags at axis 0/1;
`key_done` acknowledges only after Fairy has applied its policy.
Fairy handles the event and calls `present`
with the click result/count/status, or `changed` with the edit count/status.
C mirrors these counts for state replies and updates the controls. A notified
CLI input or click is acknowledged after the corresponding Fairy handler.
`WM_CLOSE` is intercepted before Windows' default destruction. Fairy calls
`resolve_close` with 1 to destroy the window or 0 to keep it open. A pending
CLI close receives `{"ok":true,"closed":true}` only after successful destruction,
or `{"ok":true,"closed":false}` after denial. The controller remains usable on
denial. Native `WM_DESTROY`/`WM_QUIT` still provide the exit notification;
`fairy_win32_close` releases native resources when Fairy's loop ends. Native
startup, transport failures/EOF and failing smoke checks retain forced cleanup.
The edit count measures native change notifications, not characters or CLI
commands; a replacement, clear or keyboard edit can produce a notification.
`next` returns before Fairy processes the event; C retains no Fairy
callback, context or input/output buffer. This single-window example uses the
existing pointer/scalar FFI and introduces no public SDK or export contract.
Application C bridge files keep `/W4 /WX`; generated C partitions keep their
existing consumer warning policy in both native builders.

`fairy.windows.json` selects the GUI subsystem and explicit native inputs.
See [ADR 0552](../../docs/adr/0552-minimal-windows-gui-build.md) for bounds and
the native-code trust boundary. Headers and resource assets are build-time
dependencies; the resulting example executable can run on its own.
Local C includes and resource asset paths should be written relative to the
project root; native source snapshots are compiled in temporary storage.
