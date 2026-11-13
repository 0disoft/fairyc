---
title: First Program
description: Create your first Fairy C package, read through the generated code, and run checks, tests, and native builds.
---

Let's create a starter Fairy C package, read the generated files line by line, and run it.

## 1. Create a New Package

Pick an empty directory outside the compiler repository and run `fairyc new` with a unique `--uid`:

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

This generates five files: `fairy.toml`, `src/main.fy`, `tests/main.fy`, `.gitignore`, and `README.md`.

Look at `fairy.toml`:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

And look at `src/main.fy`:

```c
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

A quick breakdown:

- `module main;` declares this file's module name. The entry file `src/main.fy` must be `module main;`.
- `pub fn i32 answer()` puts the return type (`i32`) before the function name, and `pub` lets other modules (such as `tests/main.fy`) call it.
- `fn i32 main()` is the entry point. Whatever `i32` it returns becomes the process exit code (`42 - 42` returns `0`).

## 2. Check Syntax, Types, and Borrows

```powershell
fairyc check
```

Expected output:

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

The first `check` creates `fairy.lock`. After that, you can pass `--locked` so future runs verify the manifest hasn't drifted unexpectedly:

```powershell
fairyc check --locked
```

## 3. Run the Starter Test

`tests/main.fy` already includes a starter test. To add more tests later, mark `fn void` functions with `@test`:

```c
module tests::main;

import main as app;

@test
fn void answer_is_42() {
    assert(app::answer() == 42);
}
```

Run it with:

```powershell
fairyc test --locked
```

Expected output:

```text
test tests::main::answer_is_42 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

Pass `--native` (`fairyc test --native --locked`) if you want tests compiled into native binaries instead of run in the interpreter.

## 4. Run and Build Natively

Compile and run a temporary native binary:

```powershell
fairyc run --locked
```

Since `main()` returns `0`, the process finishes cleanly with exit status `0`. To keep the compiled executable at `build/hello-fairy.exe` (Windows) or `build/hello-fairy` (Linux):

```powershell
fairyc build --release --locked
```

## Common Beginner Gotchas

- **Forgetting `--uid` in `fairyc new`**: `--uid` is required and needs at least two lowercase dot-separated kebab segments (like `example.hello-fairy`).
- **Uppercase letters in `name` or `uid`**: Manifest validation allows lowercase kebab-case only.
- **Running `--locked` right after editing `fairy.toml`**: Run `fairyc check` once without `--locked` to refresh `fairy.lock`.
- **Exit code `6` on `build` or `run`**: Your host C compiler (MSVC + Windows SDK on Windows, or `gcc`/`clang` on Linux) was not found or failed to link.
