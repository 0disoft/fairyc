---
title: Language Basics
description: Learn Fairy C bindings, types, functions, control flow, Option/Result handling, and @effects tags step by step.
---

If you have written any C, Java, or TypeScript, Fairy C syntax will feel familiar right away. The main difference is that Fairy C refuses a few common shortcuts—like uninitialized variables, implicit integer mixing, or silently ignored errors—so bugs show up at compile time instead.

## 1. Bindings: `let` and `var`

Use `let` for values that do not change and `var` for mutable ones. Every local variable must be initialized when declared:

```c
let i32 limit = 10;
var i32 total = 0;
var u8[3] bytes = [1, 2, 3];
```

## 2. Types and Explicit `as` Casts

- **Integers**: Signed `i8`, `i16`, `i32`, `i64`, `isize` and unsigned `u8`, `u16`, `u32`, `u64`, `usize`.
- **Booleans**: `bool` (`true`, `false`).
- **Text and bytes**: `"..."` for UTF-8 strings (`str`), `b"..."` for byte arrays/slices, and `c"..."` for NUL-terminated C strings (`CStr`).
- **Arrays, structs, enums**: `T[N]` (including repeat syntax `[0; 10]`), nominal `struct`, and `enum`.

Two things catch newcomers coming from C:

1. **No implicit integer promotions**: Adding a `u8` to an `i32` requires an explicit `as` cast:

```c
let u8 byte_val = 40;
let i32 total = (byte_val as i32) + 2;
```

2. **Overflow traps instead of wrapping**: If integer math overflows or divides by zero, the program traps immediately (exit code `5`) rather than wrapping into a bogus number.

## 3. Functions and Control Flow

Functions place the return type before the name, and structs can define methods with `Type::method`:

```c
struct State { i32 number; }

fn i32 State::read(ref State this) {
    return this.number;
}

fn i32 classify(i32 x) {
    return if (x > 0) { 1 } else { 0 };
}
```

- **Loops & `defer`**: `while`, C-style `for`, `for-in` over views, and `defer` to run cleanup statements before leaving a block.
- **No chained comparisons**: Write `(0 < x) && (x < 10)` rather than `0 < x < 10` (`E0009`).

## 4. `Option<T>` and `Result<T, E>`

Instead of null pointers, Fairy C uses `Option<T>` (`.some(v)` / `.none`) for optional values and `Result<T, E>` (`.ok(v)` / `.err(e)`) for operations that can fail.

You cannot accidentally forget to check a `Result` (`E4101`). Match it with `switch`, propagate it upward with `?`, or intentionally opt out with `discard`:

```c
enum Error { Failed; }

fn Result<i32, Error> add_positive(Result<i32, Error> input) {
    let i32 value = input?; // Returns .err(...) immediately if input failed
    return .ok(value + 1);
}

fn i32 ignore_result(Result<i32, Error> input) {
    discard input;
    return 0;
}
```

## 5. `@effects(...)` Tags

Functions that perform console I/O, touch files, read clocks, or inspect process arguments declare those side effects with `@effects(...)` (`console`, `filesystem`, `environment`, `process`, `network`, `clock`, `random`):

```c
import std::io as io;

@effects(console)
fn bool print_u32(u32 value) {
    return switch (io::write_u32(value)) {
        case .ok(_): { true }
        case .err(_): { false }
    };
}
```

Calling an effectful function from a function that forgot to list that effect triggers `E7102`—making side effects obvious directly from function signatures.
