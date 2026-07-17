# Testing Standard

- Status: Draft
- Owner: UNASSIGNED
- Semantic sources: `docs/language/`, `docs/mir/`, `docs/abi/`, `docs/runtime/`, `docs/compatibility/`, `docs/diagnostics/catalog-v0.md`, `docs/atlas/`
- Merge-blocking validation: `VALIDATION.md`

No implementation or fixture directories exist yet. This document defines the evidence classes that a future implementation must create; it does not claim any test currently passes.

## Oracle order

1. Product specification and accepted ADRs define the intended boundary.
2. Versioned language and MIR fixtures become the executable contract.
3. The MIR verifier rejects structurally invalid compiler output.
4. The MIR interpreter provides the first executable semantics oracle.
5. Generated C is compared against the interpreter under the C compilers and profiles declared by the target fixture.
6. Later backends are compared against the same interpreter and C evidence rather than defining new behavior.

## Fixture classes

| Class | Proves |
| --- | --- |
| Parse pass and fail | Grammar acceptance, recovery boundary, source spans, and deterministic diagnostics |
| Module pass and fail | File-to-module identity, explicit aliases, forbidden import forms, cycles, and source visibility |
| Type and initialization fail | Inert/resource separation and initialized-local rules |
| Resource flow pass and fail | Whole-root moves, branch and loop states, consuming close, and no double drop |
| Borrow-origin pass and fail | Local last-use loans, conflicts, single-origin return, and storage bans |
| Region pass and fail | Escape prevention, capability storage ban, and no `NeedsDrop` allocation |
| Result pass and fail | Must-use behavior, `?` lowering, payload moves, and cleanup on propagation |
| Built-in tagged resources | `Ok`, `Err`, `Some`, discard, match, propagation, early return, and nested active-payload drop traces |
| MIR verifier mutation | Every `Vxxxx` invariant rejects a deliberately malformed graph |
| MIR interpreter run and trap | Scalar results, branches, moves, drop traces, region reset, and trap categories |
| Drop elaboration | Partial construction, reassignment, early exits, loops, and cleanup-suffix merging |
| C differential | Interpreter parity for output, status, trap category, and drop trace across declared compilers and profiles |
| C safety | Overflow, division, shifts, evaluation order, aliasing, alignment, inactive payload, and sanitizer-assisted checks |
| ABI | Target layout, calls, callbacks, raw bindings, and boundary shims for candidate target profiles |
| Runtime and stdlib | Zero-size normalization, allocation failure, reserve rollback, UTF-8 invariants, partial I/O, Linux close consumption, and retained-pointer wrappers |
| Compatibility | Exact pre-1 matching, cache misses, runtime mismatch, diagnostic identity, stale maps, target mismatch, and relation-specific completeness |
| Atlas schema | Deterministic identity, record compatibility, unknown edges, provenance, and whole-package rewrite |
| Atlas context | Budgets, stable ordering, omissions, stale source, text caps, and prompt-injection isolation |

## Diagnostic assertions

Compile-fail fixtures assert the stable diagnostic code, primary failure category, and required related spans. They do not freeze full prose unless wording itself is the contract. A single malformed input may produce recovery diagnostics, but the fixture names which diagnostic is primary.

## Differential matrix

The first planned matrix targets `x86_64-linux-gnu`, uses the MIR interpreter plus generated C, and compares Clang with GCC at low and optimized profiles. Exact compiler and glibc baselines remain UNDECIDED. Fairy-to-C and C-to-Fairy fixture directions are both mandatory. A new target or backend cannot replace the interpreter and reference-C oracles merely because its outputs agree with itself.

ABI fixtures cover scalar and pointer layouts, nested and padded `@repr(C)` structs, fixed arrays, explicit C enums, register exhaustion, mixed integer and floating arguments, narrow extension rules, by-value aggregate calls and returns, hidden return storage, null raw pointers, and the permitted synchronous callback. Every excluded C ABI feature has a negative fixture.

## Fuzzing and sanitizers

Parser, type, Flow CFG, MIR verifier, and schema fuzzing are later configured validation surfaces. Generated-C sanitizer runs are backend evidence, not proof of language semantics. Fuzz duration, corpus size, and coverage thresholds remain uncommitted until a reproducible harness exists.

## Review blockers

- A semantic rule has no positive or negative observable fixture class.
- A backend test compares only generated text instead of executable behavior where execution is available.
- A diagnostic code changes meaning without the catalog and compile-fail fixtures changing together.
- A trap, move, or cleanup result differs by optimization profile.
- A foreign nonlocal-control test crosses a Fairy C frame instead of using a boundary shim.
- An Atlas test treats missing or opaque edges as a complete graph.
- A target is called conformant or supported while its declared fixture subset contains a skip.
- A runtime wrapper normalizes Linux-specific behavior and then documents it as portable POSIX behavior.
- Exact test counts, durations, or coverage numbers are promised without an executable harness and source of truth.
