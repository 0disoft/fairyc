---
title: 安装与环境配置
description: 从源码构建 Stage 0 fairyc 编译器，并确认你的 C 工具链已准备就绪。
---

目前尚未提供独立安装包或预编译二进制下载。你需要从仓库源码自行构建 `fairyc`，并搭配你所在平台的 C 编译器使用。

## 前置依赖

1. <strong>Git、Go 和 Bun</strong>：用于编译 Stage 0 Go 编译器（`tools/run-go.ts` 会通过 Bun 封装调用 Go）。
2. <strong>原生 C 编译器</strong>（运行 `fairyc build`、`fairyc run` 和 `fairyc test --native` 时需要）：
   - <strong>Windows</strong>：安装带有 <strong>MSVC x64 C/C++ 编译器</strong> 和 <strong>Windows SDK</strong> 的 Visual Studio / Build Tools。
   - <strong>Linux</strong>：确保 `PATH` 中有可用的 `gcc` 或 `clang`。

> <strong>提示</strong>：`fairyc check` 和解释器模式的 `fairyc test` 不会调用 C 编译器，因此即使尚未配置 MSVC 或 GCC，你也可以先对代码进行类型检查。

## 克隆并进入源码仓库

如果本地尚未获取源码，请先从 [fairyc GitHub 仓库](https://github.com/0disoft/fairyc)克隆并进入仓库根目录：

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

## 构建 `fairyc`

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
