# ADR 0253: Bound Ada and Pascal Concept Admission

- Status: Accepted candidate classification
- Date: 2026-08-04
- Product source: `docs/product/02-spec.md`
- Roadmap source: `docs/product/01-roadmap.md`
- Depends on: ADR 0136, ADR 0218, ADR 0224

## Context

Ada and Pascal contain useful answers to value-domain mistakes, accidental type interchange, and
control-flow completeness. Their complete syntax, exception, tasking, package, nested-routine, and
representation models do not fit Fairy C's C-shaped source surface, explicit costs, small runtime,
fast-build target, or constrained-C backend.

The useful boundary is therefore the problem each concept solves, not its original spelling. A
roadmap example such as `typedef distinct` or `typedef range` would otherwise be easy to mistake for
accepted syntax even though construction, arithmetic, conversion, ABI, and diagnostic behavior are
still undecided.

## Decision

1. Nominal scalar types are the first new post-Stage 0 candidate. The intended property is that two
   types with the same scalar representation are not implicitly interchangeable. Acceptance
   requires exact rules for alias versus new-type identity, explicit construction and extraction,
   constants, arithmetic, comparisons, pattern binding, public-interface hashing, FFI crossings,
   layout, diagnostics, and source migration. No keyword or declaration spelling is reserved.
2. Constrained scalar ranges are the second candidate. The intended property is that a scalar value
   carries an inclusive valid domain without changing its chosen storage representation. Acceptance
   requires exact initialization, assignment, parameter, return, arithmetic-result, cast, constant,
   runtime-input, overflow, `Result` or trap, ABI, and optimization-independent checking rules.
   Compile-time-provable invalid values must not reach executable lowering. No range syntax or
   failure type is reserved.
3. Exhaustive switching is already part of Fairy C's tagged-value model. Ada or Pascal `case` does
   not justify another construct, wildcard spelling, pattern hierarchy, or hidden default arm.
   Future enum work must preserve explicit completeness or an explicit catch-all decision.
4. Bounded function contracts are a later candidate after the current assertion, effect, borrow,
   and source-location contracts have enough real use. Any accepted form must restrict contract
   expressions to deterministic side-effect-free operations, define evaluation order and failure,
   behave identically across development and optimized builds, and specify result, prior-state,
   unsafe, foreign-call, and function-pointer boundaries. Example names such as `requires`,
   `ensures`, `@requires`, and `@ensures` are not reserved.
5. Ada representation clauses are not imported. Exact foreign layout remains owned by the existing
   target-layout engine, `@repr(C)`, C ABI rules, and generated layout assertions rather than a
   second source-level layout language.
6. Exceptions and stack unwinding, Ada tasking and protected objects, Pascal `with`, capturing
   nested routines, `begin`/`end` syntax, case-insensitive identifiers, implicit conversions, and
   interface/implementation file pairs remain non-goals. They either hide control or ownership,
   duplicate existing module and header-free compilation boundaries, or add runtime and tooling
   complexity disproportionate to the demonstrated problem.
7. This decision classifies candidates only. It changes no grammar, keyword set, diagnostic,
   public-interface encoding, MIR, C transport, SDK, runtime ABI, or target support. The current
   language revision remains `fairyc-language/8`.

## Admission evidence

Before nominal scalar types can be accepted, a feature ADR must provide representative source where
same-representation values are plausibly confused, show why a struct or ordinary function is
materially worse, and define positive, negative, cross-package, FFI, interface-hash, MIR, C-emission,
and migration fixtures.

Before constrained ranges can be accepted, a feature ADR must include boundary, empty-domain,
signedness, overflow, mutation, loop, call, return, constant-folding, runtime-conversion, and foreign
transport cases. It must also decide whether arithmetic preserves the constrained type, returns the
base type, or requires an explicit checked reconstruction; implementation accidents may not choose
that policy.

Before function contracts can be accepted, a feature ADR must demonstrate repeated precondition or
postcondition code that ordinary types and assertions cannot express as clearly. It must define
whether contract failures trap or return data, prove single evaluation, and keep checks independent
of optimization profile unless the compiler proves the same condition statically.

## Consequences

- The post-Stage 0 queue is reopened with two high-priority type-system candidates and one later
  contract candidate, without implying implementation order inside Stage 0.
- Documentation may discuss the intended safety properties, but examples must be labeled
  non-normative until a feature-specific ADR accepts exact syntax and semantics.
- Fairy C borrows Ada's value-domain discipline without adopting Ada or Pascal surface grammar,
  exception handling, tasking, or file-structure machinery.
- `docs/product/02-spec.md` remains unchanged until a candidate becomes accepted language behavior.
