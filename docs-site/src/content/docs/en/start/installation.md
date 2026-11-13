---
title: Installation and Setup
description: How to build the Stage 0 fairyc compiler from source and prepare a native C toolchain.
---

Currently, Fairy C does not ship a prebuilt installer or standalone download archive. You build the `fairyc` CLI directly from the repository source and pair it with your platform's C compiler and SDK.

## Prerequisites

1. **Git, Go, and Bun**: Needed to build the Stage 0 Go compiler prototype (`tools/run-go.ts` invokes Go cleanly via Bun).
2. **A native C toolchain** for `fairyc build`, `fairyc run`, and `fairyc test --native`:
   - **Windows (`x86_64-windows-msvc`)**: Install Visual Studio / Build Tools with the **MSVC x64 C/C++ compiler** and **Windows SDK**. Open a developer shell (or ensure MSVC and Windows SDK tools are discoverable on `PATH` / standard installation paths).
   - **Linux (`x86_64-linux-gnu`)**: Ensure `gcc` or `clang` is available on `PATH`.

> **Note:** Check and interpreter-mode `fairyc test` can inspect packages without linking a native executable, but `fairyc build`, `fairyc run`, and `fairyc test --native` require a working MSVC/Windows SDK (on Windows) or GCC/Clang (on Linux).

## Build the Compiler CLI

From the `fairyc` repository root:

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

On Linux, omit the `.exe` suffix (`-o build/fairyc`). You can then add `build/` to your `PATH` or invoke `./build/fairyc.exe` (or `./build/fairyc`) directly.

## Verify Your Setup

Run `--version` and `targets` to check the built binary:

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

Example output observed on Stage 0:

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

`fairyc targets` lists both target specifications (`x86_64-linux-gnu` and `x86_64-windows-msvc`) with `supported: no` and their missing conformance items (such as `complete-runtime` and `complete-standard-library`). That reflects formal spec conformance status, not a broken installation—your local native builds are ready once the host C toolchain is installed.

## Common Setup Issues

- **Native build fails with exit code `6`**: The compiler could not find or run MSVC (`cl.exe` / Windows SDK) on Windows, or `gcc` / `clang` on `PATH` on Linux.
- **Wrong directory**: Package commands (`check`, `build`, `run`, `test`) expect a `fairy.toml` file in the current directory.
- **Next step**: Create and run your [First Program](../first-program/).
