# Fairy-owned Win32 events and layout, 2026-11-09

Owner: [example README](../../examples/win32_minimal/README.md) and the private
FFI in `src/main.fy` / `native/window.c`. [ADR 0552](../adr/0552-minimal-windows-gui-build.md)
continues to own native application build inputs.

Latest status: [distribution size and combined Win32 verification](windows-gui-slim-20261111.md)
records all sixteen Win32 tests passing in one run, including the slim image
and retained diagnostic CLI. The [Korean IME checkpoint](windows-gui-ime-20261110.md),
[Tab focus checkpoint](windows-gui-focus-20261110.md),
[mouse-target checkpoint](windows-gui-input-stability-20261109.md) and earlier
failed/scoped observations below remain as history.

The independent [Fairy file-hash utility](windows-hash-utility-20261111.md)
adds actual native file selection, Fairy SHA-256 and report saving with focused
Go/authored development/release verification. It does not change this minimal
example; the slim-build record above remains its latest complete regression run.

## Behavior and lifetime

Fairy now owns the application loop, edit/click/key/resize/DPI counts, edit status, empty-input
decision, close policy, DPI scaling, layout and result string construction. Its `on_change` selects status 105;
`on_click` reads bounded UTF-8 and produces the text. Editing preserves the last
applied result until the next click. `on_close` currently allows closing.
`on_size` computes all five control rectangles, including horizontal anchoring,
vertical positions/heights and minimum window dimensions. These use 96-DPI base
units scaled by Fairy with rounding; font height is supplied by Fairy too.
Fairy enables PMv2 before creating UI and supplies scaled initial dimensions to
`open`. `on_dpi` supplies the new minimum before accepting the copied suggested
rectangle, then runs `on_size`; this ordering allows high-to-low DPI shrink.
C owns Win32 window/control operations, message pumping, character conversion,
mirrored counts for observation and the existing JSONL transport.

The private functions are `fairy_win32_open`, `fairy_win32_next`,
`fairy_win32_read_input`, `fairy_win32_present`, `fairy_win32_changed`,
`fairy_win32_resolve_close`, `fairy_win32_client_extent`, `fairy_win32_move`,
`fairy_win32_layout_done`, `fairy_win32_initialize_dpi`, `fairy_win32_dpi`,
`fairy_win32_font`, `fairy_win32_accept_dpi`, `fairy_win32_key`,
`fairy_win32_key_done` and `fairy_win32_close`.
`next` returns 1 for a click, 2 for `EN_CHANGE`, 3 for `WM_CLOSE`, 4 for `WM_SIZE`,
5 for `WM_DPICHANGED`, 6 for Return/Esc key-down, and 0 on exit.
Fairy increments the corresponding edit/click count, then calls `present` or
`changed` to render and mirror its state. Only then does C acknowledge a pending
CLI request. A CLI reply therefore observes the handled event. The edit count
measures native change notifications, not characters or CLI commands. C retains no
Fairy function or input/output buffer. The opaque window lives until close.
The example remains single-window and single-threaded.

Size notifications have a separate coalesced pending flag so they cannot
overwrite input/click/close events. The pump delivers pending DPI work first,
then size work before other pending events. Fairy reads current client extents through scalar FFI,
computes positions, calls native control moves, and completes with
`layout_done`. C then mirrors the Fairy count/minimum and acknowledges initial
readiness or a pending resize request. Other event acknowledgements are kept
separate. Geometry replies use `GetWindowRect`, `GetClientRect` and
`MapWindowPoints` to observe actual rectangles rather than echo supplied boxes.

The native pump now runs on a Win32 fiber on the same OS thread as Fairy. A
`WM_SIZE` or `WM_DPICHANGED` handler on the pump fiber yields to the application fiber, including
when Windows' default modal sizing loop is active. Fairy's existing `next` /
`on_size` / `layout_done` path applies the geometry and resumes the suspended
native frame. No Fairy callback/export, extra OS thread or C layout policy is
introduced. Both fibers preserve floating-point state; the pump reserves a
1 MiB stack with the executable's default initial commit. Shutdown destroys
the window, resumes suspended native frames through `WM_QUIT`, deletes the
pump fiber and converts the application fiber back to a thread. Initialization
failure returns failure instead of silently using delayed layout.

`WM_CLOSE` returns from WndProc without default destruction. The CLI `close`
command sends that same message. Fairy resolves the event with
`fairy_win32_resolve_close(window, 1)` to destroy or 0 to decline; a pending CLI
request is acknowledged only after that decision, with `closed:true` after
successful destruction or `closed:false` on denial. `WM_DESTROY` posts `WM_QUIT`
and Fairy's loop then releases the native resources. Startup/transport errors,
pipe EOF and smoke assertion failures retain native forced cleanup; these are
separate from the normal user close policy. The successful smoke path sends
`WM_CLOSE` and fails on its next timer tick if Fairy declines closing.

Optional desktop input is owned by the example's `desktop_input.py` and
`control.py`. `arm-close` only prepares observation/foreground activation.
`SendInput` performs the actual title-bar click or Alt+F4; WndProc records
`SC_CLOSE` before default processing generates `WM_CLOSE`. Armed replies include
`systemClose` and are still emitted only after Fairy's close decision.

An initial synchronous callback implementation passed the Go frontend but was
rejected by the authored Fairy CLI's function-parameter subset. It was replaced
by the event pump before committing. No callback grammar or export support was
added. The first authored pump build also exposed generated C warnings treated
as errors. Both native builders now keep `/W4 /WX` for project C bridges and
retain their existing partition-consumer flags for generated C. No emitter
diagnostic, semantic check or compiler work budget was weakened or raised.

## Earlier click-handling verification

- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|Resources|WorkspaceBuild)$' -count=1 -v`:
  PASS, 29.895s. Both target profiles pass GUI smoke and JSONL verification;
  resource loading, subsystem inspection and failed-build preservation pass.
- The negative test now changes **only Fairy source**, replacing `b"Result: "`
  with `b"Broken: "`. The resulting native application's event smoke exits 5.
  This proves the result depends on Fairy policy, not a C fallback.
- Targeted `TestWindowsGUIWorkspaceBuild` rerun with an unused-variable warning
  in an otherwise valid project C bridge: PASS, 18.544s. The warning still
  rejects the build and preserves the prior executable under `/W4 /WX`.
- With `FAIRYC_RUN_PARITY_INTEGRATION=1`,
  `go test -mod=readonly ./internal/cli -run '^TestWindowsGUIAuthoredDriver$'
  -count=1 -v`: PASS, 255.273s. Fresh authored companion and full low CLI host
  (210 C parts) pass development/release `build` and `run -- --smoke`.
  Raw packet/subsystem/resource-preservation fixtures are independent of the
  example's changing application ABI; public CLI checks use the real Fairy app.
- `go run -mod=readonly ../../cmd/fairyc check` from `examples/lexer_candidate`:
  PASS, 777 source files / 9,482 MIR functions; lock refreshed.
- `python -B control.py --verify` passes the Go-built application. The freshly
  authored CLI also builds the example with `build --locked` and
  `build --locked --release`; each resulting image passes the same JSONL
  verifier (ten observations, five invalid native commands rejected, clean exit).
  Korean/Greek text, quotes, backslash, empty input, repeated clicks, maximum
  length, clear/apply and unchanged state after rejection are included.
- A persistent JSONL session then sent `input` with `Fairy C에서 처리한 클릭`
  and `click` to the authored release app. The visible window reported
  `Result: Fairy C에서 처리한 클릭` and `WM_COMMAND: BN_CLICKED`.
- Formatting and scoped `git diff --check`: PASS.

| Builder | Development app | Release app |
|---|---:|---:|
| Go CLI | 122,880 bytes | 119,808 bytes |
| Authored Fairy CLI/companion, low host | 138,240 bytes | 121,856 bytes |

Local logs: `.cache/windows-gui-fairy-events-go-20261109.log`,
`windows-gui-fairy-events-authored-20261109.log`,
`windows-gui-fairy-driver-check-20261109.log` and
`windows-gui-fairy-handler-{development,release}-20261109.json`.
Sources own behavior; ignored logs/images are local receipts.

## Limits

JSONL operations exercise real EDIT/BUTTON notifications and observed control
text without Computer Use. Physical input delivery, IME, DPI, appearance review,
Linux, remote CI, full repository tests, optimized compiler-host bootstrap,
complete self-host/fixed point and installers were not run for this Windows
example change. Generated C can still produce warnings under its existing
consumer policy; this is not a warning-free emitter claim.

Only the private example C bridge API changed; JSONL v1 and the compiler's
public CLI/language/FFI contracts did not. No DB, dependencies, workflows/runners,
repository-hygiene policy, version bump, push or deployment changes.
Earlier implementation commits: `2c574172` native warning scope and `153993c5`
Fairy click handling and source-backed regression.

## EN_CHANGE follow-up verification

- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure)$' -count=1 -v`: PASS, 18.512s.
  Both profiles pass; the negative cases modify only Fairy's result prefix,
  edit increment, or edit status and each correctly fails the native event smoke.
- After tightening `control.py` to assert exact edit/click counts throughout,
  `go test -mod=readonly ./internal/cli -run '^TestWindowsGUIExample$'
  -count=1 -v`: PASS, 12.283s, including development/release JSONL control.
- The previously authenticated authored Fairy CLI/companion builds the current
  example with `build --locked` and `build --locked --release`. The strengthened
  `control.py --verify` passes each resulting application: ten observations,
  edit notifications 1/2/3, five rejected invalid native commands and clean exit.
  This reuses the compiler host; it is not a fresh compiler bootstrap or fixed
  point verification.
- A persistent CLI session against the authored release app observes Korean
  input with `edits=1`, `clicks=0`, status `WM_COMMAND: EN_CHANGE`, and the initial
  result unchanged. Clicking yields `clicks=1` and the expected result. Replacing
  the input yields `edits=2`, `clicks=1`, `EN_CHANGE` and the previous result
  unchanged. The session then closes the application successfully.
- Scoped `git diff --check`: PASS. No public contract or compiler source changed;
  the limits above also apply to this follow-up.

| Builder | Development app | Release app |
|---|---:|---:|
| Go CLI, EN_CHANGE follow-up | 123,392 bytes | 120,320 bytes |
| Existing authored Fairy CLI/companion, low host | 138,752 bytes | 122,368 bytes |

Local receipts: `.cache/windows-gui-fairy-change-final-20261109.log`,
`windows-gui-fairy-change-control-final-20261109.log` and
`windows-gui-fairy-change-{development,release}-20261109.json`.
Implementation commit: `6c4fa8a2`; strengthened CLI/regression checks:
`4ca3e45b`. README and this execution note document the private bridge boundary.

## WM_CLOSE follow-up verification

- `go test -mod=readonly ./internal/cli -run '^TestWindowsGUIExample$'
  -count=1 -v`: PASS, 8.308s, before the regression extension.
- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure)$' -count=1 -v`: PASS, 19.723s.
  Development/release native smoke and JSONL control pass. The new negative
  case changes only Fairy's `resolve_close(window, 1)` to `resolve_close(window, 0)`:
  native smoke correctly fails with exit 5, and CLI requests twice receive
  `closed:false` with unchanged state and a live process. Input and click still
  work afterward. The existing three negative event cases also pass.
- `go run -mod=readonly ../../cmd/fairyc check` in the example: PASS,
  1 source / 4 MIR functions, lock refreshed.
- The existing authored Fairy CLI/companion builds the current app using
  `build --locked` and `build --locked --release`; each passes
  `python -B control.py --verify` with ten observations, five rejected native
  commands, `closed:true` and exit 0. The authored release app is 122,368 bytes;
  the Go-built development/release images are 123,904 / 120,320 bytes.
  Compiler source is unchanged; the authored compiler host was reused rather
  than bootstrapped again.
- A persistent CLI session on the authored release app sends Korean input,
  observes `edits=1`, clicks and observes `clicks=1` plus the expected result,
  then sends `close`, receives `{"ok":true,"closed":true}` and exits 0.
- `gofmt` and scoped `git diff --check`: PASS.

Receipts: `.cache/windows-gui-fairy-close-go-20261109.log` and
`windows-gui-fairy-close-{development,release}-20261109.json`.
Implementation: `c8153fa9`; regression: `209ff44d`.
Only the private example bridge gains a close-resolution function; existing
JSONL close replies retain the boolean `closed` field, including denial.
No public compiler API, DB, dependency, hygiene, runner, version, push or deploy
change. Full repository tests, Linux, hosted CI, compiler bootstrap/fixed point
and physical title-bar/Alt+F4/IME delivery were skipped for this scoped example
change. Normal close messages are exercised through real Win32 `WM_CLOSE`;
physical input delivery remains unverified.

## Desktop title-bar and Alt+F4 follow-up

The previous step tested native close messages. This follow-up additionally
exercises the Windows mouse/keyboard input stream through `SendInput`.

- `go run -mod=readonly ../../cmd/fairyc check` in the example: PASS,
  1 source / 4 MIR functions, lock unchanged (no Fairy source change).
- Using the existing authored Fairy CLI/companion, `build --locked` and
  `build --locked --release` both pass. Each image passes
  `python -B control.py --verify-desktop-close`: the native close button and
  Alt+F4 each report `systemClose:true`, Fairy approval, and process exit 0.
  The mouse receipt records `HTCLOSE` and three injected events; the keyboard
  receipt records four injected events. The standard `control.py --verify`
  also passes the updated authored release app.
- With `FAIRYC_RUN_DESKTOP_INTEGRATION=1` and `GOMAXPROCS=1`,
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|DesktopClose)$' -count=1 -v`:
  PASS, 28.018s. Go-built development/release apps pass both desktop actions
  alongside the existing native smoke and control checks. Changing only Fairy
  close policy to denial leaves the window open with unchanged state for both
  actual desktop actions; subsequent input/click still work. All four source
  mutation cases pass.
- `TestWindowsGUIDesktopClose` and the desktop denial probe are explicitly
  gated by `FAIRYC_RUN_DESKTOP_INTEGRATION=1`. Ordinary tests keep their existing
  control-message checks without sending desktop input. No CI workflow or
  runner was added or changed.
- `gofmt` and scoped `git diff --check`: PASS.

Local receipts: `.cache/windows-gui-desktop-go-20261109.log`,
`windows-gui-desktop-{development,release}-20261109.json` and
`windows-gui-desktop-baseline-20261109.json`.
Go development/release images: 124,416 / 120,832 bytes; authored development/release:
139,776 / 122,880 bytes. The authored compiler host was reused, not freshly bootstrapped.
Implementation: `1f6ee5eb`; verification: `a5850e7e`.

The desktop must be interactive, unlocked and idle. The helper restricts its
target to the child PID and Fairy class, checks foreground ownership and held
keys/buttons, obtains DPI-aware native title-bar rectangles, checks visibility
and `HTCLOSE`, and restores the cursor after the reply when it has not moved.
Background CLI activation may require briefly attaching and then detaching
input queues. Global desktop input can still race concurrent user activity;
foreground failure, blocked input or missing native observation fail instead
of substituting direct close messages. Physical hardware input, IME, visual
appearance, multi-monitor/DPI combinations and elevated/locked/headless targets
are not certified. Full repository tests, Linux, hosted CI and full compiler
bootstrap/fixed point were skipped for this Windows example/controller change.

Only the private example protocol adds `arm-close` and desktop-close metadata;
normal close replies and public compiler/language/FFI contracts are unchanged.
No DB, dependencies, repository-hygiene policy, runner, version, push or deploy
changes. The controller uses only Python's standard library; desktop input adds
no runtime dependency to the GUI executable.

## WM_SIZE and Fairy layout follow-up

- `go run -mod=readonly ../../cmd/fairyc check` in the example: PASS,
  1 source / 5 MIR functions, lock updated.
- The existing authored Fairy CLI/companion builds both profiles with
  `build --locked` / `build --locked --release`. Each image passes
  `python -B control.py --verify-resize`. Actual outer sizes are
  800x480, 400x240, 400x480, 400x480 and 640x360; processed resize counts are
  2/3/4/4/5, including an initial count of 1. Five actual child rectangles match
  Fairy's policy at every step. Height-only/repeated sizing preserves layout,
  resizing preserves Unicode text, result, status and edit/click counts;
  nine invalid native commands and five invalid Python values leave state
  unchanged. Apply and normal close still work afterward, with process exit 0.
- With `FAIRYC_RUN_DESKTOP_INTEGRATION=1` and `GOMAXPROCS=1`,
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|DesktopClose)$' -count=1 -v`:
  PASS, 26.930s. Both Go-built profiles pass native smoke, standard JSONL and
  resize verification. Five Fairy-source mutations fail native smoke as
  expected, including changing only `inset = 16` to 20. Existing close denial,
  actual title-bar clicking and Alt+F4 checks also pass.
- The standard `control.py --verify` passed the authored release layout
  implementation before extending the resize-specific checks. `gofmt` and
  scoped `git diff --check` pass.

Receipts: `.cache/windows-gui-size-go-20261109.log`,
`windows-gui-size-{development,release}-20261109.json` and
`windows-gui-size-baseline-20261109.json`.
Go development/release sizes are 129,024 / 124,416 bytes; authored sizes are
148,480 / 126,976 bytes. The existing authored compiler host was reused.
Implementation: `9525108a`; controller/regression: `fb37ab3b`.

The bridge uses existing scalar FFI for extents, individual moves and completion.
An integer-slice pointer-cast experiment was rejected; an indexed integer-borrow
cast trial hit an internal bootstrap invariant. Those paths are absent from
the final implementation; compiler source and language support were not changed.

The private example protocol adds `resize` and measured geometry fields.
Width/height describe the outer window, including its frame, in the app's
Win32 coordinate space. Native code validates bounds and applies Fairy's
supplied minimum; it performs no application layout calculation. The default
resize verifier uses real `SetWindowPos` changes and actual `WM_SIZE` delivery,
without injecting mouse/keyboard input. This is not a border-drag input test.
Native modal border dragging can defer Fairy's pulled layout until the modal
operation returns; continuous intermediate drag layout is not claimed.
Vertical placement remains fixed, matching the prior example.

Full repository tests, Linux, hosted CI, fresh compiler bootstrap/fixed point,
physical border dragging, IME and multi-monitor/DPI combinations were skipped
for this Windows example change. No public compiler API, DB, dependencies,
repository-hygiene policy, runner, version bump, push or deployment changes.

## Live modal border-drag follow-up

This supersedes the prior restriction about waiting for modal dragging to end.
The existing Fairy layout function is unchanged; only native scheduling and
optional desktop verification change.

- The existing authored Fairy CLI/companion builds development and release
  with `build --locked` / `build --locked --release`. Both apps pass
  `python -B control.py --verify-drag`: four actual bottom-right drag checkpoints
  report `sizing:true`, `mouseDown:true`, increased Fairy resize counts and live
  completions, correct actual child rectangles and preserved application state.
  Release samples include widths 531/573/633/553; development samples include
  531/582/633/557 in the app's coordinate space. Native entry/exit, post-release
  input/click and process exit 0 pass. Sizes/timing may differ with DPI and
  Windows' intermediate drag updates.
- The authored release app also passes existing `control.py --verify` and
  `--verify-resize` after the pump change.
- With `FAIRYC_RUN_DESKTOP_INTEGRATION=1` and `GOMAXPROCS=1`,
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|DesktopClose|LiveDrag)$' -count=1 -v`:
  PASS, 39.542s. Both Go-built profiles pass existing smoke, standard/resize
  JSONL checks, actual close-button/Alt+F4 input and actual live dragging.
  Five existing Fairy-source mutation controls pass. A new native control
  removes only the modal `SwitchToFiber` handoff: it builds successfully and
  the drag verifier exits 1 on stale Fairy layout observed before mouse release.
  This distinguishes live layout from a final update after the drag ends.
- `gofmt` and scoped `git diff --check`: PASS.

The CLI uses `SendInput` for the real mouse press/motions/release. It validates
the border with `HTBOTTOMRIGHT`, waits for native sizing entry, polls actual
geometry while held and observes sizing exit before cursor restoration. It
releases the button in its failure path too. Earlier exploratory attempts
stopped on lost foreground or cursor/entry synchronization; no such failure is
counted as a pass. The final verifier synchronizes against native modal entry
and samples distinct changes in the intended direction. Desktop tests remain
opt-in and require an idle, unlocked interactive desktop.

Receipts: `.cache/windows-gui-live-drag-go-20261109.log`,
`windows-gui-live-drag-{development,release}-20261109.json`,
`windows-gui-live-resize-baseline-20261109.json` and
`windows-gui-live-events-baseline-20261109.json`.
Go development/release images: 130,560 / 125,952 bytes.
Authored development/release images: 149,504 / 128,512 bytes.
Implementation: `1cb3e2dd`; controller/regression: `5e214196`.
The authored compiler host was reused rather than bootstrapped again.

Only the private example's telemetry and Python JSON command surface expand;
native application FFI signatures and Fairy source/lock are unchanged. No public
compiler/language API, DB, dependency, hygiene, runner, version, push or deploy
change. Full repository tests, Linux, hosted CI, full compiler bootstrap/fixed
point, physical mouse hardware, visual paint/frame-rate review, outline-only
dragging, elevated/locked/headless targets and other monitor/DPI combinations
were not certified by this Windows example task. The OS must deliver live
`WM_SIZE` notifications for intermediate layout; the bridge does not replace
the native drag behavior or change system-wide full-window-drag settings.

## WM_DPICHANGED and scaled layout follow-up

The example now enables Per-Monitor V2 before UI creation, using
`SetProcessDpiAwarenessContext`; Windows 10 1703 or newer is required. Fairy's
`scale` converts 96-DPI base units with integer rounding. `on_size` scales all
five control rectangles, the minimum window size and the requested font height.
The bridge caches the requested Segoe UI font and deletes replaced/final fonts.

For [WM_DPICHANGED](https://learn.microsoft.com/en-us/windows/win32/hidpi/wm-dpichanged),
C copies the new DPI and suggested RECT before yielding event 5. Fairy's
`on_dpi` passes the scaled minimum to `accept_dpi` before native `SetWindowPos`,
then runs layout. This fixes a reproduced 192-to-96 shrink where the old
800x480 minimum otherwise prevented the suggested 640x360 size. DPI work
precedes pending resize work; acknowledgement waits for completed layout.
Other input/click/close events retain their existing pending channel.

### Passed checks

- `go run -mod=readonly ../../cmd/fairyc check`: PASS, 1 source / 7 MIR
  functions, example lock refreshed.
- Existing authored Fairy CLI/companion `build --locked` and
  `build --locked --release`: PASS. Both images pass `control.py --verify-dpi`
  on the real GUI. Explicitly injected DPI values are 96/144/192/96/120;
  actual selected font heights are 12/18/24/12/15. Actual outer rectangles and
  five native child rectangles match Fairy policy; state and text survive,
  six malformed native requests leave state unchanged, Apply still works,
  and close exits 0. The authored release resize baseline also passed.
- With `GOMAXPROCS=1` and desktop injection disabled,
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|DPI)$' -count=1 -v`:
  PASS, 37.907s. Both Go-built profiles pass smoke, standard/resize and DPI
  verification. Five existing Fairy mutation controls and a new Fairy-only
  ignored-DPI-acceptance control pass. The latter builds successfully but
  `--verify-dpi` exits 1, proving C does not apply the suggested rectangle
  without Fairy's decision.
- `gofmt` and scoped `git diff --check`: PASS.

Go development/release images: 136,704 / 130,560 bytes. Authored images:
155,648 / 132,608 bytes. The previously authenticated authored compiler host
was reused; this is not a fresh bootstrap/fixed-point claim.
Feature commit: `286c602b`; minimum-size ordering/test transport: `5fb6eeb0`;
CLI and regression: `1f7236ff`.

### Limits and earlier failed desktop rechecks

Inventory reads one monitor at 125%, physical bounds 1920x1080. PMv2 and
initial `GetDpiForWindow=120` are observed. Controlled messages operate on real
windows/fonts, but **do not change the OS DPI**: receipts use
`verification:"injected-win32-dpi"` and `physicalMonitorTransitionTested:false`.
`dpi` is the last message DPI; `windowDpi` independently reports the OS value.
`dpiApplied` records the accepted rectangle, while `window` measures the actual
result. No display settings were changed and no cross-monitor transition was
performed. The reusable CLI inventories monitors but does not move the window
between them automatically.

At the DPI checkpoint, the broader desktop-enabled command selecting
`Example|EventFailure|DesktopClose|LiveDrag|DPI` **failed**, 69.667s: scoped DPI
and ordinary checks passed, while release close/drag checks timed out or failed
to enter sizing. A focused desktop recheck passed close in both profiles but
still failed a drag case. Diagnostics later observed `AttachThreadInput` to
the foreground thread failing with Windows error 5; attachment to the owned
GUI thread succeeded, but foreground acquisition was denied. These outcomes
did not establish a passing desktop-input regression run. At that checkpoint,
exploratory hover / button / input-queue probes were removed and the prior
desktop helper and native input path were unchanged. The targeted passing
command above deliberately does not inject desktop input. The stabilization
follow-up below supersedes these desktop-input failures.

Receipts: `.cache/windows-gui-dpi-scoped-20261109.log`,
`windows-gui-dpi-{development,release}-20261109.json`,
`windows-gui-dpi-baseline-20261109.json`, and the failed broader run
`windows-gui-dpi-go-20261109.log` / focused `windows-gui-dpi-desktop-20261109.log`.
Full repository tests, Linux, hosted CI, full compiler bootstrap/fixed point,
physical cross-monitor/system-scale changes, visual raster/paint quality and
other monitor/DPI combinations were not certified by this Windows example task.

Only private example FFI/JSONL surfaces expand (`dpi-test`, DPI/selected-font
telemetry and read-only `dpi_inventory.py`). Public compiler/language/SDK API,
DB, dependencies, repository-hygiene policy, workflows/runners, versions, push
and deployment are unchanged.

## Desktop-input stabilization follow-up

The close/drag helper now retries foreground activation for up to two seconds.
An `AttachThreadInput` failure against an unrelated foreground queue is recorded
and does not suppress the independent owned-window activation attempt. Every
input still requires the expected PID/HWND and foreground; temporary queue
attachments are detached before input. Activation must remain stable for 100 ms.
Before activation and input, held buttons/modifiers/F4 may clear for up to two
seconds. Persistent held input fails without injection; the helper never releases
a user's held input. Receipts expose `focusAttempts` and `attachErrors`.

Mouse input now waits for a fresh `WM_NCMOUSEMOVE` at the expected non-client
hit and the subsequent native press before releasing or dragging. The read-only
`WM_APP+1` probe observes a hover counter, hit code, press counter and message;
it cannot initiate a close or resize. Receipts expose `hoverDelivered`,
`pressDelivered`, `pressMessage` and `pressHit`. The native press hit must match
`HTCLOSE` or `HTBOTTOMRIGHT`; a drag requires `WM_NCLBUTTONDOWN`. Missing probes
reject stale executables with a rebuild instruction. Waits are bounded, and all
failure paths release the helper's own injected mouse button. The existing
four-point live layout and sizing-exit checks remain required.

`WM_CLOSE` now hands control from the pump fiber to Fairy before returning to
Windows, including during native caption-button tracking. Fairy still owns
approval/denial and C performs the requested destruction. Armed desktop replies
also require `closeHandoff:true`, observed while the native callback is active.
This prevents a delayed close from being counted as a synchronous handoff.
Ordinary `close` replies and the public compiler/SDK interfaces are unchanged.

### Verification

- With `GOMAXPROCS=1` and `FAIRYC_RUN_DESKTOP_INTEGRATION=1`, the final command
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|DesktopClose|LiveDrag|DPI)$' -count=1 -v`
  passed in **94.380s**. Development/release smoke, ordinary controls/resize,
  real desktop close, live drag and injected DPI checks passed together.
  Fairy close denial remained interactive after both real close methods.
  Removing only the close handoff made the desktop verifier fail; the existing
  missing-live-layout and Fairy policy/DPI mutation controls also passed.
  Go-built development/release images: 137,216 / 130,560 bytes.
  Receipt: `.cache/windows-gui-input-final-go-20261109.log`.
- The reused authored Fairy CLI/companion successfully rebuilt the release
  example with `build --locked --release` (133,120 bytes). Three consecutive
  iterations passed real title-bar close, Alt+F4 and four-point live dragging.
  Every close recorded `systemClose:true`, `closeHandoff:true` and process exit 0;
  every drag observed correct Fairy layout while the button was still held.
  Receipt: `.cache/windows-gui-input-stability-20261109.json`, including source,
  image and compiler SHA-256 values; raw summary:
  `.cache/windows-gui-input-stability-20261109.log`.

The earlier broad DPI failure remains recorded above as history. During this
follow-up, an intermediate combined run failed in 51.005s on close-denial press
delivery and lost foreground. Fresh-hover observation and activation settling
were added, after which the same combined suite passed in 55.100s. A subsequent
authored desktop attempt stopped on held-input detection before injection;
bounded key-release waiting was then added. These stopped attempts are not
counted as passes. The three-iteration authored receipt uses the final helper.

This remains OS-injected input on one unlocked interactive Windows desktop at
125% scaling. Concurrent user input can still invalidate foreground ownership;
locked/headless/elevated desktops, physical hardware and physical cross-monitor
DPI transitions are not certified. Full repository tests, Linux, hosted CI and a
fresh compiler bootstrap were not run because this task is limited to the Win32
example/controller. No DB, dependency, hygiene-policy, runner/workflow, version,
push or deployment change is included.

Implementation commits: `beda2692` (foreground queue access), `0bed02ac`
(native delivery and synchronous close), `afb9e3fe` (bounded key-release wait).

## Fairy keyboard policy: Enter and Esc

The native pump captures Return/Esc `WM_KEYDOWN` for the window or its children
before `IsDialogMessageW`, avoiding the default-button fallback. Event 6 passes
the virtual key and modifier/previous-state flags to Fairy via `key(axis)`.
Fairy's `on_key` maps plain Return to action 1 (the existing Apply handler) and
plain Esc to action 3 (the existing close policy). Nonzero flags choose action 0.
The counters and these decisions live in Fairy; C does not call `BM_CLICK` or
destroy the window as a keyboard policy. Tab and other native keys retain their
existing dialog handling. Input and Apply focus are supported.

Enter shares empty-input/Unicode/result construction with mouse Apply while
using status resources 110/111 (`WM_KEYDOWN: Enter`, including empty input).
Mouse Apply retains 106/109. `key_done` mirrors Fairy's processed count/action
and acknowledges only after the operation, including after Esc destruction or
denial. State exposes `keyCode`, `keyFlags`, `keyFocus`, `keyAction`, `keys`,
`awaitingKey` and `closed:false`; a closed key reply has `closed:true`.
The flags are Shift=1, Ctrl=2, Alt=4, Windows=8 and previous-key-state=16.

The private `arm-key<TAB>code<TAB>focus` command only arms observation and
focuses input 201 or Apply 202; accepted codes are Return 13 and Esc 27. The
controller then sends actual `SendInput` down/up pairs and verifies foreground,
native focus, observed code/flags/count and completed Fairy processing. It
never fabricates a keyboard/action message for these checks. The helper's held
input guard now includes Return/Esc. Interactive JSON supports `command:"key"`
with `key:"enter"`, `"escape"` or `"ctrl-enter"` and optional
`focus:"input"|"apply"`. See the example README for commands.

### Keyboard verification results

- `go run -mod=readonly ../../cmd/fairyc check`: PASS, 1 source / 8 MIR
  functions; the example lock was refreshed without changing package versions.
- Reused authored Fairy CLI/companion `build --locked` and
  `build --locked --release`: PASS. Both images passed
  `python -B control.py --verify-keyboard`: empty Enter, Unicode Enter,
  ignored Ctrl+Enter, Enter on Apply, no delayed duplicate action, and Esc exit
  0. Development/release images: 161,280 / 138,240 bytes.
- With `GOMAXPROCS=1` and `FAIRYC_RUN_DESKTOP_INTEGRATION=1`,
  `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUIKeyboard(Policy)?$' -count=1 -v`: PASS, **27.560s**.
  Development/release key verification and three Fairy-only broken-policy
  controls passed. Changing only Enter mapping, modifier policy or Esc mapping
  makes the verifier fail for that behavior. A fourth Fairy-only close-denial
  control keeps the window interactive after Esc from both controls, rejects
  malformed native arm requests without mutation, and applies new input via
  real Enter afterward. Go-built images: 141,312 / 135,168 bytes.
- `gofmt` and scoped `git diff --check`: PASS.

Receipts: `.cache/windows-gui-keyboard-focused-final-20261109.log`,
`windows-gui-keyboard-authored-development-20261109.json`,
`windows-gui-keyboard-authored-release-final-20261109.json` and
`windows-gui-keyboard-summary-20261109.json` (source/compiler/image hashes).

### Broader desktop observations and limits

The intermediate combined selection
`^TestWindowsGUI(Example|EventFailure|DesktopClose|LiveDrag|DPI|Keyboard|KeyboardPolicy)$`
did not pass as a single run. The first attempt (68.976s) stopped ordinary smoke
with exit 5; an isolated diagnostic copy then passed three smoke executions.
The later attempt (78.502s) passed ordinary smoke/control, both desktop-close
profiles, live drag, DPI and the keyboard tests, but the existing close-denial
mouse probe observed native press hit 11 instead of `HTCLOSE` and aborted.
No cause is claimed from these observations. Retesting only
`^TestWindowsGUIEventFailure$/^close-policy$` with desktop input enabled passed
in **4.259s**. Log: `.cache/windows-gui-keyboard-close-policy-recheck-20261109.log`.
These scoped passes are not represented as a green combined run.

One later authored release receipt-collection attempt also returned nonzero
without preserving the controller JSON; its cause is unclassified. The direct
final authored release command passed and its JSON is preserved above. No
failed/stopped attempt is counted as a pass.

The test desktop remains one monitor at 125%. Real OS-injected shortcuts were
tested; physical keyboard hardware, held-key auto-repeat timing, IME composition,
locked/headless/elevated desktops and physical monitor/DPI transitions were not
certified. Full repository tests, Linux, hosted CI and a fresh compiler bootstrap
were omitted because this task changes only the Windows example/controller.
Desktop input still requires an idle, unlocked session.

Only private example FFI/JSONL and resource strings expand. Public compiler,
language and SDK API, DB, dependencies, hygiene policy and CI runners/workflows
are unchanged; no push, deploy or version bump is included.
Commits: `a128c1e2` (Fairy key policy), `76baac6e` (keyboard status),
`99dd4b69` (actual-key CLI and policy regressions).
