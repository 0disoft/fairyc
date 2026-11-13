---
title: Projects, Modules, and Dependencies
description: Managing fairy.toml manifests, fairy.lock, module visibility, local workspace dependencies, and package tests.
---

Every Fairy C package centers on a `fairy.toml` manifest at the package root, optional workspace path dependencies, and a `fairy.lock` lockfile.

## The `fairy.toml` Manifest (`fairyc.manifest/v1`)

`fairy.toml` uses a strict closed schema—unknown fields or mismatched keys are rejected rather than ignored:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

- **`uid`**: Permanent dot-separated lowercase kebab identifier (at least two segments).
- **`name`**: Lowercase kebab CLI/package output name (`build/<name>.exe` on Windows).
- **`version`**: Exact SemVer `MAJOR.MINOR.PATCH` string (ranges are rejected).
- **`language`**: Exact language revision identifier (`fairyc-language/20` in current Stage 0).
- **`source`**: Explicit relative source directory (such as `"src"`).

## Modules, Imports, and Visibility

- **File-to-module mapping**: Each `.fy` file begins with `module <name>;` matching its relative path inside `source` (`E0002` if mismatched).
- **Explicit aliased imports**: Wildcard and glob imports are forbidden (`E0003`). Import packages or modules with an explicit alias:

```c
module main;

import answer as answer;

fn i32 main() {
    return answer::value();
}
```

- **Visibility**: Declarations are module-private unless marked `pub` (`pub fn`, `pub struct`, `pub resource`). Accessing non-public items from another module fails with `E0103`.

## Local Workspaces and Path Dependencies

Multi-package projects declare workspace members and explicit path dependencies in the root `fairy.toml` (`examples/workspace_quickstart`):

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.workspace-quickstart.app"
name = "workspace-quickstart"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"

[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
```

Workspace rules:

- `workspace.members` must be explicit, sorted, and unique; there is no parent-directory discovery.
- Every `[dependencies.<alias>]` entry specifies `path`, exact expected `version`, and `features = [...]` (`features = []` is required when empty).
- Stable `fairyc` resolves local workspace dependencies only—manifest v1 intentionally excludes C preprocessor `defines`, `include_paths`, `cflags`, and automatic remote registry downloads.

## `fairy.lock` and `--locked`

Running `fairyc check`, `fairyc build`, or `fairyc run` without `--locked` creates or refreshes `fairy.lock`. Once locked, passing `--locked` (including in `fairyc test --locked`) fails with exit code `7` if any manifest or local dependency changed without re-checking.

## Organizing Tests in `tests/`

Put test functions in `.fy` files under `tests/` using `@test` on private safe zero-argument `fn void` functions (`E9001`):

```c
module tests::main;

import answer as answer;

@test
fn void answer_is_42() {
    assert(answer::value() == 42);
}
```

Run tests with:

- `fairyc test --locked` (semantic interpreter)
- `fairyc test --native --locked` (isolated compiled native test binaries)
- `fairyc test --list --json --locked` (inspect test identities)
