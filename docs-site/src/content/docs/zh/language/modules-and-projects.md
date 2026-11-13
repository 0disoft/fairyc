---
title: 模块、包与 std
description: 学习 Fairy C 的模块与导入语法、pub 可见性、fairy.toml 工作区配置以及当前的 std 标准模块。
---

当你需要把代码拆分到多个文件或引入其他库时，就会用到 Fairy C 的模块和包系统。这里没有头文件（`#include`），也没有 C 预处理器宏。

## 1. 文件路径与 `module` 名称对应

每个 `.fy` 文件都以一条 `module` 声明开头，名称与它在源码根目录下的路径相对应（`src/app/math.fy` -> `module app::math;`）：

```c
module app::math;

pub fn i32 add(i32 a, i32 b) {
    return a + b;
}
```

只有标记了 `pub` 的条目才对其他模块可见。

## 2. 显式别名导入（`import ... as ...`）

每次导入都必须指定别名（`as name`），让所有外部名称的来源都保持清晰明确的限定：

```c
module main;

import app::math as math;
import std::io as io;

@effects(console)
fn i32 main() {
    let i32 v = math::add(70, 7);
    discard io::write_i32(v);
    return 0;
}
```

不允许使用通配符导入（`*`）或循环模块依赖。

## 3. `fairy.toml`、工作区与 `fairy.lock`

在 `fairy.toml` 中定义包信息与直接依赖：

```toml
[package]
name = "my_app"
uid = "example.my-app"
version = "0.1.0"

[dependencies]
answer = { path = "../libs/answer" }
```

- 根目录下的 `[workspace]` 节允许你一起管理多个包（`examples/workspace_quickstart` 展示了这种目录结构）。
- 运行 `fairyc check` 或 `fairyc test` 会将解析后的包依赖图记录到 `fairy.lock` 中。当你希望在依赖发生漂移时让构建直接失败，可以传入 `--locked`。
- 需要额外链接器设置的 Windows 目标可以配合 `fairy.windows.json` 使用。

## 4. 可用的 `std` 模块

当包链接了 `std` SDK 根目录时（如仓库示例中所示），可以使用以下模块：

| 模块 | 用途 | 示例函数 / 类型 |
| --- | --- | --- |
| `std::io` | 控制台 I/O（`@effects(console)`） | `write`, `write_i32`, `write_i64`, `write_bytes`, `write_stderr` |
| `std::vec` | 可增长容器 `Vec<T>` | `empty<T>`, `push<T>`, `pop<T>`, `at<T>`, `truncate<T>`, `clear<T>` |
| `std::text` | UTF-8 字符串与 `CString` 辅助函数 | `validate_utf8`, `copy`, `append`, `as_bytes`, `cstring_from_text` |
| `std::bytes` | 字节切片与缓冲区辅助函数 | 字节比较与操作辅助函数 |
| `std::fs` | 文件系统操作（`@effects(filesystem)`） | `open_read`, `create_truncate`, `read`, `write_all`, `seek_start`, `close` |
| `std::num` | 受检解析与窄化类型转换 | `parse_i64_decimal`, `usize_from_i64`, `u8_from_i64` |
| `std::process` | 进程参数与退出处理 | `process::arg(...)` |
| `std::alloc` | 显式分配器接口 | 显式缓冲区/资源分配原语 |

---

- 上一章：[所有权、借用与资源](../memory-and-resources/)
- 下一章：[C 互操作与 Win32](../../examples/c-interop-and-win32/)
