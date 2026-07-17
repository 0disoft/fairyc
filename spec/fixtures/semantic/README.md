# Semantic Fixtures v0

This directory is the machine-readable companion to `docs/language/0007-name-resolution-and-typing-v0.md`, the ownership and borrow contracts under `docs/language/`, and `docs/adr/0010-stage-0-semantic-fixture-contract.md`.

`manifest-v0.json` is the case index. `schema-v0.json` defines its JSON Schema 2020-12 shape. Every case is a parser-valid mini-package with one source root and an explicit, sorted source list.

Entries in the top-level `contracts` array are repository-root-relative paths. Case `sourceRoot` values are manifest-relative, while each `sources` entry and every span source are relative to that case source root.

## Phase boundary

- `resolve` covers module aliases, item coordinates, lexical names, duplicates, and source visibility.
- `type` covers canonical types, literal selection, calls, operators, conditions, assignments, and returns.
- `flow` covers initialization, resource states, loans, borrow origins, regions, must-use values, and cleanup facts.
- A pass case names the last phase that must complete without a semantic diagnostic.
- A fail case names the first phase that owns the listed diagnostic.
- Any `E00xx` parser diagnostic is a fixture setup failure, not an expected semantic result.

## Assertion boundary

Pass assertions identify source text with source-root-relative paths and half-open byte spans. Resolved targets include the declaration span so local or parameter names remain unambiguous without exposing build-local IDs. Assertions may also record canonical types and borrow facts. They never assert AST, HIR, or Flow block IDs.

Fail assertions list the complete deterministic set of independent diagnostics. Diagnostics caused only by an already-poisoned name or type are suppressed as cascades. Full prose is intentionally outside the fixture contract.

## Paths, spans, and order

- Paths use `/`, are relative to `sourceRoot`, and cannot contain `.` or `..` segments.
- Source lists are complete, unique, and sorted by unsigned UTF-8 byte order.
- Spans are zero-based half-open byte ranges over original UTF-8 source bytes.
- Diagnostics sort by path, primary start byte, primary end byte, and code.
- Related spans sort by role, path, start byte, and end byte.

## Adding a case

Add all source files below `cases/pass/<id>/src/` or `cases/fail/<id>/src/`, list them in the manifest, and keep every module declaration equal to its source-root-relative `.fy` path. Prefer one independent failure per fail case. A semantic rule is not fixture-backed until its positive or counterexample evidence is represented without implementation-specific dumps.
