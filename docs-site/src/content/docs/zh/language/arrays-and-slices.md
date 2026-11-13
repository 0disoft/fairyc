---
title: 数组、切片与视图
description: 学习定长数组 T[N]、切片视图 ref T[] / ref mut T[]、区间切片、结构体内的内联数组字段，以及 str/u8[]/CStr 的区别。
---

系统级代码经常需要处理连续的内存块。Fairy C 将 <strong>定长数组</strong>（编译期已知长度）与 <strong>切片视图</strong>（不发生拷贝地借用一段内存区间）区分开来。

## 1. 定长数组（`T[N]`）

数组类型写作 `ElementType[Length]`（Stage 0 支持 `1` 到 `4096` 的长度）：

```c
let i32[3] scores = [10, 20, 30];
var u8[16] buffer = [0; 16];       // 将 0 重复 16 次

let usize len = scores.len;        // 3 (usize)
let i32 first = scores[0];
buffer[1] = 65;                    // 可变，因为 buffer 是 var
buffer[1] += 1;                    // 66
```

- <strong>索引的类型是 `usize`</strong>：像 `0` 这样的整数字面量在索引位置会自动采用 `usize` 类型。
- <strong>始终执行边界检查</strong>：在 `>= len` 的索引处读取或写入会立即触发 `bounds` 陷阱，而不会读到垃圾内存。

非 `@repr(C)` 结构体也可以直接嵌入内联有界数组字段：

```c
struct Packet {
    u8[8] bytes;
    usize length;
}
```

## 2. 切片视图（`ref T[]`、`ref mut T[]`）

按值传递整个定长数组会复制其中的元素。如果你只想就地查看或修改元素，请改为借用切片视图：

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

- <strong>`ref T[]`</strong>：只读视图，支持 `.len`、`.is_empty` 以及带边界检查的 `view[index]` 读取。
- <strong>`ref mut T[]`</strong>：独占可变视图，当底层存储为 `var` 时允许写入元素。

### 区间切片（`arr[start..end]`）

使用半开区间 `start..end` 对 `ref u8[]` 视图进行切片（或直接从具名数组借用一段区间，例如 `ref points[1..3]`）：

```c
let ref u8[] full = b"Fairy C";
let ref u8[] head = full[0..5];   // b"Fairy"
let ref u8[] tail = full[6..];    // b"C"
let ref u8[] all = full[..];      // 整个视图
```

无效区间（`start > end` 或 `end > len`）会立即触发 `bounds` 陷阱。

## 3. 字符串与字节字面量对比

| 字面量 | 类型 | 用途 |
| --- | --- | --- |
| `"Fairy"` | `ref str` | 经过校验的 UTF-8 文本视图；提供字节长度 `.len` 和 `.is_empty` |
| `b"Fairy\xff"` | `ref u8[]` | 原始字节切片；支持索引 `b[0]` 和切片 `b[1..3]` |
| `c"Fairy"` | `ref CStr` | 保证以 NUL（`\0`）结尾的视图，专用于 C API |

`ref str` 不允许直接按字节索引 `s[0]`，以免意外切分合法的 UTF-8 字符边界。需要访问原始字节时，请使用 `std::text::as_bytes(s)` 进行显式转换。

---

- 上一章：[结构体、枚举与 Switch](../structs-enums-and-errors/)
- 下一章：[所有权、借用与资源](../memory-and-resources/)
