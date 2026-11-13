---
title: First Program
description: Create a new Fairy C package, check its types and borrows, run tests, and build a native binary.
---

This walkthrough uses `fairyc new` to create a starter package, inspect the generated files, run static checks and unit tests, and execute the compiled native binary.

## 1. Create a Package

Choose a directory outside the compiler repository (or a clean workspace folder) and run `fairyc new` with a required `--uid`:

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

This creates five files:

- `fairy.toml`: package manifest
- `src/main.fy`: application entry source
- `tests/main.fy`: starter unit test module
- `.gitignore` and `README.md`

Inspect `fairy.toml`:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

Inspect `src/main.fy`:

```c
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

Every executable Fairy C package starts in `module main;` with `fn i32 main()`. The returned `i32` becomes the process exit code.

## 2. Run Static Check

Validate syntax, types, ownership, and borrows, and generate `fairy.lock`:

```powershell
fairyc check
```

Expected output:

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

Once `fairy.lock` exists, pass `--locked` to ensure builds fail if manifest or source dependencies drift unexpectedly:

```powershell
fairyc check --locked
```

## 3. Run the Starter Test

Execute tests in `tests/` using the semantic interpreter:

```powershell
fairyc test --locked
```

Expected output:

```text
test tests::main::answer_is_42 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

You can also compile and run tests as isolated native binaries with `fairyc test --native --locked`.

## 4. Run and Build Natively

Run the program directly as a temporary native binary:

```powershell
fairyc run --locked
```

Because `answer() - 42` evaluates to `0`, `fairyc run --locked` finishes with exit status `0` and produces no leftover artifact in the folder.

To keep the compiled executable under `build/hello-fairy.exe` (Windows) or `build/hello-fairy` (Linux):

```powershell
fairyc build --release --locked
```

## Common Beginner Mistakes

- **Omitting `--uid` in `fairyc new`**: `--uid` is mandatory and must contain at least two lowercase dot-separated kebab segments (for example, `example.hello-fairy`).
- **Capital letters in package `name` or `uid`**: Manifest validation rejects uppercase letters. Keep both lowercase kebab-case.
- **Running `check --locked` right after editing dependencies**: Update `fairy.lock` first with `fairyc check` before locking CI/local runs again.
- **Exit code `6` during `run` or `build`**: Verify MSVC + Windows SDK (Windows) or `gcc`/`clang` (Linux) is accessible in your shell environment.
