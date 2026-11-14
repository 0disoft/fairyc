---
title: 所有权、借用与资源
description: 理解复制（Copy）与移动（Move）、ref/ref mut 借用、resource cleanup 块、region 内存区域、@abilities/burn、@effects 以及 unsafe 指针。
---

本章介绍 Fairy C 最具代表性的设计理念：在编译期防止释放后使用（use-after-free）和遗漏清理调用，同时在需要与操作系统交互时仍允许你下探到 `unsafe` 和 C 指针。

## 1. 复制（Copy）与移动（Move）

小型的惰性值（如 `i32`、`bool`、普通 `struct`）在赋值或传递给函数时会自动复制。

而声明为 `resource` 的值则代表唯一所有权（例如打开的文件句柄或已分配的缓冲区）。按值传递 `resource` 会将所有权 <strong>移动</strong> 给被调用方——之后再使用原变量会被编译器拒绝。

## 2. 使用 `ref` 和 `ref mut` 进行借用

如果想让另一个函数读取或修改数据而不移交所有权，可以使用借用：

- <strong>`ref T`（共享借用）</strong>：多个读取者可以同时存在。
- <strong>`ref mut T`（独占借用）</strong>：同一时间只能有一个活跃的可变借用者。

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

通过 `*n` 可以读取或更新标量借用。返回借用引用的函数必须将其来源绑定到某个输入参数上——返回局部变量的引用会被编译器拒绝，从而彻底杜绝悬垂指针。

## 3. 自动清理：`resource` 与 `cleanup`

不必在每个提前 `return` 的路径上手动配对 `CreateFile` 或 `malloc`，只需为 `resource` 附带一个 `cleanup { ... }` 代码块：

```c
resource Buffer {
    ptr mut u8 data;

    cleanup {
        // 在离开作用域、break、continue 或 return 时自动运行！
    }
}
```

资源字段在定义模块之外保持私有，从而将原始句柄安全地封装在工厂函数和方法 API 之后。如果清理代码本身也会执行外部效果，还可以写 `@effects(...) cleanup { ... }`；持有并销毁该资源的调用方也必须声明相应的效果（`E7102`）。

## 4. `region`：批量内存区域作用域

当需要分配大量生命周期相同的临时缓冲区时，可以使用 `region` 代码块：

```c
fn Result<i32, MyError> parse_temp(Region arena, ref u8[] input) {
    region scratch in arena {
        // scratch 内的临时分配会在离开代码块时一并重置
    }
    return .ok(0);
}
```

## 5. `@abilities` 与线性资源 `burn`

默认情况下，`resource` 会在其所有者离开作用域时自动释放。如果你希望实现线性语义，让遗忘某个值（例如凭证、令牌、回执）直接成为编译错误，可以声明不含 `drop` 的 `@abilities`：

```c
@abilities(store, key)   // 省略 drop 即禁止隐式清理！
resource Receipt {
    u32 amount;
    cleanup {}
}
```

这样的值必须被传递出去或返回（如果离开作用域时仍处于存活状态会报 `E1108`），并且只能在声明它的模块内部使用 `burn receipt;` 显式销毁。（可用的能力包括 `drop`、`store` 和 `key`，其中 `key` 依赖 `store`。）

## 6. `@effects(...)`：显式外部权限

涉及外部 I/O 或宿主资源的函数必须声明 `@effects(...)`：

```c
@effects(console)
fn void greet() {
    // 允许调用控制台相关函数
}
```

常见的副作用名称包括 `console`、`filesystem`、`environment`、`process`、`network`、`clock` 和 `random`。任何调用带有 `@effects(...)` 函数的调用方——或者销毁其 `cleanup` 块声明了副作用的资源——也必须在自己的签名中包含这些副作用（否则会报 `E7102`）。

## 7. `unsafe` 与原始指针（`ptr` / `ptr mut`）

调用 Win32 或 C 函数（`extern C`）时，请使用 `ptr T`、`ptr mut T`，以及 `unsafe` 函数或 `unsafe { ... }` 代码块：

```c
unsafe extern C fn void c_free(ptr mut void p);
```

请记住，`unsafe` 仅用于解锁原始指针转换、解引用和外部函数调用——它 <strong>不会</strong> 绕过 `@effects(...)` 检查。`null` 字面量只能用于初始化或比较（`p == null`、`p != null`）原始的 `ptr T`、`ptr mut T` 或 `unsafe extern C fn` 指针，绝不能用于安全的 `ref T` 或 `ref mut T` 借用。

---

- 上一章：[数组、切片与视图](../arrays-and-slices/)
- 下一章：[模块、包与 std](../modules-and-projects/)
