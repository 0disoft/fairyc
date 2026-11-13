---
title: Memory and Resources
description: Understand resource cleanup, move semantics, ref/ref mut borrowing, and return-origin rules with clear examples.
---

Ownership and borrowing sound intimidating at first, yet the core intuition is simple: **every cleanup-needing value has one owner at a time, and lending it follows clear read/write rules.**

## 1. Plain Values vs. `resource` Types

Plain structs containing integers or booleans copy by value. But if a value holds an OS file handle or heap allocation, copying it would cause two variables to free the same handle twice.

Fairy C solves this with `resource` types and attached `cleanup { ... }` blocks:

```c
@repr(C)
struct raw_text_result {
    u32 status;
    u32 allocation_status;
}

@link_name("fy_std_string_drop_storage_v0")
unsafe extern C fn raw_text_result string_drop_storage_raw(ptr mut void value);

pub resource String {
    ptr mut void data;
    usize capacity;
    usize alignment;
    usize length;

    cleanup {
        let raw_text_result dropped = unsafe {
            string_drop_storage_raw(ref mut this as ptr mut void)
        };
    }
}
```

Whenever a live `String` leaves its scope (`{ ... }`), the compiler automatically calls its `cleanup` block in reverse declaration order.

## 2. Moving Ownership

Passing or assigning a `resource` by value transfers both the handle and cleanup responsibility to the receiver:

- Touching the old variable after moving it is rejected (`E1101`).
- Moving a resource in only one branch of an `if`/`else` leaves its state ambiguous afterward unless resolved (`E1104`).
- Plain `struct` types cannot hold `resource` fields (`E1002`), and you cannot move one field out of a `resource` on its own (`E1103`).

## 3. Borrowing with `ref T` and `ref mut T`

When a helper function only needs temporary access, borrow instead of moving:

- **`ref T` (shared borrow)**: Read-only access. Any number of readers can look at the same value as long as nobody mutates or moves the owner while they are active.
- **`ref mut T` (mutable borrow)**: Exclusive read-write access. No other read, write, or move on that owner may overlap (`E2101`).

```c
fn i32 read(ref i32 value) { return *value; }
fn void edit(ref mut i32 value) { *value = *value + 1; }
fn u8 first(ref u8[] view) { return view[0]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }

fn i32 main() {
    var i32 number = 41;
    let ref mut i32 view = ref mut number;
    edit(view);
    read(view);

    var u8[2] data = [3, 4];
    let ref mut u8[] buffer = ref mut data;
    fill(buffer);
    return *view;
}
```

Even when borrowing a slice view (`view[lo..hi]`), the compiler tracks the underlying root owner consistently so aliasing mutable views cannot overlap.

## 4. Returning Borrows

Two practical rules keep returned references safe:

1. **No returning locals (`E2102`)**: Local variables vanish when a function returns, so you cannot return a `ref` pointing to one.
2. **Single input origin (`E2103`)**: A function returning `ref T` or `ref u8[]` must trace back to static data or a **single** input borrow parameter rather than choosing dynamically between two different inputs. Return an owned value if a function needs to pick between two sources.
