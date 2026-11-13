---
title: Ownership, Borrows, and Resources
description: Understand Copy vs Move, ref/ref mut borrows, resource cleanup blocks, region arenas, @abilities/burn, @effects, and unsafe pointers.
---

This chapter covers Fairy C's defining ideas: preventing use-after-free and forgotten cleanup calls at compile time, while still letting you drop down to `unsafe` and C pointers when talking to the OS.

## 1. Copy vs Move

Small inert values (`i32`, `bool`, plain `struct`s) copy automatically when assigned or passed to a function.

Values declared as a `resource`, however, represent unique ownership (such as an open file handle or allocated buffer). Passing a `resource` by value **moves** ownership to the callee—using the original variable afterward is rejected by the compiler.

## 2. Borrowing with `ref` and `ref mut`

To let another function read or modify data without giving away ownership, borrow it:

- **`ref T` (shared borrow)**: Multiple readers may coexist.
- **`ref mut T` (exclusive borrow)**: Only one mutable borrower can be active at a time.

```c
fn i32 read_num(ref i32 n) {
    return *n;
}

fn void add_ten(ref mut i32 n) {
    *n += 10;
}

fn i32 main() {
    var i32 score = 67;
    add_ten(ref mut score);
    return read_num(ref score);   // 77
}
```

Read or update a scalar borrow via `*n`. Functions that return a borrowed reference must tie it back to an input parameter's origin—returning a reference to a local variable is rejected so dangling pointers cannot happen.

## 3. Automatic Cleanup: `resource` and `cleanup`

Instead of pairing every `CreateFile` or `malloc` manually across every early `return`, attach one `cleanup { ... }` block to a `resource`:

```c
resource Buffer {
    ptr mut u8 data;

    cleanup {
        // Runs automatically on scope exit, break, continue, or return!
    }
}
```

Resource fields stay private outside the defining module, keeping raw handles encapsulated behind safe factory and method APIs.

## 4. `region`: Bulk Arena Scopes

When allocating many short-lived buffers that all share one lifetime, use a `region` block:

```c
fn Result<i32, MyError> parse_temp(Region arena, ref u8[] input) {
    region scratch in arena {
        // Temporary allocations inside scratch reset together at block exit
    }
    return .ok(0);
}
```

## 5. `@abilities` and Linear `burn`

By default, a `resource` drops automatically when its owner goes out of scope. If you want linear behavior where forgetting a value (an obligation ticket, token, receipt) is a compile error, declare `@abilities` without `drop`:

```c
@abilities(store, key)   // omitting drop forbids implicit cleanup!
resource Receipt {
    u32 amount;
    cleanup {}
}
```

Such a value must be passed or returned (`E1108` if left live at scope exit), and can only be destroyed explicitly inside its declaring module using `burn receipt;`. (Valid abilities are `drop`, `store`, and `key`, where `key` requires `store`.)

## 6. `@effects(...)`: Explicit External Authority

Functions touching external I/O or host resources declare `@effects(...)`:

```c
@effects(console)
fn void greet() {
    // Allowed to call console functions
}
```

Common names include `console`, `filesystem`, `environment`, `process`, `network`, `clock`, and `random`. Any caller invoking an `@effects(...)` function—or dropping a resource whose `cleanup` declares effects—must include those effects too (`E7102`).

## 7. `unsafe` and Raw Pointers (`ptr` / `ptr mut`)

When calling Win32 or C functions (`extern C`), use `ptr T`, `ptr mut T`, and an `unsafe` function or `unsafe { ... }` block:

```c
unsafe extern C fn void c_free(ptr mut void p);
```

Keep in mind that `unsafe` only unlocks raw pointer casts, dereferences, and foreign calls—it does **not** bypass `@effects(...)` checking.

---

- Previous: [Arrays, Slices, and Views](../arrays-and-slices/)
- Next: [Modules, Packages, and std](../modules-and-projects/)
