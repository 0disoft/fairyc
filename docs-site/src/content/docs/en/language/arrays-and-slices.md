---
title: Arrays, Slices, and Views
description: Learn fixed arrays T[N], slice views ref T[] / ref mut T[], range slicing, inline struct array fields, and str/u8[]/CStr differences.
---

Systems code works constantly with contiguous memory blocks. Fairy C separates **fixed-size arrays** (whose length is known at compile time) from **slice views** (which borrow a range of memory without copying it).

## 1. Fixed Arrays (`T[N]`)

Write array types as `ElementType[Length]` (Stage 0 supports lengths from `1` to `4096`):

```c
let i32[3] scores = [10, 20, 30];
var u8[16] buffer = [0; 16];       // repeat 0 sixteen times

let usize len = scores.len;        // 3 (usize)
let i32 first = scores[0];
buffer[1] = 65;                    // mutable because buffer is var
buffer[1] += 1;                    // 66
```

- **Indexes have type `usize`**: Integer literals like `0` automatically take `usize` in index position.
- **Bounds checks always run**: Reading or writing at an index `>= len` traps immediately with `bounds` instead of reading garbage memory.

Non-`@repr(C)` structs can also embed inline bounded arrays as fields:

```c
struct Packet {
    u8[8] bytes;
    usize length;
}
```

## 2. Slice Views (`ref T[]`, `ref mut T[]`)

Passing a whole fixed array by value copies its elements. When you just want to inspect or mutate elements in place, borrow a slice view instead:

```c
fn u8 first_byte(ref u8[] view) {
    if (view.is_empty) { return 0; }
    return view[0];
}

fn void clear_head(ref mut u8[] view) {
    if (view.len > 0) {
        view[0] = 0;
    }
}

fn u8 demo() {
    var u8[4] data = [70, 97, 105, 114];
    clear_head(ref mut data);
    return first_byte(ref data);
}
```

- **`ref T[]`**: Read-only view supporting `.len`, `.is_empty`, and checked `view[index]` reads.
- **`ref mut T[]`**: Exclusive mutable view allowing element writes when the underlying storage is `var`.

### Range Slicing (`arr[start..end]`)

Slice a `ref u8[]` view (or borrow a range directly from a named array like `ref points[1..3]`) using half-open `start..end` ranges:

```c
let ref u8[] full = b"Fairy C";
let ref u8[] head = full[0..5];   // b"Fairy"
let ref u8[] tail = full[6..];    // b"C"
let ref u8[] all = full[..];      // entire view
```

Invalid ranges (`start > end` or `end > len`) trap immediately with `bounds`.

## 3. String and Byte Literals Compared

| Literal | Type | Purpose |
| --- | --- | --- |
| `"Fairy"` | `ref str` | Validated UTF-8 text view; exposes byte `.len` and `.is_empty` |
| `b"Fairy\xff"` | `ref u8[]` | Raw byte slice; supports indexing `b[0]` and slicing `b[1..3]` |
| `c"Fairy"` | `ref CStr` | Guaranteed NUL-terminated (`\0`) view for C APIs |

`ref str` prevents direct byte indexing `s[0]` so valid UTF-8 boundaries aren't accidentally split. When you need raw bytes, convert explicitly with `std::text::as_bytes(s)`.

---

- Previous: [Structs, Enums, and Switch](../structs-enums-and-errors/)
- Next: [Ownership, Borrows, and Resources](../memory-and-resources/)
