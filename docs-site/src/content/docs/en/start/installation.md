---
title: Installation and Setup
description: Build the Stage 0 fairyc compiler from source and make sure your C toolchain is ready.
---

There is no standalone installer wizard (`.msi`) or prebuilt Linux archive yet. You can either download the **experimental Windows x64 preview ZIP** from GitHub Releases or **build the latest `main` compiler from source**.

| Install Method | Target Version / Snapshot | Best For |
| --- | --- | --- |
| **Option A: Preview ZIP** | `v0.1.0-preview.1` (`2026-10-25`, commit `682d6591`) | Trying core console builds on Windows x64 without Go or Bun |
| **Option B: Build from source** | `main` development build (`sdk: 0.0.0-stage0.87`, commit `e1d48cb7`) | Latest features including `new --template win32`, `package`, and Linux builds |

> Note: Because `v0.1.0-preview.1` is an earlier snapshot, recent additions such as `fairyc new --template win32` and `fairyc package` require **Option B (Build from source)**.

## Host C Toolchain (Shared Prerequisite)

`fairyc check` and interpreter-mode `fairyc test` run without a C compiler, whereas native `fairyc build`, `fairyc run`, and `fairyc test --native` require a platform C toolchain:

- **Windows**: Visual Studio / Build Tools with the **MSVC x64 C/C++ compiler** and **Windows SDK**. (Win32 desktop resources and `fairyc package` also require **Python 3.11+** on `PATH`.)
- **Linux**: `gcc` or `clang` available on `PATH`.

## Option A: Windows Preview ZIP (`v0.1.0-preview.1`)

To try the core Windows x64 CLI without installing Go or Bun, download `fairyc-windows-amd64-preview-candidate.zip` and its `.sha256` file from [GitHub Releases (`v0.1.0-preview.1`)](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1):

```powershell
# Verify SHA-256, extract, and check version
Get-FileHash .\fairyc-windows-amd64-preview-candidate.zip -Algorithm SHA256
Expand-Archive .\fairyc-windows-amd64-preview-candidate.zip -DestinationPath .\fairyc-preview
.\fairyc-preview\fairyc.exe --version
```

## Option B: Build `fairyc` from Source (`main`)

To use the latest Stage 0 commands (`--template win32`, `fairyc package`, Linux GNU x64 builds), ensure **Git, Go, and Bun** are installed, then clone the [fairyc GitHub repository](https://github.com/0disoft/fairyc) and move into the repository root:

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

From the `fairyc` repository root:

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

On Linux, use `-o build/fairyc`. Add `build/` to your `PATH` (or invoke `./build/fairyc.exe` directly).

## Verifying the Build

Run `--version` and `targets`:

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

Example `--version` output:

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

When you run `fairyc targets`, both `x86_64-windows-msvc` and `x86_64-linux-gnu` display `supported: no`. That simply reflects incomplete formal spec coverage in Stage 0—your local builds and runs are ready as soon as your C toolchain is installed.
