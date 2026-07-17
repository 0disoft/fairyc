# Stage 0 Surface and Module Boundary

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Stage 0 needs a parseable, teachable surface that exposes ownership and unsafe boundaries without retaining C declarator ambiguity or adding Rust-sized syntax. The previous design fixed semantics but left grammar, modules, imports, mutability, explicit discard, and restricted user errors undecided.

## Decision

1. Source is UTF-8. Stage 0 identifiers use ASCII letters, digits, and underscore.
2. Statements use explicit semicolons; there is no automatic semicolon insertion.
3. Declarations use `name: Type`, and every local has an initializer.
4. `let` prevents reassignment but may still be consumed. `var` permits reassignment and mutable borrowing.
5. Resource transfer has no source-level `move` keyword. A by-value position consumes a resource and diagnostics name the transfer.
6. A file declares one module. Imports require aliases; wildcard, selective, cyclic, and re-export imports are excluded.
7. Functions declare return types, including `void`. Methods live in `impl Type` and use only `self: &Self`, `self: &mut Self`, or `self: Self`.
8. Stage 0 includes inert `struct`, whole-value `resource`, and restricted user `enum` whose payload fields are inert and copyable. Built-in `Option` and `Result` may contain resources.
9. `if` is an expression. `match` supports only shallow variant patterns. Assignment is a statement, not an expression.
10. `Result` is explicitly discarded with `discard`. Postfix `?` applies only to `Result`.
11. Unsafe operations use `unsafe {}` or `unsafe fn`; raw C declarations use individual `unsafe extern C fn` declarations.
12. Mutable globals, C declarators and casts, implicit truthiness, overloads, default or named arguments, operator overloading, preprocessing, and textual includes are excluded.

## Rationale

The surface keeps one spelling for each operation and lets the parser build syntax without asking name resolution whether an identifier is a type. Removing a `move` keyword avoids ceremony that adds no information beyond a by-value signature. Restricted enums are necessary for typed errors, while resource-bearing general enums would require variant-specific partial-move machinery.

## Consequences

- Module paths and file paths must have a deterministic mapping.
- Some ordinary C declarations become longer but mechanically readable.
- Generic collections, resource-bearing user enums, closures, and rich patterns remain unavailable.
- The syntax contract can be implemented before formatter-grade green trees.

## Revisit conditions

Revisit one exclusion only when representative Stage 0 programs require it, the grammar remains context-free at the syntax layer, and diagnostics and ownership lowering are specified first.
