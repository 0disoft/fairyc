---
title: Memory and Resources
description: Whole-value affine resources, deterministic cleanup, shared vs. mutable borrows, and return origin tracking in Fairy C.
---

Fairy C manages resources without a garbage collector by combining **move-only `resource` types** with **compile-time borrow and lifetime checking**.

## Copyable Values vs. Affine `resource` Types

Plain structs containing integers or booleans copy by value. In contrast, a `resource` declaration defines an affine (single-owner, move-only) handle whose `cleanup` block runs automatically when the value goes out of scope while still live:

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

Key ownership rules enforced at compile time:

- **Move on pass or assign**: Moving a `resource` transfers responsibility for running `cleanup`. Reading it again afterward fails (`E1101`).
- **No partial moves or inert containers**: You cannot move individual fields out of a `resource` (`E1103`), nor store a `resource` inside an ordinary inert `struct` (`E1002`).
- **Branch join state**: If one `if`/`else` branch moves a resource and the other does not, the variable enters a `MaybeLive` state unless resolved before leaving the scope (`E1104`).
- **Reverse drop order**: Live resources in a block clean up in reverse declaration order; `Option`/`Result` payloads clean up conditionally based on their tag.

## Shared (`ref T`) and Exclusive (`ref mut T`) Borrows

Instead of transferring ownership, functions can borrow values safely:

- `ref T`: Shared read-only borrow. Multiple active shared borrows may coexist, provided the root owner is not mutated or moved while any loan is active.
- `ref mut T`: Exclusive mutable borrow. Prevents any other read, write, or move through the root owner until the borrow finishes.
- **Whole-root loan rule**: Borrowing a field or sub-slice (`view[lo..hi]`) locks the owning root consistently so no aliasing mutable access can overlap (`E2101`).

```c
fn i32 read(ref i32 value) { return *value; }
fn void edit(ref mut i32 value) { *value = *value + 1; }
fn u8 first(ref u8[] view) { return view[0]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }

fn i32 example() {
    var i32 number = 41;
    let ref mut i32 view = ref mut number;
    edit(view);
    var u8[2] data = [3, 4];
    let ref mut u8[] buffer = ref mut data;
    fill(buffer);
    return *view;
}
```

## Return-Origin Rules and `Region` Arenas

When a function returns a borrow (`ref T` or `ref u8[]`), Stage 0 enforces the **single-origin rule**:

- The returned borrow must trace back either to static data or to **at most one** borrowed parameter origin (`E2103` if two different parameters could both be returned).
- Returning a borrow pointing to a local stack variable is rejected (`E2102`).
- Temporary region-based allocations (`Region`) prevent escaping the region scope (`E3101..E3104`) and forbid types that require `cleanup` in plain arenas (`E3102`).

## Common Ownership & Borrow Errors

- **`E1101` (Use after move)**: A `resource` was already passed by value or consumed. Pass `ref` / `ref mut` instead when the caller still needs it.
- **`E2101` (Loan conflict)**: Mutating or moving a variable while a `ref` view still borrows from it later in the scope.
- **`E2103` (Ambiguous return origin)**: A function returning `ref u8[]` chooses between two input borrows; pass/return an owned value or redesign to return from one parameter.
