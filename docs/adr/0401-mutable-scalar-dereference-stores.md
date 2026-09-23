# ADR 0401: Mutable scalar dereference stores

- Status: Accepted
- Date: 2026-09-24
- Owning contract: `docs/language/0002-borrows-and-return-origins.md`
- Depends on: ADRs 0394 and 0400

## Decision

Plain statement assignment `*slot = value` writes through a named `ref mut` to
`bool`, `u8`, `i32`, `u32`, or `usize`. The right-hand side has the exact scalar
type and is evaluated once after obtaining the destination reference. The
exclusive loan remains live through that evaluation: a competing borrow, direct
owner read, or owner move is rejected. Reading through the same reference is
allowed. This makes `std::vec::at_mut<i32>` useful for scalar updates.

The private MIR `deref-store` operation has no value destination. Its verifier
requires a mutable scalar reference and an exactly typed value. The interpreter
updates the original local referent across calls; the constrained C backend
emits one typed pointer write. Neither path adds allocation, runtime metadata,
or a helper call.

Aggregate replacement, compound dereference assignment, temporary reference
targets, and mutable field-address projection remain unsupported. The grammar
already parses assignment and unary `*`, so language `/17`, SDK
`0.0.0-stage0.63`, and runtime revision 0 remain unchanged. This completes an
executable subset of existing syntax without changing a public ABI.

## Validation

Semantic cases cover direct locals, parameters, reference reads, conflicting
aliases and owner reads, and unsupported targets. MIR mutation tests reject
shared or mistyped references and malformed stores. A private artifact round
trip executes the operation in the interpreter. The Windows `Vec<i32>` source
probe writes and reads the same element under default and optimized MSVC
profiles, and rejects an overlapping alias. Generated C contains a direct
pointer write. These checks establish behavior, not a measured compile-time or
binary-size regression bound.
