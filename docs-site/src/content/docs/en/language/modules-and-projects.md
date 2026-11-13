---
title: Projects, Modules, and Dependencies
description: Organize multi-file packages with module declarations, pub visibility, fairy.toml workspaces, and tests/.
---

Once your program grows beyond a single `src/main.fy`, Fairy C splits code cleanly using `module` paths, `pub` visibility, and local workspace packages in `fairy.toml`.

## 1. Modules, Imports, and Visibility

Every `.fy` file begins with a `module` header matching its relative path under `source` (`E0002` otherwise):

- `src/main.fy` → `module main;`
- `src/net/http.fy` → `module net::http;`

When importing another module, always give it an explicit alias (`import ... as ...;`). Wildcard imports (`*`) are disallowed (`E0003`) so you can always trace where a symbol came from:

```c
module main;

import answer as answer;

fn i32 main() {
    return answer::value();
}
```

Everything inside a module is **private by default**. Mark functions, structs, or resources with `pub` when another module needs to access them (`E0103` otherwise).

## 2. Inside `fairy.toml`

`fairy.toml` uses a strict schema so typos fail instead of being silently ignored:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

- `uid`: Dot-separated lowercase identifier (`example.hello-fairy`)
- `name`: Lowercase package name used for `build/<name>.exe`
- `version`: Exact `MAJOR.MINOR.PATCH` version
- `language`: Language revision (`"fairyc-language/20"`)
- `source`: Source root directory (`"src"`)

## 3. Local Workspace Dependencies

To depend on a local library package (such as `libs/answer` in `examples/workspace_quickstart`), list it under `[workspace]` and `[dependencies.<alias>]`:

```toml
[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
```

Stable `fairyc` supports local path dependencies only—there is no automatic remote registry fetch or C compiler flags (`cflags` / `defines`) in `fairy.toml`.

## 4. `fairy.lock` and `--locked`

Running `fairyc check`, `build`, or `run` creates or updates `fairy.lock`. Adding `--locked` tells the compiler to fail (exit code `7`) if manifests or dependencies differ from the lockfile.

## 5. Writing Tests in `tests/`

Place unit tests in `tests/` using `@test` on zero-argument `fn void` functions:

```c
module tests::main;

import answer as answer;

@test
fn void answer_is_42() {
    assert(answer::value() == 42);
}
```

Run them via `fairyc test --locked` (semantic interpreter) or `fairyc test --native --locked` (native compiled test binaries).
