# Product Brief

- Status: Draft
- Owner: UNASSIGNED
- Product source of truth: `docs/product/02-spec.md`

## Problem

C gives programmers direct control and broad ABI reach, but routine ownership, cleanup, bounds, and error-flow obligations remain conventions enforced by people. Rust addresses more of those failures but introduces a larger type-system and build-cost surface than this project wants. Fairy C targets the narrower space between them.

## Product promise

Fairy C should make ordinary native systems code easier to review and harder to misuse while retaining:

- no garbage collector and no mandatory managed runtime;
- predictable data layout and explicit allocation;
- first-class C interoperability at declared boundaries;
- fast edit-check-build loops and bounded build-cache growth;
- release semantics that do not silently weaken safe-code behavior;
- compiler-generated semantic maps for humans, IDEs, CI, and LLM tools.

## Primary users

- C developers starting new systems components rather than mechanically translating arbitrary legacy C;
- small compiler, tooling, game, server-native, and native-library teams that value explicit cost models;
- maintainers who need stable machine-readable ownership, error, unsafe, and test relationships.

## Product wedge

The initial wedge is a small command-line compiler with a deliberately restricted language core:

1. inert value structs;
2. move-only resource types with deterministic drop;
3. non-null borrows and explicit raw pointers;
4. first-class arenas for grouped lifetimes;
5. separate `Option` and `Result` types;
6. identical safe-code failure semantics across optimization levels.

## Non-goals

Fairy C does not promise universal memory safety, automatic source compatibility with C, a built-in async runtime, garbage collection, exceptions, class inheritance, arbitrary safe alias graphs, Stage 0 embedded or freestanding support, or performance superiority over every expert-written C program.

## Success test

The project succeeds when a small Stage 0 compiler can compile representative resource, arena, error, bounds, and C-FFI programs with deterministic diagnostics and backend-independent behavior. Popularity, benchmark rankings, and self-hosting are not Stage 0 success criteria.
