---
title: 安装与环境配置
description: 从源码构建 Stage 0 fairyc 编译器，并确认你的 C 工具链已准备就绪。
---

目前尚未提供独立安装向导（`.msi` 等）或 Linux 预编译压缩包，但你可以直接下载 <strong>GitHub Releases 上的 Windows x64 预览版 ZIP</strong>，或者 <strong>从仓库源码构建最新 `main` 分支编译器</strong>。

| 安装方式 | 对应版本 / 基线 | 适用场景与范围 |
| --- | --- | --- |
| <strong>方式 A：预览版 ZIP</strong> | `v0.1.0-preview.1`（`2026-10-25`，提交 `682d6591`） | 无需安装 Go/Bun，在 Windows x64 上快速试用核心控制台与语言子集 |
| <strong>方式 B：从源码构建</strong> | `main` 开发构建（`sdk: 0.0.0-stage0.87`，提交 `e1d48cb7`） | 需要 `new --template win32`、`package` 等最新桌面特性或 Linux 构建时 |

> <strong>提示</strong>：`v0.1.0-preview.1` 属于早期快照；`fairyc new --template win32` 和 `fairyc package` 等近期新增功能请使用<strong>方式 B（源码构建）</strong>。

## 准备原生 C 工具链（通用前提）

`fairyc check` 与解释器模式的 `fairyc test` 不调用 C 编译器，但生成原生可执行文件的 `fairyc build`、`fairyc run` 与 `fairyc test --native` 需要宿主平台 C 工具链：

- <strong>Windows</strong>：安装带有 <strong>MSVC x64 C/C++ 编译器</strong> 和 <strong>Windows SDK</strong> 的 Visual Studio / Build Tools。（若使用 Win32 桌面资源与 `fairyc package`，还需在 `PATH` 中安装 <strong>Python 3.11+</strong>。）
- <strong>Linux</strong>：确保 `PATH` 中有可用的 `gcc` 或 `clang`。

## 方式 A：使用 Windows 预览版 ZIP（`v0.1.0-preview.1`）

若不想安装 Go/Bun，仅在 Windows x64 上试用核心 CLI，可前往 [GitHub Releases（`v0.1.0-preview.1`）](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1) 下载 `fairyc-windows-amd64-preview-candidate.zip` 及其 `.sha256` 校验文件：

```powershell
# 校验 SHA-256 后解压并查看版本
Get-FileHash .\fairyc-windows-amd64-preview-candidate.zip -Algorithm SHA256
Expand-Archive .\fairyc-windows-amd64-preview-candidate.zip -DestinationPath .\fairyc-preview
.\fairyc-preview\fairyc.exe --version
```

## 方式 B：从仓库源码构建最新 `fairyc`

若需使用最新 Stage 0 命令（`--template win32`、`fairyc package`、Linux GNU x64 构建等），请先准备 <strong>Git、Go 和 Bun</strong>，然后克隆 [fairyc GitHub 仓库](https://github.com/0disoft/fairyc)并进入根目录：

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

在 `fairyc` 仓库根目录下运行：

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

在 Linux 上，请将输出参数改为 `-o build/fairyc`。随后可以将 `build/` 添加到 `PATH` 环境变量中（或者直接通过 `./build/fairyc.exe` 调用）。

## 验证构建

运行 `--version` 和 `targets`：

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

`--version` 输出示例：

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

运行 `fairyc targets` 时，`x86_64-windows-msvc` 和 `x86_64-linux-gnu` 都会显示 `supported: no`。这仅说明 Stage 0 阶段的正式规范覆盖尚不完整——只要你的 C 工具链已安装，就可以正常进行本地构建和运行。
