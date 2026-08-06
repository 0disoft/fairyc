# ADR 0317: Public PathKind predicates

- Status: Accepted
- Date: 2026-08-06
- SDK: `0.0.0-stage0.46`
- Depends on: ADR 0281, ADR 0293

## Decision

1. Add effect-free `std::fs::path_kind_is_file(PathKind) -> bool` and
   `std::fs::path_kind_is_directory(PathKind) -> bool`.
2. Both predicates are exhaustive: `.Other` is false for both, and exactly the named variant is true.
3. Preserve `path_kind` snapshot/error behavior and every target C symbol and runtime ABI.
4. Advance the compiler-bundled SDK to `0.0.0-stage0.46`; language remains
   `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

## Evidence

The SDK contract test requires both public functions. The checked-in Windows std::fs quickstart
uses both predicates for an ordinary file and directory, then low and optimized generated-Fairy
executions retain the exact missing-path error and cleanup behavior.

## Consequences

Callers no longer need private exhaustive switches for the two common PathKind questions. The
predicates remain observations, not race-free authorization or existence prechecks.
