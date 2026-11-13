---
title: Introduction
description: What Fairy C is for, how it compares to Brownie C, and what works in Stage 0 today.
---

Fairy C is an experimental systems programming language designed to give C-like native control while checking resource lifetimes, borrows, arithmetic overflow, and side-effect boundaries before runtime.

## What Fairy C Focuses On

- **Explicit resource ownership**: Move-only `resource` types attach deterministic `cleanup` blocks instead of relying on garbage collection.
- **Compile-time borrow checking**: Shared (`ref T`) and exclusive (`ref mut T`) references and slice views (`u8[]`) verify root lifetimes across scopes and calls.
- **Checked integer operations**: Signed and unsigned integer overflow and division by zero trap instead of silently wrapping.
- **Declared side-effect capabilities**: Functions interacting with external systems declare `@effects(...)` tags such as `console`, `filesystem`, `clock`, or `network`.
- **Direct C interoperability**: `@repr(C)` structs, `unsafe extern C fn` declarations, and explicit `unsafe` blocks bridge directly to C libraries and OS APIs such as Win32.

## Good Fit vs. Not a Fit Yet

Fairy C fits well when you want to write small native CLI tools, low-overhead system bridges, or Win32/C-ABI wrappers with checked resource cleanup and static type/flow checks.

Because the compiler is still a **Stage 0 prototype**, it is not yet a drop-in replacement for a mature toolchain with full target conformance, user-defined generics, closures, or a public package registry.

## Fairy C vs. Brownie C

| Aspect | Fairy C (`.fy`) | Brownie C (`.brc`) |
| --- | --- | --- |
| Role | Native systems programming language | Embedded rule and byte-transformation language |
| Execution model | Compiles via C11 to native executables | Compiles to private bytecode inside a bounded VM |
| Compiler host | Go Stage 0 compiler prototype (`fairyc`) | Implemented in Fairy C (`browniec`) |
| OS & C access | Direct `unsafe extern C fn` and stdlib I/O | No arbitrary FFI or direct OS file/network access |
| Best suited for | CLI utilities, Win32 apps, native engines | Deterministic rule evaluation under tight budgets |

## Design Goals, Public CLI, and Internal Commands

When exploring the repository, keep these three layers distinct:

1. **Public stable CLI (`fairyc`)**: `new`, `check`, `build`, `run`, `test`, `explain`, and `targets`. These commands work on packages defined by `fairy.toml` and `fairy.lock`.
2. **Internal / unstable commands (`fairyc experimental ...`)**: Subcommands such as `emit-c`, `run` (semantic interpreter), and `atlas-*` code-intelligence inspection commands exist for compiler development and tooling experiments, not as a stable user interface.
3. **Target conformance goals**: `fairyc targets` lists `x86_64-windows-msvc` and `x86_64-linux-gnu` as **target prototypes** (`supported: no`), meaning formal target-spec conformance and full standard-library coverage remain incomplete even though local native builds succeed on configured toolchains.
