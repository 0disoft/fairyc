# ADR 0033: Transport flat i32 resources through constrained C

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- MIR source: `docs/mir/mir-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`
- Implementation source: `internal/cbackend/`

## Context

Executable MIR now owns ordered resource definitions and rejects nonempty user `drop` bodies before
backend handoff. The interpreter can construct, move, read, reassign, and drop those values, but the
C backend still rejected every resource. That left ordinary whole-resource programs unable to reach
the existing Linux build path even when their storage was only a fixed list of `i32` fields and
cleanup had no user-visible effect.

Using a Go struct or a source-named C struct would be the wrong shortcut. Go host layout is not the
target contract, source identifiers must not become generated-C symbols, and a successful C compile
alone would not prove that the C compiler selected the layout computed for
`x86_64-linux-gnu`.

## Decision

1. The constrained C backend accepts a resource definition only when it has at least one field,
   every field is exactly `i32`, and its verified drop classification is `empty`. Every resource
   definition in the emitted program must satisfy this gate.
2. Definitions are sorted by canonical Fairy type and receive translation-unit-private numeric C
   names such as `fy_resource_0`. Fields retain declaration order but receive numeric members such
   as `f_0`; no source resource or field name is emitted.
3. `internal/layout`, bound to the declared `x86_64-linux-gnu` `TargetSpec`, computes size, natural
   alignment, and field offsets. Generated C11 includes `_Static_assert` checks for `sizeof`,
   `_Alignof`, and every `offsetof`, so a disagreeing C toolchain fails before producing an
   executable.
4. Direct flat resources may be locals, parameters, internal function results, call arguments, and
   return values. The transport uses ordinary private C struct assignment and by-value calls. This
   does not define a public Fairy ABI or an `@repr(C)` resource layout.
5. MIR resource construction becomes one ordered compound literal and field reads select the
   verified numeric member. `initialize` and ordinary temporary assignment become C struct
   assignment. Whole-root `move` and empty `drop` remain explicit verified MIR events but require no
   runtime cleanup call; generated `(void)` uses keep their discharge points visible without
   inventing user drop effects.
6. Drop flags, partial construction, empty resources, non-`i32` or nested fields, nonempty drop
   bodies, and resource-bearing `Option` or `Result` values remain rejected. The backend does not
   recursively guess an aggregate representation.
7. The C-emission v0 manifest grows from eight to eleven cases: local construction and field read,
   by-value call move, and live reassignment. The Linux matrix therefore plans 44 process-status
   comparisons across Clang/GCC and low/optimized profiles. Its build smoke also carries one flat
   resource through the product CLI, without increasing the four compiler/profile smoke runs.
8. The MIR interpreter remains the semantic oracle. Process status can compare the final return or
   trap but cannot observe an empty drop event, so this slice does not claim native cleanup-trace,
   sanitizer, public ABI, or user drop-body conformance.
9. `fairyc.target-spec/v0` initially reports the implemented subset as
   `scalar-and-flat-i32-resource-c11`; ADR 0034 later expands it to
   `scalar-and-flat-i32-resource-tagged-c11` while all target maturity booleans remain false.

## Consequences

- A useful whole-resource subset now reaches raw C emission and the eligible Linux build path.
- Layout disagreement becomes a compile-time failure at the generated-C boundary instead of a
  silent host-dependent ABI accident.
- Source resource names remain absent from generated identifiers and private ABI.
- Native proof remains unobserved on the current Windows host, and broader resource cleanup still
  requires a new observable contract rather than another struct special case.

## Rejected alternatives

- Use Go struct layout: makes the compiler host an accidental target specification.
- Emit source names into C: creates collision, escaping, and accidental ABI surfaces.
- Treat every resource recursively as a C struct: admits layout and cleanup behavior with no
  supported field-type, drop, or tagged-payload contract.
- Delete move and drop operations before emission: erases the verified ownership boundary and makes
  later cleanup work reconstruct semantics from C assignments.
