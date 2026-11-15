---
title: C 互操作与 Win32 应用
description: 学习如何使用 @repr(C) 和 unsafe extern C fn 调用 C 函数，并了解真实的 Win32 示例（win32_minimal 与 win32_hash）。
---

当你想复用现有的 C 库或直接调用 Windows Win32 API 时，Fairy C 提供了显式的 C 内存布局和外部函数原语。

## 1. C 互操作的三大基础构件

### 用 `@repr(C)` 控制结构体布局
通常编译器可以自由重排结构体中的字段顺序。跨 C 边界共享结构体时，请加上 `@repr(C)` 以确保内存布局和对齐方式与 C 一致（见 `examples/win32_hash/src/writer.fy`）：

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

### `unsafe extern C fn` 与 `unsafe { ... }`
使用 `unsafe extern C fn` 声明外部 C 函数，并在 `unsafe { ... }` 块中调用它们。把 `unsafe` 调用封装进 `resource` 的 `cleanup` 块里，可以让其余 Fairy C 代码保持干净与安全：

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

当外部 C 符号名与 Fairy C 中的声明名称不同时，可以使用 `@link_name("c_symbol")` 指定实际链接符号：

```c
@link_name("strlen")
unsafe extern C fn usize c_strlen(ptr u8 text);
```

### 向 C 传递回调函数（`examples/c_interop_callback`）
可以通过原始指针签名向 C 传递同步的 Fairy 回调函数（`unsafe extern C fn` 不能直接接收安全的 `ref T`——否则会报 `E5102`）：

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## 2. Win32 桥接配置（`fairy.windows.json`）

桌面示例使用 `fairy.windows.json` 配置（`fairyc.windows-build/v1`）来链接 C 桥接源码、`.rc` 资源以及 Windows 系统库：

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## 3. 示例 1：`examples/win32_minimal`

一个将操作系统底层细节与应用逻辑拆分的极简 Win32 GUI 示例：

- <strong>C 桥接层（`native/window.c`、`native/ime.c`）</strong>：窗口创建、Win32 事件循环、韩文/IME 输入法组合以及 DPI/字体 API。
- <strong>Fairy C 层（`src/main.fy`）</strong>：按 DPI 缩放的布局（`on_size`、`on_dpi`）、键盘/Tab 处理（`on_key`、`on_tab`）以及文本处理（`on_click`）。

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

在 `windows-gui-slim-20261111` 发布封装测量中（剥离了可选的管道/冒烟测试辅助代码，同时保留静态 CRT 和 Windows DLL 导入），Windows x64 MSVC release 二进制文件使用 Go CLI 构建为 `121,344` 字节，使用自举 CLI（authored CLI）构建为 `123,904` 字节（`121 KiB`）。

## 4. 示例 2：`examples/win32_hash`（Fairy Hash）

一个兼具桌面界面与 `--hash <file> [report]` 命令行模式的 SHA-256 完整性校验工具，将 `std::fs`、`std::bytes`、`std::process`、`std::io` 和 `@effects(console, filesystem, clock)` 与一层轻薄的 C 桥接（`native/window.c`、`native/reader.c`、`native/writer.c`）结合在一起。

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

在 `2026-11-13` 的 Windows x64 锁定发布构建（`fairyc-go.exe build --locked --release`，嵌入了 9 种尺寸的 `.ico` 图标、`VERSIONINFO` 和 Common Controls v6 清单）中，`FairyHash.exe` 体积为 `196,096` 字节（`191.5 KiB`；在嵌入这些图标与版本资源之前为 `188,416` 字节）。

> 上述体积测量结果仅针对这些特定的 Windows x64 MSVC release 构建及其 C 桥接代码和静态 CRT——它们是具体示例的实测数据，并非对所有 Fairy C 二进制文件体积的统一保证。

## 5. Win32 应用端到端工作流（从 `new --template win32` 到 `package`）

使用最新源码构建（`main`）的编译器，你可以直接通过 `--template win32` 生成桌面应用骨架、配置图标与版本，并构建及打包为本地未签名便携 ZIP。（桌面资源准备与打包步骤需要 Windows x64、MSVC + Windows SDK，以及在 `PATH` 中安装 <strong>Python 3.11+</strong>。）

1. <strong>创建 Win32 项目</strong>：自动生成 `fairy.toml`、`app.toml`、`fairy.windows.json`、`native/app.ico`、`native/app.rc` 以及初始源码和测试。

   ```powershell
   fairyc new my-tool --uid example.my-tool --template win32
   cd my-tool
   ```

2. <strong>配置图标、应用信息与版本</strong>：
   - <strong>图标与应用信息（`app.toml`）</strong>：修改 `product_name`、`company_name`、`description`、`executable_name` 以及 `icon = "native/app.ico"`（指向项目内的相对 `.ico` 路径）。
   - <strong>版本号（`fairy.toml`）</strong>：`[package]` 下的 `version = "0.1.0"` 会同时用于 `VERSIONINFO` 资源和打包元数据。

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

3. <strong>检查、测试并执行发布构建</strong>：

   ```powershell
   fairyc check
   fairyc test --locked --file tests/domain.fy
   fairyc build --locked --release
   ```

4. <strong>生成并校验便携 ZIP（`package`）</strong>：在 `dist/` 目录下生成未签名本地 ZIP 归档并输出校验报告。

   ```powershell
   fairyc package --locked --json
   ```
