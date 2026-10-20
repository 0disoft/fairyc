# Product Brief

- Status: Draft
- Owner: UNASSIGNED
- Product source of truth: `docs/product/02-spec.md`

## Problem

C gives programmers direct control and broad ABI reach, but routine ownership, cleanup, bounds, and error-flow obligations remain conventions enforced by people. Rust addresses more of those failures but introduces a larger type-system and build-cost surface than this project wants. Fairy C targets the narrower space between them.

## Product Vision

Fairy C aims to make native systems code significantly easier to maintain and harder to misuse, while ensuring:

- No garbage collector and no mandatory managed runtime
- Predictable data layout and explicit memory allocation
- Direct, zero-overhead C interoperability at declared boundaries
- Fast edit-check-build loops with bounded build cache size
- Consistent release semantics that never silently disable safe-code checks
- Compiler-generated semantic metadata for developer tooling, IDEs, and CI

## Target Audience

- Systems developers writing new native components who want C-level control with modern safety guarantees
- Tooling, compiler, game engine, and native library developers who prioritize explicit cost models
- Maintainers requiring transparent, machine-readable representations of ownership, errors, and safety invariants

## Initial Language Scope

The initial milestone focuses on a compact compiler core with carefully scoped capabilities:

1. Inert value structs
2. Move-only resource types with deterministic cleanup (drop)
3. Non-null references (borrows) alongside explicit raw pointers
4. First-class arena allocation for grouped lifetimes
5. Built-in `Option` and `Result` types for structured error handling
6. Identical failure semantics across both debug and optimized builds

## Non-Goals

Fairy C intentionally omits features that conflict with explicit control and a small runtime footprint, including:
universal automatic memory safety, drop-in C source compatibility, built-in asynchronous runtimes, garbage collection, exceptions, class inheritance, and arbitrary cyclic alias graphs.

## Success Criteria

The Stage 0 milestone succeeds when the compiler reliably builds representative programs that use resources, arenas, structured error handling, and C FFI, and produces deterministic diagnostics and consistent cross-platform behavior. Broad ecosystem adoption and self-hosting are reserved for future phases.
