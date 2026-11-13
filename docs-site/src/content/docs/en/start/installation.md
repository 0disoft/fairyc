---
title: Installation and Setup
description: Build the Stage 0 fairyc compiler from source and make sure your C toolchain is ready.
---

There is no standalone installer or prebuilt binary download yet. You build `fairyc` from the repository source and pair it with your platform's C compiler.

## Prerequisites

1. **Git, Go, and Bun**: Needed to compile the Stage 0 Go compiler (`tools/run-go.ts` wraps Go through Bun).
2. **A native C compiler** (needed for `fairyc build`, `fairyc run`, and `fairyc test --native`):
   - **Windows**: Visual Studio / Build Tools with the **MSVC x64 C/C++ compiler** and **Windows SDK**.
   - **Linux**: `gcc` or `clang` available on `PATH`.

> **Tip:** `fairyc check` and interpreter-mode `fairyc test` do not invoke a C compiler, so you can type-check code even before configuring MSVC or GCC.

## Building `fairyc`

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
