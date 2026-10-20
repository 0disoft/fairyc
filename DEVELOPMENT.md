# Development Guide

This document covers development guidelines, local build instructions, and architectural boundaries for the Go Stage 0 compiler.

## Reference Specifications

- Architecture Decisions: `docs/adr/0001-initial-architecture-boundaries.md`
- Parser Specification: `docs/adr/0009-stage-0-grammar-and-parser-contract.md`
- Product Specification: `docs/product/02-spec.md`

## Toolchain Guidelines

- `go.mod` specifies the required Go toolchain version.
- Dependencies (including the TOML parser) are managed via `go.mod` and `go.sum`.
- Local Go build caches are placed under `.cache/go-*` (ignored in `.gitignore`). Cache contents are temporary and not checked into source control.

## Building the Windows Compiler Binary

Compile the Stage 0 compiler into a standalone Windows binary (`fairyc.exe`) to avoid the overhead of `go run` during local testing.

From the repository root in PowerShell:

```powershell
New-Item -ItemType Directory -Force build | Out-Null
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
.\build\fairyc.exe experimental run-target examples/windows_quickstart/main.fy --target x86_64-windows-msvc --profile optimized
.\build\fairyc.exe experimental exec-target examples/windows_output/main.fy --source examples/windows_output/windows_console.fy --target x86_64-windows-msvc --profile optimized

Push-Location examples/workspace_quickstart
..\..\build\fairyc.exe experimental check-workspace
..\..\build\fairyc.exe experimental run-workspace --locked --target x86_64-windows-msvc --profile optimized
Pop-Location
```

### Notes on Local Execution
- The commands above build the compiler binary, compile sample `.fy` sources through MSVC into native PE executables, and verify workspace commands.
- `run-target` compiles and executes the program directly (outputting `native result: 42`).
- `exec-target` forwards standard process output (printing `Fairy C says hi` and returning exit code 42).
- To clean up local build artifacts, remove the `build/` directory and any generated `fairy.lock` files.

## Package Architecture Boundaries

- `internal/source`: Manages immutable source bytes, module identity, and parsing budget.
- `internal/lexer`: Handles lossless tokenization, whitespace/comments trivia, and literal processing.
- `internal/syntax`: Concrete syntax tree (CST) arena, recursive-descent parsing, Pratt expressions, error recovery, and syntax diagnostics.
- `internal/ast`: Lowers diagnostic-free CST into abstract syntax trees for semantic analysis.
- `internal/parserfixture`: Validates syntax behavior against `spec/fixtures/parser/manifest-v0.json`.

## PR Review Checklist

Pull requests should follow these guidelines:

- **Clear Requirements**: Feature additions must map to an approved design or ADR.
- **Strict Validation**: Do not weaken existing tests or skip error diagnostics.
- **Clean Source of Truth**: Never commit generated binaries, local caches, or temporary build outputs.
- **Fail-Closed Parser**: The parser must reject invalid syntax with appropriate diagnostics rather than producing partial ASTs.
