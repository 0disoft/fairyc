---
title: Variables, Types, and Operators
description: Learn let vs var, compile-time const, scalar and string types, literals, checked arithmetic, and explicit as casts in Fairy C.
---

If you have ever glanced at C, Fairy C's syntax will feel familiar right away. Statements end with `;` and blocks use `{ }`. The difference is that the compiler catches common footguns—like uninitialized variables, silent integer overflows, and accidental implicit casts—before your program even runs.

## 1. `let`, `var`, and `const`

Use `let` for bindings you do not reassign, and `var` for mutable variables.

```c
let i32 limit = 100;      // cannot be reassigned later
var i32 count = 0;        // can be updated with count = 1;
let auto_inferred = 77;   // type can be omitted when clear from the value
```

- **Initialization is required**: Writing `var i32 count;` without a value is not allowed, eliminating uninitialized-read bugs.
- **`let` only locks rebinding**: Even if a `resource` is bound with `let`, you can still transfer (move) its ownership to another function.

For module-level compile-time constants, use `const` (currently private scalar constants of type `bool`, `u8`, `i32`, or `u32`). Pure scalar helper functions annotated with `@const_eval` can also run inside `const` initializers and `static_assert(...)`:

```c
@const_eval
fn i32 square(i32 n) { return n * n; }

const i32 MAX_RETRY = square(2) + 3;
static_assert(MAX_RETRY == 7);
```

`static_assert(...)` checks a pure compile-time condition during compilation. Comments support `//` line comments, `///` doc comments, and nested `/* ... */` block comments.

## 2. Core Types and Literals

| Category | Type Names | Notes |
| --- | --- | --- |
| Signed integers | `i8`, `i16`, `i32`, `i64`, `isize` | `i32` for general integers, `i64` for 64-bit |
| Unsigned integers | `u8`, `u16`, `u32`, `u64`, `usize` | `u8` for bytes, `usize` for array indexes and lengths |
| Boolean | `bool` | `true`, `false` (never mixed with integers) |
| Floating point | `f32`, `f64` | Bounded transport support such as `@repr(C)` fields |
| C ABI scalars | `c::int`, `c::uint`, `c::size_t`, ... | Used when binding `extern C` functions |
| Text & byte views | `ref str`, `ref CStr`, `ref u8[]` | UTF-8 strings, NUL-terminated C strings, byte slices |

Literals support numeric separators (`_`), hex (`0x`), binary (`0b`), and three string/byte forms:

```c
let i32 million = 1_000_000;
let u32 mask = 0xff_00_aa;
let u8 bits = 0b1010_0101;

let ref str text = "Hello Fairy"; // UTF-8 string view
let ref u8[] raw = b"PK\x03\x04";  // byte slice literal
let ref CStr title = c"Window";   // NUL-terminated C string view
```

Note that Stage 0 does not have character literals (`'a'`) or numeric suffixes (`10u32`). Types come from explicit annotations or contextual expectation.

## 3. Operators and Checked Arithmetic

Standard arithmetic (`+`, `-`, `*`, `/`, `%`), bitwise operators (`~`, `&`, `|`, `^`, `<<`, `>>`), and compound assignments (`+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, `|=`) are built in. (`++`, `--`, and `? :` do not exist.)

A key safety rule:

```c
var u8 x = 250;
x += 10; // Unlike C wrapping silently to 4, execution traps immediately!
```

- **Overflow and divide-by-zero trap**: Arithmetic overflow, unsigned underflow, dividing by zero, and out-of-range shifts stop safely instead of corrupting values.
- **No chained comparisons**: Write `(0 < x) && (x < 10)` instead of `0 < x < 10` (`E0009`).
- **Conditions require `bool`**: You cannot write `if (count)`; always write an explicit comparison such as `if (count != 0)`.

## 4. Explicit Type Casts (`as`)

Different integer types never mix implicitly. Use `as` for explicit, value-preserving widening conversions:

```c
let u8 byte_val = 40;
let i32 a = byte_val as i32;      // u8 -> i32
let u32 b = byte_val as u32;      // u8 -> u32
let usize c = b as usize;         // u32 -> usize
```

Safe `as` currently covers widening pairs that never lose data (`u8 -> i32/u32/usize`, `u32 -> usize`, plus `i32 <-> c::int` on 32-bit `c::int` targets). Narrowing or sign-changing conversions use explicit fallible helpers such as those in `std::num`.

---

Next: [Functions and Control Flow](../control-flow-and-functions/)
