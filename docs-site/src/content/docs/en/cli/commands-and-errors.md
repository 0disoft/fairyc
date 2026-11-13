---
title: CLI Usage and Common Errors
description: Stable fairyc workspace commands, flags, exit codes, and how to diagnose compiler and build failures.
---

The stable `fairyc` CLI operates on packages configured by `fairy.toml` and `fairy.lock`.

## Stable Commands

| Command | Purpose |
| --- | --- |
| `fairyc new <name> --uid <uid>` | Create a starter package directory (`fairy.toml`, `src/main.fy`, `tests/main.fy`) |
| `fairyc check [--locked]` | Parse, type-check, borrow-check, and update (or verify) `fairy.lock` |
| `fairyc build [--release] [--locked]` | Compile and link `build/<package-name>.exe` (Windows) or `build/<package-name>` (Linux) |
| `fairyc run [--release] [--locked] [-- <args...>]` | Build a temporary native binary, run it forwarding stdin/args, and remove the binary |
| `fairyc test [options] [--locked]` | Run tests in the semantic interpreter, or natively with `--native` |
| `fairyc explain <code>` | Print details for a diagnostic code such as `E2101` or `E7102` |
| `fairyc targets` | Show registered target specifications and current conformance flags |
| `fairyc --version` / `--help` | Print compiler identity and command help |

> **Stable vs. Experimental:** Any `fairyc experimental ...` subcommand (such as `emit-c` or `atlas-*`) is an unstable compiler-development tool rather than a supported user workflow.

## Useful Flags

- `--locked`: Require an existing, up-to-date `fairy.lock` rather than modifying it. Recommended for repeatable local checks and CI.
- `--release`: Enable optimized C backend compilation in `build`, `run`, and `test --native`.
- `--json`: Machine-readable output for `fairyc test` and `fairyc test --list`.
- `--native` (for `fairyc test`): Execute isolated native test binaries instead of the semantic interpreter. Accepts `--timeout-seconds <1..60>`, `--stdin-file <path>`, and `--capture-output` (with `--json`).

## Process Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Success (for `fairyc run`, only when the compiled program itself exits `0`) |
| `2` | Invalid CLI invocation or unsupported command option |
| `3` | Source syntax, type, borrow, effect, or manifest validation failure |
| `4` | Internal compiler failure |
| `5` | Fairy program runtime trap (for example, integer overflow, division by zero, or failed `assert`) |
| `6` | Native C toolchain, linker, or executable launch failure |
| `7` | Workspace, manifest (`fairy.toml`), or lockfile (`fairy.lock`) mismatch |

During `fairyc run`, any non-zero `i32` returned normally by your `main()` function is forwarded as the process exit status, unless the program hit a Fairy trap (exit code `5`).

## Looking Up and Fixing Common Errors

Use `fairyc explain <code>` when a diagnostic reports a Stage 0 error code:

| Code Family / Example | Meaning & Quick Fix |
| --- | --- |
| `E0001..E0011` | UTF-8, module path mismatch (`E0002`), wildcard/invalid import (`E0003`), or chained comparison (`E0009`). Check `module ...;` matches the file path and add parentheses around comparisons. |
| `E1001..E1014` | Type mismatch, missing initializer (`E1001`), or explicit cast needed (`E1006`). Use `as` for scalar conversions and match exact labels. |
| `E1101..E1109` | Move/cleanup violations (`resource` used after move `E1101`, `MaybeLive` join `E1104`). Borrow with `ref` instead of moving when keeping ownership. |
| `E2101..E2105` | Borrow conflicts (`E2101`) or multi-origin returned borrow (`E2103`). End conflicting borrows before mutating or return an owned value. |
| `E4101..E4202` | Unhandled `Result` (`E4101`) or ignored `@must_use` value (`E4202`). Handle with `switch`, `?`, or explicit `discard`. |
| `E5101..E5106` | Unsafe/FFI boundary errors. Raw pointer operations and `extern C` calls must appear inside `unsafe { ... }`. |
| `E7101..E7102` | Missing `@effects(...)` declaration (`E7102`). Add the callee's effect (such as `console` or `filesystem`) to the enclosing function. |
