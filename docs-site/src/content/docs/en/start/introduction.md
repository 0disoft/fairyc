---
title: Introduction
description: Why Fairy C exists, what syntax ideas it uses, and how it compares to Brownie C.
---

Writing C gives you direct control over memory and OS APIs, but it is easy to forget a `free()`, touch freed data, or let an integer silently wrap around on overflow.

Fairy C is an experimental systems programming language that keeps C-like native execution without garbage collection while letting the compiler catch ownership, borrowing, and arithmetic mistakes early.

## Key Ideas in the Language

- **Resources know how to clean themselves up**: Types holding handles or buffers can be declared as `resource` with a `cleanup { ... }` block. When a live resource leaves scope, the compiler schedules cleanup automatically.
- **Move vs. borrow (`ref` and `ref mut`)**: You either transfer ownership of a resource or lend read-only (`ref T`) / exclusive mutable (`ref mut T`) access while the compiler checks for overlapping conflicts.
- **Checked integer math**: Integer overflow and division by zero trap cleanly instead of wrapping silently.
- **Explicit `@effects(...)` tags**: Functions that touch the console, filesystem, clock, or network list those effects in their signature.
- **Direct C interop**: `@repr(C)` structs and `unsafe extern C fn` let Fairy C call existing C libraries and Win32 APIs directly.

## Where It Fits Today

Fairy C is well suited for small native CLI tools, C ABI wrappers, and focused Win32 utilities.

Because the compiler is a **Stage 0 prototype**, stick to the stable workspace commands (`new`, `check`, `build`, `run`, `test`, `explain`, `targets`) and treat `fairyc experimental ...` subcommands as internal compiler-development tools. Formal target conformance (`fairyc targets` currently shows `supported: no`) and a full standard library are still in progress.

## How Does It Compare to C++ and Rust?

If you are coming from another systems language, here is the quick mental model:

- **Compared to C++**: Fairy C drops `#include` headers, preprocessor macros, exceptions, class inheritance, and operator overloading. It uses explicit `module`/`import` declarations, `Result` + `?` error handling, explicit `resource cleanup` blocks, compile-time `ref`/`ref mut` borrow checks, and checked integer/bounds arithmetic instead of silent wraparound or undefined behavior.
- **Compared to Rust**: Fairy C aims for a much smaller, C-shaped surface compiled through a host C11 toolchain. Current Stage 0 has no traits, macros, `async`, or explicit `'a` lifetime annotations in user code (only narrow value-capturing `fn i32(i32)` closures are supported). Instead of a general `Drop` trait on arbitrary structs, Fairy C separates plain copyable `struct` values from move-only `resource` types with explicit `cleanup { ... }` blocks, and tracks I/O capabilities with `@effects(...)`.
- **Maturity**: Unlike C++ and Rust, which are mature production ecosystems, Fairy C is still an experimental **Stage 0 prototype** with a small standard library and narrow target scope.

## Fairy C vs. Brownie C

| Aspect | Fairy C (`.fy`) | Brownie C (`.brc`) |
| --- | --- | --- |
| Purpose | Standalone native programs and libraries | Embedded rule and byte checks inside a host |
| Execution | Compiled via C11 to native executables | Run inside a bounded bytecode VM |
| OS & C access | Direct `unsafe extern C fn` and stdlib I/O | No arbitrary C calls, files, or network access |
| Budgeting | Normal native OS execution | Bounded by an instruction `--fuel` budget |
