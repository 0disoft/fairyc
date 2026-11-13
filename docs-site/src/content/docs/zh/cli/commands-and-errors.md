---
title: CLI 用法与常见错误
description: 常用的 fairyc CLI 命令、退出码说明以及常见编译错误的快速排查方法。
---

这里整理了一份实用的速查表，涵盖日常最常用的稳定 `fairyc` CLI 命令以及初次使用时最容易遇到的编译器错误。

## 日常命令

| 命令 | 使用场景 |
| --- | --- |
| `fairyc new <name> --uid <uid>` | 创建一个新的包目录 |
| `fairyc check [--locked]` | 快速执行语法、类型和借用检查，不编译 C 代码 |
| `fairyc build [--release] [--locked]` | 生成 `build/<name>.exe`（Windows）或 `build/<name>`（Linux） |
| `fairyc run [--release] [--locked] [-- <args...>]` | 立即编译并运行临时二进制文件 |
| `fairyc test [--native] [--locked]` | 在解释器中运行 `tests/`，或以原生二进制方式运行（`--native`） |
| `fairyc explain <code>` | 查看如 `E2101` 这类诊断码的详细说明 |

请坚持使用以上稳定命令——`fairyc experimental ...` 子命令是供编译器开发使用的内部工具。

## 退出码一览

- <strong>`0`</strong>：成功（如果 `main()` 未触发陷阱，`run` 也会原样透传其非零返回值）
- <strong>`2`</strong>：CLI 参数或调用语法有误
- <strong>`3`</strong>：`.fy` 源码或清单文件中存在编译错误
- <strong>`5`</strong>：运行时陷阱（整数溢出、除以零、`assert` 失败）
- <strong>`6`</strong>：未找到宿主机 C 编译器/链接器，或编译链接失败
- <strong>`7`</strong>：`fairy.toml` 与 `fairy.lock` 不匹配（请重新运行 `fairyc check`）

## 常见错误及修复方法

看到任何错误码时，都可以运行 `fairyc explain <CODE>`。新手遇到的错误大多属于以下几类：

1. <strong>`E1001`（未初始化的变量）</strong>：把 `var i32 count;` 改成 `var i32 count = 0;`。
2. <strong>`E1006` / `E0009`（混合整数类型或链式比较）</strong>：使用 `as` 进行显式转换，并将 `0 < x < 10` 改写为 `(0 < x) && (x < 10)`。
3. <strong>`E1101`（移动后使用）</strong>：你按值传递了一个 `resource` 后又尝试再次使用它。如果调用方后续还需要使用，请改用 `ref` 或 `ref mut` 借用。
4. <strong>`E2101`（借用冲突）</strong>：当某个 `ref` 借用在代码块后续仍会被使用时，你对原变量进行了修改或移动。请在使用完该借用之后再执行修改。
5. <strong>`E4101`（忽略了 `Result`）</strong>：使用 `switch`、`?` 或显式 `discard` 来处理返回的 `Result`。
6. <strong>`E7102`（缺少 `@effects(...)`）</strong>：将被调用函数的副作用（如 `console` 或 `filesystem`）添加到当前函数的 `@effects(...)` 属性中。
