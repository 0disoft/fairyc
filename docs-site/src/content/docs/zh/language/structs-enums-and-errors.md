---
title: 结构体、枚举与 Switch
description: 学习 Fairy C 的结构体、字段简写、Type::method 语法、枚举、穷举式 switch 匹配，以及 Option/Result/? 错误传播。
---

当程序规模超过几个整数变量时，你会希望把相关字段组织在一起，并清晰地对多种可能的结果进行建模。Fairy C 通过 `struct`、`enum`、穷举式 `switch` 以及 `Option` / `Result` 来满足这些需求。

## 1. 结构体与字段简写

使用 `struct` 定义普通记录类型。如果希望字段在定义模块之外也可访问，请加上 `pub`：

```c
pub struct Point {
    pub i32 x;
    pub i32 y;
}

fn i32 demo() {
    let i32 x = 10;
    let i32 y = 32;

    // .y 是 .y = y 的简写
    let Point p = Point { .x = x, .y };
    return p.x + p.y;
}
```

只要局部变量与字段同名，就可以使用 `.field` 简写。所有字段都必须按声明顺序提供。

### 使用 `Type::method` 定义方法

在函数名前加上 `Type::` 前缀，即可将其关联到某个 `struct` 或 `resource`。当第一个参数名为 `this`（`ref T this`、`ref mut T this` 或 `T this`）时，可以通过点语法调用：

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

## 2. 枚举与 C 兼容枚举

使用 `enum` 表示互斥的变体（variant），变体还可以选择携带具名的惰性负载字段：

```c
enum ParseError {
    Empty;
    InvalidByte { u8 offset; u8 byte; };
}
```

当上下文已经明确目标类型时，构造变体可以直接以点号开头：`.Empty` 或 `.InvalidByte { .offset = 2, .byte = 255 }`。

对于用在 `@repr(C)` 记录中的纯 C 布局整数枚举，请标注其底层 C 整数类型：

```c
@repr(c::uint)
enum Status {
    Ready;
    Busy;
}
```

## 3. 穷举式 `switch` 匹配

`switch` 强制要求显式覆盖每一个变体。分支之间不存在 C 语言风格的隐式贯穿（fallthrough）。

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

- 每个 `case` 都必须带有 `{ ... }` 代码块。
- 未使用的字段可以用 `_` 忽略；末尾不写 `;` 的表达式可以让整个 `switch` 直接返回一个值。

## 4. `Option<T>`、`Result<T, E>` 与 `?`

- <strong>`Option<T>`</strong>：`.some(value)` 或 `.none`
- <strong>`Result<T, E>`</strong>：`.ok(value)` 或 `.err(error)`（也可以通过 `result.is_ok` 直接获取 `bool` 类型的成功状态）

为了把错误向上传递而不必手写重复的 `switch` 样板代码，可以在表达式后面加上后缀 `?`：

```c
fn Result<i32, ParseError> add_one(Result<i32, ParseError> input) {
    let i32 n = input?;    // 失败时提前返回 .err(...)
    return .ok(n + 1);
}
```

### `@must_use` 与 `discard`

忽略一个 `Result` 而不处理它会导致编译错误。如果你确实有意丢弃某个 `Result`（或任何标记了 `@must_use` 的函数返回值），请使用 `discard` 显式确认：

```c
discard add_one(.err(.Empty));

@must_use
fn usize bytes_written() {
    return 16;
}
```

---

- 上一章：[函数与控制流](../control-flow-and-functions/)
- 下一章：[数组、切片与视图](../arrays-and-slices/)
