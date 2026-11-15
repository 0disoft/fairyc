---
title: 当前支持范围与限制
description: 了解 Stage 0 Fairy C 目前已支持的功能、尚未完成的部分，以及 fairyc targets 输出的含义。
---

Fairy C 目前是一个活跃开发中的 <strong>Stage 0 编译器原型</strong>。了解当前已经可用的功能与仍在开发中的部分，有助于建立合理的预期。

> <strong>验证基线</strong>：`2026-11-16` · 提交 [`e1d48cb7`](https://github.com/0disoft/fairyc/commit/e1d48cb7)

## 目前已支持的功能

- 包管理与本地 `[workspace]` 路径依赖（`fairy.toml` + `fairy.lock`）
- 带 `cleanup` 的 `resource` 所有权、`ref`/`ref mut` 借用、切片视图（`u8[]`）以及单一来源（single-origin）的借用返回
- 受检整数算术、`Option<T>`、`Result<T, E>`、`?` 错误传播以及 `@effects(...)`
- 直接 C ABI 互操作（`@repr(C)`、`unsafe extern C fn`、同步 C 回调、Win32 桥接示例）
- 核心 Stage 0 SDK 模块（`std::io`、`std::fs`、`std::bytes`、`std::alloc`、`std::text`、`std::vec`、`std::process`）

在 Windows 上配合 MSVC + Windows SDK，或在 Linux 上配合 GCC/Clang，你可以在上述语言子集范围内构建并运行控制台工具和 Win32 示例。需要注意的是，由于正式的目标规范一致性和完整的标准库覆盖尚未完成，`fairyc targets` 目前对 `x86_64-windows-msvc` 和 `x86_64-linux-gnu` 都会输出 `supported: no`。

## 当前限制

- <strong>尚非完整的通用工具链</strong>：目前不承诺全平台覆盖、任意 C++ 兼容性或通用性能保证。
- <strong>标准库尚不完整</strong>：暂不包含异步运行时、HTTP 客户端/服务端以及跨平台 GUI 工具包。
- <strong>仅支持本地依赖</strong>：目前没有远程包注册中心或自动下载器。
- <strong>语言特性限制</strong>：泛型仅限于受约束的标量类型，返回借用的回调/闭包受到限制（`E2104`、`E5104`），且 `f32`/`f64` 目前主要用于 `@repr(C)` 的 C 结构体布局。
