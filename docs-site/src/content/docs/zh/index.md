---
title: Fairy C
description: 一门具备显式资源清理、借用检查与 C 语言互操作能力的实验性原生系统编程语言。
---

Fairy C 是一门实验性系统编程语言，旨在像 C 语言一样构建小巧的原生二进制程序，同时在编译期捕获资源泄漏、悬垂借用和整数溢出等常见错误。

目前的 `fairyc` 编译器是一个用 Go 编写的 Stage 0 原型。它通过 C11 后端，调用你系统上的 C 工具链（Windows 上为 MSVC + Windows SDK，Linux 上为 GCC 或 Clang）来编译 `.fy` 包。

## 从哪里开始

- <strong>快速入门</strong>：[简介](./start/introduction/) · [安装与环境配置](./start/installation/) · [第一个程序](./start/first-program/)
- <strong>语言教程</strong>：[变量、类型与运算符](./language/basics/) · [函数与控制流](./language/control-flow-and-functions/) · [结构体、枚举与 Switch](./language/structs-enums-and-errors/) · [数组、切片与视图](./language/arrays-and-slices/) · [所有权、借用与资源](./language/memory-and-resources/) · [模块、包与 std](./language/modules-and-projects/)
- <strong>实战示例</strong>：[C 互操作与 Win32 应用](./examples/c-interop-and-win32/)
- <strong>CLI 与现状</strong>：[CLI 用法与常见错误](./cli/commands-and-errors/) · [当前支持范围与限制](./support/status-and-limits/)

## Fairy C 与 Brownie C

**Fairy C** 用于构建独立的原生可执行文件以及 C/操作系统库；而 **Brownie C** 则是另一门独立的嵌入式规则与字节转换语言，运行在受限的虚拟机中。Brownie C 自身的编译器和虚拟机正是用 Fairy C 编写的。
