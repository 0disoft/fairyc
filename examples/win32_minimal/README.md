# Minimal Win32 GUI

From this folder on Windows x64 with MSVC and a Windows SDK installed:

```powershell
fairyc build --release
.\build\win32-minimal.exe
```

`fairyc run --release` also opens the window. Close it normally, or use
`fairyc run --release -- --smoke` to exercise creation, painting, the message
loop and timer-driven destruction automatically. Smoke mode also verifies real
edit notifications and button clicks, empty input, Unicode results, repeated
edits, the input length limit and rejection of a command without its control
handle. It exits with code 5 if an event check fails.

Type into **Message** and click **Apply**. The result displays the entered
Unicode text; an empty message shows a prompt. The status line identifies
`WM_COMMAND / EN_CHANGE` from editing and `WM_COMMAND / BN_CLICKED` from the
button. Input is bounded to 127 UTF-16 code units. Tab moves between the edit
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
`BN_CLICKED`. The ordinary WndProc handles both and the reply reads actual
control text. This verifies programmatic control events; it does not certify
physical keyboard/mouse delivery, IME, screen rendering or accessibility.

The GUI enables this private example protocol only with `--automation-stdio`
and inherited pipe stdin/stdout. The Python controller owns that process and
closes it on stdin EOF or error. It opens no socket, contacts no service and
does not discover or control other running windows. Native requests use LF
lines `state`, `input<TAB>UTF-8`, `click`, `close`; replies are JSON lines.
The native buffer is bounded to 1023 bytes and validates UTF-8, NUL and input
length before mutation. The Python JSON request limit is 4096 bytes; multiline
input and unknown fields reject. There is one outstanding request at a time.
An alternate built example image can be selected with `--exe <path>`.

Fairy owns `main()`. A small C bridge owns the Win32 window procedure and
message loop. The `.rc` string table is compiled by SDK `rc.exe`. No UI
framework or browser runtime is bundled; the executable uses Windows system
DLLs and the static CRT. This prototype does not expose arbitrary Fairy
callbacks to Windows or generate Win32 SDK bindings.

`fairy.windows.json` selects the GUI subsystem and explicit native inputs.
See [ADR 0552](../../docs/adr/0552-minimal-windows-gui-build.md) for bounds and
the native-code trust boundary. Headers and resource assets are build-time
dependencies; the resulting example executable can run on its own.
Local C includes and resource asset paths should be written relative to the
project root; native source snapshots are compiled in temporary storage.
