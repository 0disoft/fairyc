---
title: Functions and Control Flow
description: Learn function declarations, argument labels, scalar generics, if-expressions, while, C-style for, for-in iteration, and defer in Fairy C.
---

Now that variables and operators are clear, let's look at functions and control flow. Fairy C keeps a C-like rhythm while adding conveniences like value-producing `if`, safe `for-in` loops, and `defer`.

## 1. Functions and Argument Labels

Functions follow `fn ReturnType name(Parameters)`. Use `void` when no value is returned.

```c
fn i32 clamp(i32 value, i32 minimum, i32 maximum) {
    if (value < minimum) { return minimum; }
    if (value > maximum) { return maximum; }
    return value;
}

fn void log_step() {
    // void functions may omit an explicit return;
}
```

Unlike Rust, **non-`void` function bodies do not implicitly return their final line without `return`**. Always write `return value;`.

### Position-Preserving Argument Labels

When calling a non-generic module-local function, you can clarify arguments using `.parameter_name = value`:

```c
let i32 result = clamp(77, .minimum = 0, .maximum = 100);
```

Labels cannot reorder arguments or supply defaults—each label must match that parameter's exact position—so they serve strictly as readability guards.

### `@inline` Function Hint

Non-generic helper functions can carry `@inline` (at most once, with no arguments) to permit cross-module or local inline expansion; calls inside an `@inline` body must use bare same-module names without `::`.

```c
@inline
pub fn i32 combine(i32 a, i32 b) {
    return a + b;
}
```

## 2. Generic Scalar Functions (`<T>`)

Ordinary functions can take explicit scalar type parameters `<T>` and be called as `fn_name<i32>(...)`:

```c
fn T pick_first<T>(T first, T second) {
    return first;
}

fn i32 main() {
    return pick_first<i32>(77, 0);
}
```

Type arguments currently cover core scalars (`bool`, `u8`, `i32`, `u32`, `usize`) and are specialized at compile time. (Type inference at call sites and user-defined generic structs/enums are not part of Stage 0.)

### Function Values and Narrow Closures (`fn i32(i32)`)

Stage 0 supports narrow first-class function values and value-capturing closures of shape `fn i32(i32)`.

```c
fn i32 apply_once(fn i32(i32) f, i32 x) {
    return f(x);
}

fn i32 make_result() {
    i32 offset = 7;
    fn i32(i32) add_offset = fn [i32 offset] i32(i32 x) {
        return x + offset;
    };
    return apply_once(add_offset, 70);
}
```

- Capture lists (`[i32 offset]`) capture up to 4 immutable `i32` values by copy. Closure values can be passed, returned, and stored in plain `struct` fields.
- Closure bodies stay pure (no `var`, no effects, and no external calls), and closure types cannot be placed inside `Option`, `Result`, `ref`, `ptr`, arrays, `@repr(C)` structs, `resource`, or `extern C` signatures.

## 3. `if / else` as Expressions

Conditions always require parentheses `(...)`. If each branch ends with an expression without a semicolon, `if` evaluates directly to a value:

```c
let i32 score = 85;
let i32 grade = if (score >= 90) {
    1
} else if (score >= 80) {
    2
} else {
    3
};
```

Both branches must produce the exact same type. Likewise, standalone `{ let i32 a = 70; a + 7 }` blocks act as expressions when they end with an expression without `;`.

## 4. `while` and Two Forms of `for`

### `while` and Classic `for`

```c
var i32 n = 0;
while (n < 5) {
    n += 1;
}

var i32 sum = 0;
for (var i32 i = 0; i < 10; i += 1) {
    if (i == 3) { continue; }
    if (i == 8) { break; }
    sum += i;
}
```

Any header clause in `for (init; cond; post)` can be omitted (an empty condition means `true`). On `continue`, the `post` clause runs before the next condition check.

### `for-in` Over Arrays and Slice Views

To avoid off-by-one index mistakes when walking an array or view, use `for-in`:

```c
var i32[4] items = [10, 20, 30, 40];

// Copy each element to read
var i32 total = 0;
for (let i32 item in items) {
    total += item;
}

// Borrow each element mutably in place
for (let ref mut i32 item in items) {
    *item += 1;
}
```

You can also iterate over sub-ranges directly, such as `for (let u8 b in data[1..4]) { ... }`, as well as read-only SDK `Vec<T>` values with `for (let ref i32 item in vec) { ... }`.

## 5. `assert`, `defer`, and `source_location()`

- **`assert(cond);`**: Traps immediately if `cond` is false.
- **`defer call(...);`**: Schedules an internal `void` function call (with scalar arguments evaluated at registration time) to run right before leaving the current block—prior to implicit resource drops or region resets.
- **`source_location()`**: Built-in zero-argument intrinsic returning `"path:line:col@offset"` as `ref str`, handy for diagnostics and logging helpers.

---

- Previous: [Variables, Types, and Operators](../basics/)
- Next: [Structs, Enums, and Switch](../structs-enums-and-errors/)
