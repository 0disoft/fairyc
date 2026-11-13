---
title: Fairy C
description: Experimental native systems programming language with explicit ownership, borrow checking, and C interoperability.
---

Fairy C is an experimental native systems programming language with explicit resource ownership, compile-time borrow checking, checked integer arithmetic, capability annotations (`@effects(...)`), and direct C ABI interoperability without garbage collection.

The current compiler (`fairyc`) is a Stage 0 prototype written in Go (`fairyc-language/20`). It compiles `.fy` packages to native executables through a C11 backend using your platform's C toolchain.

## Explore the Documentation

- **Getting Started**: [Introduction](./start/introduction/) · [Installation and Setup](./start/installation/) · [First Program](./start/first-program/)
- **Learn the Language**: [Language Basics](./language/basics/) · [Memory and Resources](./language/memory-and-resources/) · [Projects, Modules, and Dependencies](./language/modules-and-projects/)
- **Examples**: [C Interop and Win32 Apps](./examples/c-interop-and-win32/) (`win32_minimal`, `win32_hash`)
- **Support Status**: [Current Support and Limitations](./support/status-and-limits/) · [CLI Usage and Common Errors](./cli/commands-and-errors/)

## How Fairy C Relates to Brownie C

Fairy C is a general-purpose native systems language built for executables and libraries that interact with C and OS APIs.

**Brownie C** is a separate embedded rule and byte-transformation language with a bounded virtual machine, and it is being implemented *in* Fairy C. If you want to build native programs, you are in the right place.
