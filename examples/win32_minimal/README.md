# Minimal Win32 GUI

From this folder on Windows x64 with MSVC and a Windows SDK installed:

```powershell
fairyc build --release
.\build\win32-minimal.exe
```

`fairyc run --release` also opens the window. Close it normally, or use
`fairyc run --release -- --smoke` to exercise creation, painting, the message
loop and timer-driven destruction automatically.

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
