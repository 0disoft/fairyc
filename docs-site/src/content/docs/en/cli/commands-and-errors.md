---
title: CLI Usage and Common Errors
description: Everyday fairyc commands, exit codes, and quick fixes for common compiler errors.
---

Here is a practical cheat sheet for the stable `fairyc` CLI commands and the compiler errors you are most likely to run into first.

## Everyday Commands

| Command | When to Use It |
| --- | --- |
| `fairyc new <name> --uid <uid>` | Create a new package folder |
| `fairyc check [--locked]` | Fast syntax, type, and borrow check without compiling C |
| `fairyc build [--release] [--locked]` | Produce `build/<name>.exe` (Windows) or `build/<name>` (Linux) |
| `fairyc run [--release] [--locked] [-- <args...>]` | Compile and run a temporary binary right away |
| `fairyc test [--native] [--locked]` | Run `tests/` in the interpreter or as native binaries (`--native`) |
| `fairyc explain <code>` | Read details on a diagnostic code like `E2101` |

Stick to these stable commands—`fairyc experimental ...` subcommands are internal tools for compiler development.

## Exit Codes at a Glance

- **`0`**: Success (`run` also forwards your `main()`'s non-zero return value if it doesn't trap)
- **`2`**: Wrong CLI flags or invocation syntax
- **`3`**: Compile error in your `.fy` source or manifest
- **`5`**: Runtime trap (integer overflow, division by zero, failed `assert`)
- **`6`**: Host C compiler / linker not found or failed
- **`7`**: `fairy.toml` / `fairy.lock` mismatch (re-run `fairyc check`)

## Common Errors and How to Fix Them

Run `fairyc explain <CODE>` for any code you see. Most beginner errors fall into one of these:

1. **`E1001` (uninitialized variable)**: Write `var i32 count = 0;` instead of `var i32 count;`.
2. **`E1006` / `E0009` (mixed integer types or chained comparisons)**: Cast explicitly with `as`, and write `(0 < x) && (x < 10)` instead of `0 < x < 10`.
3. **`E1101` (use after move)**: You passed a `resource` by value and tried to use it again. Borrow with `ref` or `ref mut` if the caller still needs it.
4. **`E2101` (borrow conflict)**: You mutated or moved a variable while a `ref` borrow was still used later in the block. Finish using the borrow before mutating.
5. **`E4101` (ignored `Result`)**: Handle `Result` returns with `switch`, `?`, or explicit `discard`.
6. **`E7102` (missing `@effects(...)`)**: Add the callee's effect (like `console` or `filesystem`) to your function's `@effects(...)` attribute.
