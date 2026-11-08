# Fairy C

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/brand/fairyc-logo-horizontal-dark-v2.png">
    <img src="assets/brand/fairyc-logo-horizontal-v1.png" alt="Fairy C — a cat fairy holding a turquoise C medallion" width="540">
  </picture>
</p>

- Status: Stage 0 implementation prototype
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

- License: [Apache-2.0](LICENSE) for project-owned components; see
  [licensing scope](LICENSING.md) and [third-party notices](THIRD_PARTY_NOTICES.md).

Fairy C is an experimental Stage 0 systems programming language for native software that needs predictable control over memory and calling costs. It provides explicit resource ownership, compile-time borrow checking, and a narrow C interoperability boundary, without a garbage collector or a heavy runtime.

**Measured build outputs + cache: 140.5 KiB retained after 20 source edits**

Windows/amd64 · `dev` · `many-modules` · configured `modules=16` · one sequence ·
compiler and C toolchain excluded · [Fairy-only snapshot and raw receipt](docs/benchmarks/fairy-20261103/README.md)

Fairy C does not aim to outperform every hand-tuned C program or make every C ownership graph safe. Its narrower goal is to reject common ownership, aliasing, bounds, and effect mistakes in ordinary Fairy C code while keeping resource lifetimes and foreign-call costs visible to the programmer.

## Experimental C interoperability

The current boundary supports `unsafe extern C fn` imports and synchronous, call-bounded, non-capturing callbacks. The [C callback interop example](examples/c_interop_callback/README.md) connects checked-in `main.fy` and `legacy.c`: the C `legacy_filter` passes `[2, 12, 7, 20, 1]` to the Fairy predicate `keep_large`, which accepts values `>= 10` and updates an explicit visit counter. After five synchronous callbacks, C reports `accepted=2 callbacks=5`. From the repository root on Windows amd64 with Go and MSVC/SDK, run:

```text
go run ./tools/dev/fairyc-c-interop --profile low
go run ./tools/dev/fairyc-c-interop --profile optimized
```

This is a development harness for synchronous callbacks, not a source-export or public object-library API. Generated C11 is a compiler-private bootstrap transport, and native Linux execution is deferred.

<a id="human-quickstart"></a>
## Quickstart

### Get the compiler

Fairy C does not yet offer pre-built public installers. To build the compiler from source, follow the [Windows compiler build guide](DEVELOPMENT.md#building-the-windows-compiler-binary). If you are using an experimental ZIP distribution, extract it and keep the license files together. Go and Bun are required only for building the compiler from source, not for running the built executable.

The recommended environment is Windows (x86_64/AMD64) with MSVC and the Windows SDK.

For a small native desktop application, see the [minimal Win32 GUI example](examples/win32_minimal/README.md). Its optional `fairy.windows.json` selects the GUI subsystem, C bridge files, SDK libraries and `.rc` resources while keeping Fairy's ordinary `main()` entry.
Install Microsoft's [C++ Build Tools](https://learn.microsoft.com/en-us/cpp/overview/acquire-msvc)
(Desktop development with C++ workload) to enable native compilation and testing.
WSL2 or a virtual machine is not required; basic source checking and semantic analysis run independently of the C toolchain. Other targets and CI probes are experimental and not yet guaranteed for general use.

Contributors can also build and test the compiler on Linux using Go with GCC or Clang (see [Contributing](CONTRIBUTING.md)). SDK stage0.79 provides Linux support for atomic file replacement, exact reads, file removal, and UTF-8 process arguments. For example, the [task-manager example](examples/task_manager_cli/README.md) supports native builds and tests via `fairyc build/run/test --native` on Linux (amd64) with GCC or Clang.

### Create, test, and run

Once `fairyc` is built, you can create and test a new project starter from any working directory. If `fairyc` is not on your PATH, use its relative or absolute path.

```text
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
fairyc check
fairyc check --locked
fairyc test --list --json --locked
fairyc test --json --locked
fairyc test --locked
fairyc test --native --locked
fairyc test --native --release --locked
fairyc run --locked
fairyc run --release --locked
```

The generated `src/main.fy` contains a minimal complete program:

```text
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

The test runner reports one passed test and no failures, and both run modes exit with code 0.

The first check validates the source files and generates `fairy.lock`. On Windows/amd64 with MSVC installed, `fairyc test --native --locked` compiles each test into an isolated binary image, while `fairyc run --locked` compiles the program through generated C and executes the resulting native binary with stdin attached. The generated `README.md` lists next steps and includes a `std::io::write` example. `fairyc new` never overwrites or merges into an existing directory.

### Project status and reporting issues

This repository provides the Go-based Stage 0 prototype compiler, rather than a self-hosted or production-ready release. Background on verification is in [readiness evidence](docs/ops/parser-public-readiness.md) and the [CI run on `34fe041c`](https://github.com/0disoft/fairyc/actions/runs/35103049037).

For bug reports and general feedback, open a [GitHub Issue](https://github.com/0disoft/fairyc/issues) with the compiler commit or build hash, OS and toolchain version, a minimal reproducible example, expected vs. actual behavior, and the compiler diagnostic output.

Do not post sensitive credentials, confidential code, or vulnerability details in public issues. Report security issues to the maintainer directly.

## Current phase

The Go Stage 0 compiler now implements immutable source bytes, a lossless lexer, compact recoverable
syntax, error-free AST lowering, typed HIR, resource and borrow Flow, verified MIR, a bounded
interpreter, deterministic C11 transport, strict local workspaces, stable `new`/`check`/`build`/
`run`/`test`/C emission commands, and experimental Atlas fact export.

The step-by-step record of what each phase implemented, including the command surface at the
time, has moved to [docs/development/implementation-status.md](docs/development/implementation-status.md).

## Try the current prototype

Examples that drive a checkout directly with `go run` have moved to
[docs/development/prototype-usage.md](docs/development/prototype-usage.md). They are examples, not
an authority on command behavior.

## Build footprint

The current [Fairy-only snapshot](docs/benchmarks/fairy-20261103/README.md) records 22
output-verified build/run pairs: one cold build, one no-op build, and edits 1–20.

### Historical cross-language snapshot (`ec446d2d`)

The chart and receipt below remain from commit `ec446d2d` and were not regenerated or overwritten.
That run compared Fairy C, Zig, Rust, and C on Windows/amd64 over 20 source edits, one sequence per
configuration. The current Fairy-only run did not repeat those comparisons.

![Historical cross-language build footprint](docs/benchmarks/latest/summary.svg)

[Methodology](docs/benchmarks/README.md) ·
[Historical raw receipt](docs/benchmarks/latest/receipt.json) ·
[Historical full footprint SVG](docs/benchmarks/latest/footprint.svg) ·
[Historical HTML report](docs/benchmarks/latest/report.html)

[File-search CLI benchmark](docs/benchmarks/file-search/README.md): buffered file I/O with the same
matching rules in all four languages, including Rust's default and `small-dev` profiles.

A Rust-only [dependency build snapshot](docs/benchmarks/rust-dependencies/README.md) measures a
JSON-lines CLI using `clap` 4 derive, `serde` 1 derive, `serde_json` 1, and `regex` 1. It
compares Rust `dev` and `small-dev` profiles and provides no cross-language ranking.

![Rust dependency build footprint](docs/benchmarks/rust-dependencies/summary.svg)

Download the HTML report and open it locally; GitHub's file view does not run it.

## Design anchors

- [Architecture](ARCHITECTURE.md)
- [Language and grammar](docs/development/source-of-truth-language-and-backend.md)
- [CLI contract](docs/cli/stable-cli-contract-v0.md)
- [Source and design references](docs/development/source-of-truth.md)
- [Contributing](CONTRIBUTING.md)

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not yet adopted
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
