---
title: Language Basics
description: Core Fairy C syntax, bindings, primitive and composite types, functions, control flow, and effect annotations.
---

Fairy C uses a C-shaped syntax with explicit types on declarations, block braces on control flow, checked integer arithmetic, and explicit side-effect annotations.

## Bindings: `let` and `var`

- `let` declares an immutable binding; `var` declares a mutable binding.
- Local variables must be initialized at declaration (`E1001` if omitted).
- Integer literals adopt the annotated scalar type when in range.

```c
let i32 limit = 10;
var i32 total = 0;
var u8[3] bytes = [1, 2, 3];
```

## Scalar, String, and Composite Types

- **Integers and booleans**: `i8`, `i16`, `i32`, `i64`, `u8`, `u16`, `u32`, `u64`, `isize`, `usize`, and `bool`.
- **Floating point**: `f32` and `f64` exist for C-layout interop (`@repr(C)`), not as general safe-code arithmetic types in Stage 0.
- **Bytes and strings**:
  - `b"..."` produces byte arrays/slices; `"..."` produces UTF-8 string values (`str`); `c"..."` produces NUL-terminated `CStr` literals for C interop.
  - Indexing or length checks work directly on arrays and slice views (`bytes.len`, `bytes[0]`).
- **Fixed arrays and aggregates**: `T[N]` (`[0; 512]` repeat initializer or `[1, 2, 3]` element list), nominal `struct`, and `enum`.
- **Tagged option/error types**: `Option<T>` (`.some(value)` / `.none`) and `Result<T, E>` (`.ok(value)` / `.err(error)`).

## Functions, Control Flow, and Checked Arithmetic

Functions declare their return type before the name, and non-`void` paths must explicitly `return`:

```c
struct State { i32 number; }

fn i32 State::read(ref State this) {
    return this.number;
}

fn i32 classify(i32 x) {
    return if (x > 0) { 1 } else { 0 };
}
```

- **Branches and loops**: `if` / `else`, `switch` over enums/tags/scalars, `while`, classic C-style `for (var usize i = 0; i < n; i += 1)`, and `for-in` over views.
- **`defer`**: Schedules cleanup expressions before scope exit.
- **Checked arithmetic**: Integer `+`, `-`, `*`, `/`, and shifts trap on overflow or division by zero (exit code `5` at runtime) instead of wrapping silently.
- **Explicit casts**: Use `value as TargetType`; there are no implicit numeric widening promotions.

## Handling `Result<T, E>` and `@effects(...)`

Fairy C treats `Result` obligations strictly: you must match (`switch`), propagate (`?`), return, or explicitly `discard` them (`E4101` otherwise):

```c
enum Error { Failed; }

fn Result<i32, Error> propagate(Result<i32, Error> input, Error fallback) {
    let i32 value = input?;
    return .err(fallback);
}

fn i32 ignore(Result<i32, Error> input) {
    discard input;
    return 0;
}
```

Functions that touch console I/O, files, environment variables, process args, network, clocks, or randomness declare `@effects(...)` from the Stage 0 capability set (`console`, `filesystem`, `environment`, `process`, `network`, `clock`, `random`):

```c
import std::io as io;

@effects(console)
fn bool print_number(u32 value) {
    return switch (io::write_u32(value)) {
        case .ok(_): { true }
        case .err(_): { false }
    };
}
```

If a function calls an effectful callee—or triggers compiler-scheduled cleanup on an effectful `resource`—without listing that effect, the compiler fails with `E7102`.

## Common Mistakes

- **Mixing `i32` and `usize` without `as`**: Write `index as i32` or annotate consistently.
- **Chained comparisons without parentheses**: Write `(a < b) && (b < c)`, not `a < b < c` (`E0009`).
- **Ignoring a `Result` or `@must_use` value**: Handle it or write `discard expr;`.
