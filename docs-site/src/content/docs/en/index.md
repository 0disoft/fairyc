---
title: Fairy C
description: An experimental native systems programming language with explicit resource cleanup, borrow checking, and C interoperability.
---

Fairy C is an experimental systems programming language designed to build small native binaries like C while catching resource leaks, dangling borrows, and integer overflows at compile time.

The current `fairyc` compiler is a Stage 0 prototype written in Go. It compiles `.fy` packages through a C11 backend using your system C toolchain (MSVC + Windows SDK on Windows, or GCC/Clang on Linux).

## Where to Start

- **Getting Started**: [Introduction](./start/introduction/) · [Installation and Setup](./start/installation/) · [First Program](./start/first-program/)
- **Learn the Language**: [Variables, Types, and Operators](./language/basics/) · [Functions and Control Flow](./language/control-flow-and-functions/) · [Structs, Enums, and Switch](./language/structs-enums-and-errors/) · [Arrays, Slices, and Views](./language/arrays-and-slices/) · [Ownership, Borrows, and Resources](./language/memory-and-resources/) · [Modules, Packages, and std](./language/modules-and-projects/)
- **Examples**: [C Interop and Win32 Apps](./examples/c-interop-and-win32/)
- **CLI & Status**: [CLI Usage and Common Errors](./cli/commands-and-errors/) · [Current Support and Limitations](./support/status-and-limits/)

## Fairy C vs. Brownie C

While **Fairy C** builds standalone native executables and C/OS libraries, **Brownie C** is a separate embedded rule and byte-transformation language running inside a bounded VM. Brownie C's own compiler and VM are written in Fairy C.
