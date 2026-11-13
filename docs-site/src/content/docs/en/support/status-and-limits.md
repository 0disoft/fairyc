---
title: Current Support and Limitations
description: Exact Stage 0 prototype scope, target-spec status from fairyc targets, and what is not supported yet.
---

Fairy C is an active **Stage 0 compiler prototype** (`fairyc-language/20`, `fairyc-sdk-stdlib/0 0.0.0-stage0.87`). Knowing what works practically today versus what remains unverified keeps expectations realistic.

## Target Status (`fairyc targets`)

Running `fairyc targets` lists two registered targets, both marked `supported: no` at the formal target-specification level:

- `x86_64-windows-msvc`: `emittable: no`, `conformant: no`, `supported: no`
- `x86_64-linux-gnu`: `emittable: no`, `conformant: no`, `supported: no`

In practice, this means:

- **What works**: On a host with a configured C toolchain (MSVC + Windows SDK on Windows, or GCC/Clang on Linux), `fairyc check`, `fairyc build`, `fairyc run`, and `fairyc test` compile and run real packages, Win32/C-interop apps, and standard-library examples within the implemented Stage 0 subset.
- **What `supported: no` reflects**: Formal bidirectional C ABI layout conformance suites, complete Windows x64 unwind fixtures, full standard library coverage, complete runtime ABI coverage, and packaged toolchain provisioning are not finished yet.

## Supported Features Today

- Single-package and local path-based `[workspace]` projects defined by `fairy.toml` + `fairy.lock`.
- Move-only `resource` types with deterministic `cleanup`, whole-root borrow checking (`ref` / `ref mut` / slice views), and single-origin borrow returns.
- Checked integer arithmetic (`+`, `-`, `*`, `/`, shifts) and explicit `as` casts.
- `Option<T>`, `Result<T, E>` (with `?` propagation and unused-`Result` checking), `@must_use`, `@const_eval`, `static_assert`, `defer`, and `@effects(...)`.
- Direct C ABI interoperability (`@repr(C)`, `@link_name`, `unsafe extern C fn`, synchronous C callbacks).
- Stage 0 standard modules used across examples (`std::io`, `std::fs`, `std::bytes`, `std::alloc`, `std::text`, `std::vec`, `std::process`).

## Known Limitations and Exclusions

- **No finished universal compatibility claim**: Do not assume all target platforms, C compilers, or arbitrary C++ libraries work out of the box.
- **Partial standard library**: Only the existing `std::*` modules implemented in the Stage 0 SDK are available; rich networking, async executors, or general cross-platform GUI frameworks are not part of the standard library.
- **No remote package registry or automatic downloads**: `fairy.toml` resolves local workspace members and explicit dependencies; remote registry fetching is not part of stable `fairyc`.
- **Stage 0 generic & callback limits**: Generics follow strict explicit scalar constraints (`E45xx`), safe closures/higher-order borrow-returning callbacks are restricted (`E2104`, `E5104`), and `f32`/`f64` stay limited to C ABI representation structs.
- **Experimental subcommands are unstable**: Commands under `fairyc experimental` exist for internal compiler and tooling workflows and may change or be removed without notice.
