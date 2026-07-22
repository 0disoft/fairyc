# Development

- Status: Draft
- Owner: UNASSIGNED

## Purpose

This document records the executable development boundary for the Go Stage 0 compiler.

## Source of Truth

- Product decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Technical owner: UNASSIGNED
- Parser contract: `docs/adr/0009-stage-0-grammar-and-parser-contract.md`

## Toolchain boundary

- `go.mod` declares the current development toolchain language level. It is not yet a released
  minimum supported Go version.
- The implementation uses only the Go standard library in this slice. Verification is configured
  to run with module and checksum network access disabled.
- Local Go caches belong under ignored `.cache/go-*` paths. Cache contents are never test evidence
  or source truth.
- In the containing Mustflow workspace, use `fairyc_go_format`, `fairyc_go_vet`,
  `fairyc_go_build`, `fairyc_go_test_parser`, and `fairyc_go_test`. Their exact argv, writes, and
  timeouts are owned by the workspace command contract.

## Local Windows compiler executable

The Stage 0 compiler can be built as an ordinary local Windows executable. This removes `go run`
from repeated compiler invocations; it does not make the compiler self-hosted, released, installed,
or redistributable. From the repository root in PowerShell:

```powershell
New-Item -ItemType Directory -Force build | Out-Null
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
.\build\fairyc.exe experimental run-target examples/windows_quickstart/main.fy --target x86_64-windows-msvc --profile optimized
.\build\fairyc.exe experimental exec-target examples/windows_output/main.fy --source examples/windows_output/windows_console.fy --target x86_64-windows-msvc --profile optimized
Push-Location examples/workspace_quickstart
..\..\build\fairyc.exe experimental check-workspace
..\..\build\fairyc.exe experimental run-workspace --locked --target x86_64-windows-msvc --profile optimized
@(
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":1}'
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":2}'
) | ..\..\build\fairyc.exe experimental check-workspace-session --locked --target x86_64-windows-msvc --max-requests 2 --cache-max-entries 8 --cache-max-bytes 33554432 --cache-max-artifact-bytes 8388608
Pop-Location
```

The first command creates only the ignored output directory. The wrapper keeps Go caches under the
ignored `.cache/go-*` roots and disables module and checksum network access. The expected final
`run-target` receipt starts with `native result: 42`; `exec-target` prints `Fairy C says hi` and
returns the FY program's status 42. The workspace check writes `fairy.lock`, and the locked run
receipt starts with `native workspace result: 42`. The two session response lines report
`fallback` and then `all-hit`; that session performs checks only and emits no PE. In every native
case the FY program is compiled into and executed as an MSVC PE. WSL is neither installed nor
invoked by this path. Delete
`build/fairyc.exe`, the example `fairy.lock`, and any example `build/` output to undo local outputs.

The configured Windows smoke independently builds `fairyc.exe` into a temporary directory, invokes
its help surface, runs the checked-in quickstart through `run-target`, requires `exec-target` to
forward the checked-in output example and return process status 42, runs the checked-in local
workspace through an exact existing lock, and sends eight editor-shaped checks through one
foreground session while reading Windows process-memory peaks. It removes the compiler and target
images. That proves the standalone process paths and process-local reuse without treating an
ignored local build artifact as source truth or one local measurement as a budget.

## Implementation boundaries

- `internal/source` owns immutable source bytes, module identity, and the shared parse budget.
- `internal/lexer` owns lossless tokens, trivia, invalid-input provenance, and literal cooking.
- `internal/syntax` owns compact CST arenas, recursive-descent productions, Pratt expressions,
  recovery, and deterministic syntax diagnostics.
- `internal/ast` lowers only diagnostic-free syntax and exposes implementation-private normalized
  assertions for parser fixtures.
- `internal/parserfixture` executes `spec/fixtures/parser/manifest-v0.json` without weakening or
  rewriting the fixture contract.

## Review Blockers

- The change invents a product domain without a source.
- The change weakens validation or skips required evidence.
- The change relies on generated, cache, or build output as source truth.
- The parser accepts malformed input as a successful AST or emits diagnostics not listed by a
  failing fixture.
