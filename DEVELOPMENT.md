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
