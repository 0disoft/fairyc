---
title: 简介
description: 了解 Fairy C 的设计初衷、核心语法理念，以及它与 Brownie C 的区别。
---

编写 C 语言代码虽然能让你直接掌控内存和操作系统 API，但也很容易忘记调用 `free()`、误触已释放的数据，或者让整数在溢出时静默回绕。

Fairy C 是一门实验性系统编程语言，它在保留类似 C 的无垃圾回收原生执行能力的同时，让编译器在早期就捕获所有权、借用和算术运算上的错误。

## 语言的核心理念

- <strong>资源懂得自我清理</strong>：持有句柄或缓冲区的类型可以声明为 `resource`，并附带一个 `cleanup { ... }` 代码块。当一个存活的资源离开作用域时，编译器会自动安排清理。
- <strong>移动与借用（`ref` 和 `ref mut`）</strong>：你可以转移资源的所有权，或者借出只读（`ref T`）/ 独占可变（`ref mut T`）访问权限，同时由编译器检查是否存在重叠冲突。
- <strong>受检整数运算</strong>：整数溢出和除以零会触发安全的运行时陷阱（trap），而不是静默回绕。
- <strong>显式 `@effects(...)` 标签</strong>：凡是访问控制台、文件系统、时钟或网络的函数，都必须在签名中列出这些副作用。
- <strong>直接 C 互操作</strong>：`@repr(C)` 结构体和 `unsafe extern C fn` 让 Fairy C 可以直接调用现有的 C 库和 Win32 API。

## 当前适用场景

Fairy C 非常适合用来编写小巧的原生命令行工具、C ABI 封装层，以及专注特定功能的 Win32 实用程序。

由于目前的编译器仍处于 <strong>Stage 0 原型阶段</strong>，请优先使用稳定的工作区命令（`new`、`check`、`build`、`run`、`test`、`explain`、`targets`），并将 `fairyc experimental ...` 子命令视为内部编译器开发工具。正式的目标平台一致性支持（`fairyc targets` 目前显示 `supported: no`）以及完整的标准库仍在完善中。

## 与 C++ 和 Rust 有什么区别？

如果你已经熟悉其他系统级语言，可以通过下面几点快速建立印象：

- <strong>与 C++ 的区别</strong>：Fairy C 没有 `#include` 头文件、预处理器宏、异常（`try`/`catch`）、类继承和运算符重载；它改用明确的 `module`/`import`、`Result` + `?` 错误传播和 `resource` 下的 `cleanup` 代码块来管理资源，并在编译期检查 `ref`/`ref mut` 借用冲突，同时在运行时拦截整数溢出与数组越界。
- <strong>与 Rust 的区别</strong>：Fairy C 有意把语法和特性保持得更小、更接近 C，并通过宿主 C11 工具链编译。当前 Stage 0 没有 trait、闭包、宏系统、`async` 或需要手写的 `'a` 生命周期标注；它也不在普通结构体上使用通用 `Drop`，而是把可复制的普通 `struct` 与带 `cleanup { ... }` 的独占 `resource` 分开，并用 `@effects(...)` 显式标记外部 I/O 权限。
- <strong>成熟度</strong>：C++ 和 Rust 已经拥有成熟的生产级生态，而 Fairy C 目前仍处于验证核心理念的 <strong>Stage 0 原型阶段</strong>。

## Fairy C 与 Brownie C

| 维度 | Fairy C (`.fy`) | Brownie C (`.brc`) |
| --- | --- | --- |
| 定位 | 独立的原生程序与库 | 嵌入宿主程序中的规则与字节检查语言 |
| 执行方式 | 经由 C11 编译为原生可执行文件 | 运行在受限的字节码虚拟机中 |
| OS 与 C 访问 | 直接支持 `unsafe extern C fn` 和标准库 I/O | 不允许任意 C 调用、文件或网络访问 |
| 资源预算 | 正常的原生操作系统执行 | 受指令 `--fuel` 预算限制 |
