---
title: C Interop and Win32 Apps
description: Call C functions with @repr(C) and unsafe extern C fn, and explore real Win32 examples (win32_minimal and win32_hash).
---

When you want to reuse existing C libraries or call Windows Win32 APIs directly, Fairy C provides explicit C-layout and foreign-function primitives.

## 1. Three C-Interop Building Blocks

### `@repr(C)` for Struct Layouts
Normally the compiler is free to reorder fields in a struct. Add `@repr(C)` when sharing a struct across a C boundary so layout and alignment match C (`examples/win32_hash/src/writer.fy`):

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

### `unsafe extern C fn` and `unsafe { ... }`
Declare foreign C functions with `unsafe extern C fn` and invoke them inside `unsafe { ... }`. Putting the `unsafe` call inside a `resource`'s `cleanup` block keeps the rest of your Fairy C code clean and safe:

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

When the exported C symbol name differs from the Fairy C declaration name, attach `@link_name("c_symbol")`:

```c
@link_name("strlen")
unsafe extern C fn usize c_strlen(ptr u8 text);
```

### Passing Callbacks to C (`examples/c_interop_callback`)
Synchronous Fairy callbacks can be passed across raw pointer signatures (`unsafe extern C fn` cannot take safe `ref T` directly—`E5102`):

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## 2. Win32 Bridge Setup (`fairy.windows.json`)

Desktop examples use a `fairy.windows.json` configuration (`fairyc.windows-build/v1`) to link C shims, `.rc` resources, and Windows libraries:

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## 3. Example 1: `examples/win32_minimal`

A minimal Win32 GUI splitting OS plumbing from application logic:

- **C bridge (`native/window.c`, `native/ime.c`)**: Window creation, Win32 event loop, Hangul/IME composition, and DPI/font APIs.
- **Fairy C (`src/main.fy`)**: DPI-scaled layout (`on_size`, `on_dpi`), keyboard/Tab handling (`on_key`, `on_tab`), and text processing (`on_click`).

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

In the `windows-gui-slim-20261111` distribution-wrapper measurement (stripping opt-in pipe/smoke test helpers while keeping static CRT and Windows DLL imports), the Windows x64 MSVC release binary measured `121,344` bytes with the Go CLI and `123,904` bytes (`121 KiB`) with the authored CLI.

## 4. Example 2: `examples/win32_hash` (Fairy Hash)

A combined desktop and `--hash <file> [report]` CLI SHA-256 integrity utility combining `std::fs`, `std::bytes`, `std::process`, `std::io`, and `@effects(console, filesystem, clock)` with a thin C bridge (`native/window.c`, `native/reader.c`, `native/writer.c`).

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

In the `2026-11-13` Windows x64 locked release build (`fairyc-go.exe build --locked --release`, embedding a 9-size `.ico`, `VERSIONINFO`, and Common Controls v6 manifest), `FairyHash.exe` measured `196,096` bytes (`191.5 KiB`; `188,416` bytes prior to embedding those icon/version resources).

> These size measurements apply to those specific Windows x64 MSVC release builds with their C shims and static CRT—they are concrete example figures, not a blanket guarantee for all Fairy C binaries.

## 5. End-to-End Win32 App Workflow (`new --template win32` to `package`)

With a current Stage 0 source build (`main`), you can scaffold a Win32 desktop project, customize its icon and version metadata, build a release executable, and package an unsigned portable ZIP. (Building and packaging desktop resources require Windows x64, MSVC + Windows SDK, and **Python 3.11+** on `PATH`.)

1. **Create the Win32 project**: Generates `fairy.toml`, `app.toml`, `fairy.windows.json`, `native/app.ico`, `native/app.rc`, and starter source/tests.

   ```powershell
   fairyc new my-tool --uid example.my-tool --template win32
   cd my-tool
   ```

2. **Configure icon, branding, and version**:
   - **Icon and branding (`app.toml`)**: Edit `product_name`, `company_name`, `description`, `executable_name`, and `icon = "native/app.ico"` (a project-relative `.ico` path).
   - **Version (`fairy.toml`)**: Set `[package] version = "0.1.0"`, which feeds `VERSIONINFO` and portable archive metadata.

   ```toml
   [app]
   product_name = "My Tool"
   company_name = "Example Co"
   description = "My Tool desktop utility"
   executable_name = "my-tool.exe"
   package_name = "my-tool"
   assembly_name = "example.my-tool"
   icon = "native/app.ico"
   ```

3. **Check, test, and build the release binary**:

   ```powershell
   fairyc check
   fairyc test --locked --file tests/domain.fy
   fairyc build --locked --release
   ```

4. **Create and validate the portable ZIP (`package`)**: Outputs an unsigned local archive and verification receipt under `dist/`.

   ```powershell
   fairyc package --locked --json
   ```
