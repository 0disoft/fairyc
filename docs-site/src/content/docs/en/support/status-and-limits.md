---
title: Current Support and Limitations
description: What works in Stage 0 Fairy C today, what is still incomplete, and what fairyc targets means.
---

Fairy C is an active **Stage 0 compiler prototype**. Knowing what works today versus what is still in progress helps set realistic expectations.

## What Works Today

- Packages and local `[workspace]` path dependencies (`fairy.toml` + `fairy.lock`)
- `resource` ownership with `cleanup`, `ref`/`ref mut` borrows, slice views (`u8[]`), and single-origin borrow returns
- Checked integer arithmetic, `Option<T>`, `Result<T, E>`, `?` propagation, and `@effects(...)`
- Direct C ABI interop (`@repr(C)`, `unsafe extern C fn`, synchronous C callbacks, Win32 bridge examples)
- Core Stage 0 SDK modules (`std::io`, `std::fs`, `std::bytes`, `std::alloc`, `std::text`, `std::vec`, `std::process`)

With MSVC + Windows SDK on Windows or GCC/Clang on Linux, you can build and run console tools and Win32 examples within this subset. Note that `fairyc targets` currently prints `supported: no` for both `x86_64-windows-msvc` and `x86_64-linux-gnu` because formal target-spec conformance and full stdlib coverage are not finished yet.

## Current Limitations

- **Not a finished universal toolchain**: Full platform coverage, arbitrary C++ compatibility, and general performance guarantees are not claimed.
- **Partial standard library**: Async runtimes, HTTP clients/servers, and cross-platform GUI toolkits are not included.
- **Local dependencies only**: There is no remote package registry or automatic downloader.
- **Language restrictions**: Generics are limited to constrained scalar uses, borrow-returning callbacks/closures are restricted (`E2104`, `E5104`), and `f32`/`f64` are primarily for `@repr(C)` C layouts.
