---
title: C Interop and Win32 Apps
description: Interfacing with C via @repr(C), unsafe extern C fn, callbacks, and real Win32 examples (win32_minimal and win32_hash).
---

Fairy C interfaces with C libraries and OS APIs directly through C-compatible data layouts, explicit `unsafe extern C fn` declarations, and native Win32 bridge builds.

## C ABI Building Blocks

1. **C-compatible struct layout (`@repr(C)`)**: Ensures field order and alignment match the target C ABI (`examples/win32_hash/src/writer.fy`):

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

2. **Foreign function declarations & `unsafe` blocks**: Raw pointer calls and foreign C functions require `unsafe extern C fn` (optionally with `@link_name("...")`) and must be called inside `unsafe { ... }` (`E5101`):

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

3. **Synchronous C callbacks (`examples/c_interop_callback`)**: Passing a synchronous Fairy callback to C works across raw pointer boundaries, while passing safe `ref T` borrows directly in `extern C` signatures (`E5102`) or letting C `longjmp`/exceptions unwind through Fairy frames (`E5103`) is forbidden.

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## Windows GUI Bridge Configuration (`fairy.windows.json`)

For Win32 desktop applications, a `fairy.windows.json` file (`schema: "fairyc.windows-build/v1"`) pairs Fairy application code with native C Win32 loop/UI shims and `.rc` resources:

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## Real Example 1: `examples/win32_minimal`

`examples/win32_minimal` splits responsibilities cleanly:

- **Native C bridge (`native/window.c`, `native/ime.c`)**: Owns `HWND` creation, the Win32 message loop, `IMM32` Hangul/IME handling, and font/DPI APIs.
- **Fairy application (`src/main.fy`)**: Owns DPI-scaled control layout (`on_size`, `on_dpi`), key/Tab navigation (`on_key`, `on_tab`), and message formatting (`on_click`) using fixed byte buffers and slices.

Build and verify locally on Windows:

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

### Measured Size Record (`windows-gui-slim-20261111`)

When stripping opt-in test/JSONL pipe diagnostics into separate distribution wrappers (`build_distribution.py`), the repository recorded these exact Windows x64 MSVC release sizes:

| Compiler | Profile | Ordinary Image | Distribution Image |
| --- | --- | ---: | ---: |
| Go CLI | release | 141,824 B | 121,344 B |
| Authored CLI | release | 144,384 B (141 KiB) | 123,904 B (121 KiB) |

These distribution builds still link static CRT and import `USER32`, `GDI32`, `IMM32`, and `KERNEL32` system DLLs.

## Real Example 2: `examples/win32_hash` (Fairy Hash)

`examples/win32_hash` (Fairy Hash) is a desktop + CLI SHA-256 file integrity utility using `std::fs`, `std::bytes`, `std::process`, and `std::io` alongside a thin Win32 C shim (`native/window.c`, `native/reader.c`, `native/writer.c`).

- **Effects in `src/main.fy`**: `main()` declares `@effects(console, filesystem, clock)`.
- **CLI mode**: Outputs a UTF-8 JSON report without launching a GUI window:

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

### Measured Size Record (`windows-hash-package-20261113`)

In the `2026-11-13` portable release package verification (`fairyc-go.exe build --locked --release`, embedding a 9-size `.ico`, `VERSIONINFO`, and Common Controls v6 manifest while excluding test/dev scripts):

- `FairyHash.exe` (with embedded icon/version resources): **196,096 bytes (191.5 KiB)** (up from **188,416 bytes** before adding those resources)
- Portable unsigned ZIP (`fairy-hash-0.1.0-windows-x64.zip`): **105,640 bytes (103.16 KiB)**

> **Important scope note:** These binary sizes (`121 KiB` and `191.5 KiB`) come from specific Windows x64 MSVC release builds of `win32_minimal` and `win32_hash` with their exact C bridges and static CRT. They are project measurements under those conditions—not a general size or performance guarantee for arbitrary Fairy C programs.
