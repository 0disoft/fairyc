# ADR 0325: Copy-scalar tagged fields in private aggregates

- Status: Accepted
- Date: 2026-08-10
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0118, ADR 0324

## Decision

1. Admit private `Option<T>` and `Result<T, E>` fields in inert structs and restricted user enums
   when every payload is one canonical Fairy copy scalar: `bool`, `i32`, `u8`, `u32`, or `usize`.
2. Compute those direct tagged layouts without an aggregate dependency, register tagged types named
   only by aggregate definitions, and emit their private typedefs before every containing aggregate.
3. Keep aggregate-bearing, fixed-array-bearing, borrow-bearing, raw-pointer/C-scalar-bearing, and
   resource-bearing tags on their existing later transport path; they are not admitted as aggregate
   fields by this decision.
4. Keep all built-in tagged fields out of `@repr(C)` records. Recursive dependencies and a public
   tagged ABI remain closed.
5. Keep language, SDK, runtime ABI, target schema, MIR, cache, lockfile, and public API identities
   unchanged. This closes a private backend gap for already-valid Stage 0 source.

## Evidence

- The 48th constrained C-emission fixture embeds `Option<u32>`, `Result<u8, usize>`, and
  `Option<usize>` across a private struct and user enum, then constructs, reads, switches, compares,
  and returns 42.
- C-backend unit regressions require every direct copy-scalar tagged typedef before its first
  by-value aggregate field and retain fail-closed `Option<Struct>` and `@repr(C)` boundaries.
- A Windows-amd64 Go regression invokes the real `experimental run-host` path at low and optimized
  profiles. Both generated-Fairy MSVC executables return 42 with empty stderr.

## Consequences

Private aggregate models may use the full existing copy-scalar tagged vocabulary instead of only
the compact `i32` special cases. Aggregate-bearing tags still require a unified dependency graph
before they can become fields, and no generated C spelling or layout becomes public ABI.
