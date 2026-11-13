---
title: Modules, Packages, and std
description: Learn Fairy C module and import syntax, pub visibility, fairy.toml workspaces, and the current std standard modules.
---

Once you split code across multiple files or bring in libraries, you will use Fairy C's module and package system. There are no header files (`#include`) or C preprocessor macros.

## 1. File Paths Match `module` Names

Every `.fy` file begins with one `module` declaration matching its path under the source root (`src/app/math.fy` -> `module app::math;`):

```c
module app::math;

pub fn i32 add(i32 a, i32 b) {
    return a + b;
}
```

Only items marked `pub` are visible to other modules.

## 2. Explicit Aliased Imports (`import ... as ...`)

Every import requires an alias (`as name`), keeping all external names clearly qualified:

```c
module main;

import app::math as math;
import std::io as io;

@effects(console)
fn i32 main() {
    let i32 v = math::add(70, 7);
    discard io::write_i32(v);
    return 0;
}
```

Wildcard imports (`*`) and circular module dependencies are disallowed.

## 3. `fairy.toml`, Workspaces, and `fairy.lock`

Define packages and direct dependencies in `fairy.toml`:

```toml
[package]
name = "my_app"
uid = "example.my-app"
version = "0.1.0"

[dependencies]
answer = { path = "../libs/answer" }
```

- A root `[workspace]` section lets you manage multiple packages together (`examples/workspace_quickstart` shows this layout).
- Running `fairyc check` or `fairyc test` records the resolved package graph in `fairy.lock`. Pass `--locked` when you want builds to fail if dependencies drift.
- Windows targets using extra linker settings pair with `fairy.windows.json`.

## 4. Available `std` Modules

When a package links the `std` SDK root (as seen in repository examples), the following modules are available:

| Module | Purpose | Example Functions / Types |
| --- | --- | --- |
| `std::io` | Console I/O (`@effects(console)`) | `write`, `write_i32`, `write_i64`, `write_bytes`, `write_stderr` |
| `std::vec` | Growable `Vec<T>` container | `empty<T>`, `push<T>`, `pop<T>`, `at<T>`, `truncate<T>`, `clear<T>` |
| `std::text` | UTF-8 strings & `CString` helpers | `validate_utf8`, `copy`, `append`, `as_bytes`, `cstring_from_text` |
| `std::bytes` | Byte slice and buffer helpers | Byte comparison and manipulation helpers |
| `std::fs` | Filesystem operations (`@effects(filesystem)`) | `open_read`, `create_truncate`, `read`, `write_all`, `seek_start`, `close` |
| `std::num` | Checked parsing & narrowing casts | `parse_i64_decimal`, `usize_from_i64`, `u8_from_i64` |
| `std::process` | Process arguments & exit handling | `process::arg(...)` |
| `std::alloc` | Explicit allocator interface | Explicit buffer/resource allocation primitives |

---

- Previous: [Ownership, Borrows, and Resources](../memory-and-resources/)
- Next: [C Interop and Win32](../../examples/c-interop-and-win32/)
