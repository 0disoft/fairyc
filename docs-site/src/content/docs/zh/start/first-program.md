---
title: 第一个程序
description: 创建你的第一个 Fairy C 包，逐行阅读生成的代码，并运行检查、测试和原生构建。
---

让我们创建一个入门 Fairy C 包，逐行阅读生成的文件，并实际运行它。

## 1. 创建新包

在编译器仓库外部选择一个空目录，并使用唯一的 `--uid` 运行 `fairyc new`：

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

这会生成五个文件：`fairy.toml`、`src/main.fy`、`tests/main.fy`、`.gitignore` 和 `README.md`。

查看 `fairy.toml`：

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

再查看 `src/main.fy`：

```c
module main;

pub fn i32 answer() {
    return 77;
}

fn i32 main() {
    return answer() - 77;
}
```

快速拆解一下这段代码：

- `module main;` 声明当前文件的模块名称。入口文件 `src/main.fy` 必须声明为 `module main;`。
- `pub fn i32 answer()` 将返回类型（`i32`）写在函数名之前，而 `pub` 允许其他模块（例如 `tests/main.fy`）调用该函数。
- `fn i32 main()` 是程序入口点。它返回的 `i32` 值就是进程的退出码（`77 - 77` 返回 `0`）。

## 2. 检查语法、类型与借用

```powershell
fairyc check
```

预期输出：

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

首次运行 `check` 会创建 `fairy.lock`。之后你可以传入 `--locked`，让后续运行校验清单（manifest）是否发生意外漂移：

```powershell
fairyc check --locked
```

## 3. 运行初始测试

`tests/main.fy` 已经包含了一个初始测试。之后若要添加更多测试，只需用 `@test` 标记 `fn void` 函数即可：

```c
module tests::main;

import main as app;

@test
fn void answer_is_77() {
    assert(app::answer() == 77);
}
```

运行测试：

```powershell
fairyc test --locked
```

预期输出：

```text
test tests::main::answer_is_77 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

如果你希望将测试编译为原生二进制文件执行，而不是在解释器中运行，可以传入 `--native`（`fairyc test --native --locked`）。

## 4. 原生运行与构建

编译并运行一个临时原生二进制文件：

```powershell
fairyc run --locked
```

由于 `main()` 返回 `0`，进程会以退出状态 `0` 正常结束。如果想在 `build/hello-fairy.exe`（Windows）或 `build/hello-fairy`（Linux）保留编译后的可执行文件，可以运行：

```powershell
fairyc build --release --locked
```

## 新手常见踩坑

- <strong>在 `fairyc new` 中忘记 `--uid`</strong>：`--uid` 是必填参数，且至少需要两个由点号分隔的小写 kebab-case 分段（例如 `example.hello-fairy`）。
- <strong>`name` 或 `uid` 中包含大写字母</strong>：清单校验仅允许小写的 kebab-case 命名。
- <strong>修改 `fairy.toml` 后立即带 `--locked` 运行</strong>：请先不带 `--locked` 运行一次 `fairyc check` 来刷新 `fairy.lock`。
- <strong>执行 `build` 或 `run` 时出现退出码 `6`</strong>：未能找到宿主机 C 编译器（Windows 上为 MSVC + Windows SDK，Linux 上为 `gcc`/`clang`），或者链接失败。
