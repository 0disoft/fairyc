# ADR 0329: Nested compact tag fields in private aggregates

- Status: Accepted
- Date: 2026-08-10
- Depends on: ADR 0324, ADR 0326, ADR 0327

## Context

ADR 0327 admits a copy-only `Option` or `Result` whose direct payload is the compact
`Option<i32>` or `Result<i32, i32>` record. Those values already construct, read, switch, and execute
natively, but storing the same outer tag by value in a private struct or restricted user enum still
failed during aggregate layout. That rejection left two independently proven private transports
unable to compose.

## Decision

The constrained C backend admits the ADR 0327 nested compact tags as fields of private inert structs
and restricted user enums. The aggregate layout resolver reuses the exact compact inner record, and
the combined emission graph orders:

1. the fixed compact scalar tag typedef;
2. the copy-only outer tagged record;
3. the containing struct or user enum.

The source program was already valid before this decision. This is an additive private-backend
capability and does not define a public Fairy or C ABI.

## Boundary

Only direct compact `Option<i32>` and `Result<i32, i32>` payload records enter this composition.
General tag-to-tag dependencies, nested resources, arrays, borrows, raw-pointer or C-scalar tagged
payloads, recursion, and every tagged field in `@repr(C)` remain rejected.

## Evidence

The 51st C-emission fixture stores `Option<Option<i32>>`,
`Result<Option<i32>, Result<i32, i32>>`, and `Option<Result<i32, i32>>` across a private struct and
user enum. Interpreter execution returns 42, deterministic C emission requires dependency-ordered
complete definitions, and a dedicated Windows regression compiles and executes the generated Fairy
program with low and optimized MSVC profiles and requires native result 42.

## Compatibility

Language, SDK, runtime ABI, target, MIR, cache, lockfile, public API, and public ABI identities remain
unchanged.
