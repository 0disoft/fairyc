# Stage 0 Semantic Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

The parser contract proves that source can reach AST lowering, but it does not prove what a name denotes, which target type an expression receives, or whether a loan is legal. If the first compiler invents those test outputs while being implemented, build-local HIR IDs, diagnostic cascades, and incidental traversal order can become accidental language contracts.

## Decision

1. `spec/fixtures/semantic/manifest-v0.json` is the machine-readable Stage 0 acceptance index for name resolution, type checking, and Flow analysis.
2. Every semantic case is a parser-valid mini-package with one explicit source root and a sorted, complete list of `.fy` source files.
3. Analysis phases are ordered `resolve`, `type`, then `flow`. A pass case names the phase it completes through. A fail case names the first phase that owns its primary rejection.
4. Successful assertions use source spans plus canonical semantic facts: resolved declaration coordinates, canonical Fairy C types, and hidden borrow origins. Build-local AST, HIR, Flow, and symbol IDs are not fixture contracts.
5. Compile-fail assertions use stable diagnostic codes, a primary source-byte span, and role-labelled related spans. Full prose, terminal layout, and repair wording are not frozen.
6. Diagnostics are compared as a deterministic complete set after suppressing cascades that exist only because an earlier symbol or type is poisoned. An unlisted independent diagnostic fails the case.
7. Canonical ordering is source path by unsigned UTF-8 bytes, then `startByte`, `endByte`, and diagnostic code. Source paths use `/` and are relative to the case source root.
8. The fixture set covers qualified import resolution, explicit typing, local and branch-reachable
   last-use loans, outside-loop safe-borrow reassignment, reachable origin joins, single-parameter
   borrow return, unresolved and duplicate names, type mismatch, non-boolean conditions,
   active-loan conflicts, escaping local borrows, and multiple-origin returns.
9. Resource-state, region, `Result`, layout, MIR, and runtime assertions may reuse the mini-package and span conventions, but they require their own fixtures or a versioned extension rather than silently broadening this v0 schema.

## Rationale

Small semantic facts survive compiler refactoring. Whole-HIR snapshots do not. Exact byte spans make diagnostics reproducible across Windows and Linux, while role-labelled related spans prove that an error explains the declaration, loan, origin, or last use that caused it. Separating the semantic manifest from parser fixtures keeps syntax-pass from being confused with type-safe or borrow-safe.

## Consequences

- The compiler test runner must parse every listed source without `E00xx` before semantic assertions are evaluated.
- A resolver or type checker may use any internal arena and ID layout as long as the canonical assertions remain equal.
- `internal/semanticfixture` executes every `resolve`, `type`, and local `flow` case. The runner
  compares stable semantic facts rather than build-local IDs and never promotes parser validity or
  unsupported syntax into a semantic pass.
- Adding a stable semantic diagnostic requires the catalog, owning language contract, schema-compatible fixture, and case source to change together.

## Revisit conditions

Version the schema when a required assertion cannot be represented without implementation-specific dumps, when multi-package dependency resolution is introduced, or when diagnostic JSON itself becomes a released public contract.
