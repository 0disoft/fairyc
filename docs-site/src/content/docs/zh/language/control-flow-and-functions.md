---
title: 函数与控制流
description: 学习 Fairy C 中的函数声明、参数标签、标量泛型、if 表达式、while 循环、C 风格 for 循环、for-in 迭代以及 defer 语句。
---

弄清楚变量和运算符之后，接下来看看函数与控制流。Fairy C 保持了类似 C 语言的编码节奏，同时加入了能产生值的 `if` 表达式、安全的 `for-in` 循环以及 `defer` 等实用特性。

## 1. 函数与参数标签

函数声明遵循 `fn ReturnType name(Parameters)` 的格式。没有返回值时使用 `void`。

```c
fn i32 clamp(i32 value, i32 minimum, i32 maximum) {
    if (value < minimum) { return minimum; }
    if (value > maximum) { return maximum; }
    return value;
}

fn void log_step() {
    // void 函数可以省略显式的 return;
}
```

与 Rust 不同，<strong>非 `void` 函数体不会在省略 `return` 时隐式返回最后一行的值</strong>。请务必写出 `return value;`。

### 保持位置的参数标签

调用非泛型的本模块函数时，可以使用 `.parameter_name = value` 来让参数含义更清晰：

```c
let i32 result = clamp(77, .minimum = 0, .maximum = 100);
```

参数标签不能调整参数顺序，也不能提供默认值——每个标签都必须严格对应参数声明的位置——因此它们纯粹用于提升可读性并防止传错参数。

## 2. 泛型标量函数（`<T>`）

普通函数可以接收显式的标量类型参数 `<T>`，并以 `fn_name<i32>(...)` 的形式调用：

```c
fn T pick_first<T>(T first, T second) {
    return first;
}

fn i32 main() {
    return pick_first<i32>(77, 0);
}
```

目前类型参数涵盖核心标量类型（`bool`、`u8`、`i32`、`u32`、`usize`），并在编译期进行特化。（Stage 0 暂不支持调用处的类型推断以及用户自定义的泛型结构体/枚举。）

## 3. 作为表达式的 `if / else`

条件表达式始终需要圆括号 `(...)`。如果每个分支都以不带分号的表达式结尾，`if` 会直接求值为一个结果：

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

所有分支必须产生完全相同的类型。

## 4. `while` 与两种形式的 `for`

### `while` 与经典 `for`

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

`for (init; cond; post)` 中的任意子句都可以省略（省略条件时默认为 `true`）。执行 `continue` 时，会先运行 `post` 子句，再进行下一次条件判断。

### 遍历数组和切片视图的 `for-in`

为了避免遍历数组或视图时出现差一索引错误（off-by-one），可以使用 `for-in`：

```c
var i32[4] items = [10, 20, 30, 40];

// 复制每个元素进行读取
var i32 total = 0;
for (let i32 item in items) {
    total += item;
}

// 原地可变借用每个元素
for (let ref mut i32 item in items) {
    *item += 1;
}
```

你也可以直接遍历子区间，例如 `for (let u8 b in data[1..4]) { ... }`。

## 5. `assert`、`defer` 与 `source_location()`

- <strong>`assert(cond);`</strong>：当 `cond` 为 `false` 时立即触发陷阱中止。
- <strong>`defer call(...);`</strong>：注册一个内部 `void` 函数调用（其标量参数在注册时求值），安排在离开当前代码块前执行——执行时机早于隐式的资源销毁或 `region` 重置。
- <strong>`source_location()`</strong>：内置的零参数内建函数，返回类型为 `ref str` 的 `"path:line:col@offset"` 字符串，非常适合用于诊断和日志辅助函数。

---

- 上一章：[变量、类型与运算符](../basics/)
- 下一章：[结构体、枚举与 Switch](../structs-enums-and-errors/)
