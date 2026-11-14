---
title: Structs, Enums, and Switch
description: Learn Fairy C structs, field shorthand, Type::method syntax, enums, exhaustive switch matching, and Option/Result/? error propagation.
---

Once programs grow past a few integers, you will want to group fields together and model multiple outcomes cleanly. Fairy C combines `struct`, `enum`, exhaustive `switch`, and `Option` / `Result` for this.

## 1. Structs and Field Shorthand

Define a plain record with `struct`. Add `pub` to fields you want accessible outside the defining module:

```c
pub struct Point {
    pub i32 x;
    pub i32 y;
}

fn i32 demo() {
    let i32 x = 10;
    let i32 y = 32;

    // .y is shorthand for .y = y
    let Point p = Point { .x = x, .y };
    return p.x + p.y;
}
```

Whenever a local variable has the same name as the field, `.field` works as shorthand. All fields must be supplied in declaration order.

### Methods with `Type::method`

Associate functions with a `struct` or `resource` by prefixing the name with `Type::`. When the first parameter is named `this` (`ref T this`, `ref mut T this`, or `T this`), you can call it with dot syntax:

```c
struct Counter {
    i32 value;
}

fn i32 Counter::read(ref Counter this) {
    return this.value;
}

fn void Counter::add(ref mut Counter this, i32 delta) {
    this.value += delta;
}

fn i32 main() {
    var Counter c = Counter { .value = 75 };
    c.add(2);
    return c.read();
}
```

## 2. Enums and C-Compatible Enums

Use `enum` to represent mutually exclusive variants, optionally carrying named inert payload fields:

```c
enum ParseError {
    Empty;
    InvalidByte { usize offset; u8 byte; }
}
```

When the expected type is known from context, construct variants starting with a dot: `.Empty` or `.InvalidByte { .offset = 2, .byte = 255 }`.

For plain C-layout integer enums inside `@repr(C)` records, annotate the backing C integer type:

```c
@repr(c::uint)
enum Status {
    Ready;
    Busy;
}
```

## 3. Exhaustive `switch` Matching

`switch` forces you to cover every variant explicitly. There is no implicit C-style fallthrough between cases.

```c
fn i32 code_of(ParseError err) {
    return switch (err) {
        case .Empty: { 1 }
        case .InvalidByte { .offset = pos, .byte = _ }: {
            (pos as i32) + 10
        }
    };
}
```

- Every `case` requires a `{ ... }` block.
- Unused fields can be ignored with `_`, and leaving a final expression without `;` lets the `switch` itself return a value.

## 4. `Option<T>`, `Result<T, E>`, and `?`

- **`Option<T>`**: `.some(value)` or `.none`
- **`Result<T, E>`**: `.ok(value)` or `.err(error)` (`result.is_ok` also returns a `bool` success check)

To forward errors upward without writing repetitive `switch` boilerplate, append postfix `?`:

```c
fn Result<i32, ParseError> add_one(Result<i32, ParseError> input) {
    let i32 n = input?;    // returns .err(...) early on failure
    return .ok(n + 1);
}
```

### `@must_use` and `discard`

Ignoring a `Result` without handling it is a compile error. If you intentionally want to drop a `Result` (or any function marked `@must_use`), acknowledge it explicitly with `discard`:

```c
discard add_one(.err(.Empty));

@must_use
fn usize bytes_written() {
    return 16;
}
```

---

- Previous: [Functions and Control Flow](../control-flow-and-functions/)
- Next: [Arrays, Slices, and Views](../arrays-and-slices/)
