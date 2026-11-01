# ADR 0492: Nested owning Option and Result cleanup

- Status: Accepted for prototype
- Date: 2026-11-01
- Owners: [Values and resources](../language/0001-values-and-resources.md), [MIR v0](../mir/mir-v0.md)
- Builds on: [ADR 0491](0491-zero-field-resources.md)

## Decision

Admit compiler-private nested `Option` and `Result` transports containing resources,
including zero-field resources and existing direct resource parents. The owning path
has at most eight nested Option/Result layers. Existing inert tagged payload families
keep their own admission rules; this decision does not add floating tagged leaves,
owning user enums, owning arrays, resource fields containing tags or public C layout.
Examples include `Option<Option<Token>>`, `Option<Result<Box, Empty>>` and
`Result<Option<Box>, Result<Box, Empty>>`. Every branch must have an admitted representation.
Beyond eight owning layers, source/MIR can remain valid but this C backend rejects emission.

Construction initializes the complete value; moves through locals, parameters and returns
transfer the root cleanup obligation. Matching transfers an owned inner tag as a whole,
with exact payload identity and MovePayload metadata. Reassignment evaluates the new value
before destroying the old live value. `none` and inactive Result branches acquire no active
resource cleanup. Source partial moves and use of a branch-dependent MaybeLive root stay
rejected. No compiler-owned conditional root flag is exposed or newly added.

Candidate preparation propagates recursive ownership and bounds owning depth. Layout and
C definitions follow payload dependencies while retaining stable numeric type IDs. Every
required Option/Result discriminant is declared even when its branch is never constructed.
Drop glue descends through active discriminants only, then runs the existing direct resource
drop body before reverse field cleanup. Parent mutations still propagate to child cleanup.
Instrumentation follows the same active paths; it reports resource leaves, not tag wrappers.

MIR verification checks extracted payload type against the inferred container, and requires
MovePayload exactly when that payload recursively owns a resource. Clearing ownership or
retyping an extraction fails verification and emission. Per-function memoized type facts
avoid exponential visits to shared inert Result graphs. Wire opcodes and envelope stay unchanged.

## Evidence and limits

[Windows execution receipt](../ops/windows-resource-forms-20261101.md) records MSVC C11/C17
low/optimized, exact independent C cleanup observations, interpreter/native trace agreement,
MIR codec, stable package memory/disk cache restoration, depth and negative checks. Both
Linux-profile production and instrumentation emission pass locally. Native Linux including
ASan/UBSan remains in the batched queue; local emission is not native execution evidence.

Private transport extension only: grammar, foreign ABI, SDK/CLI API, runtime, dependency,
DB, CI/runner and repository hygiene configuration are unchanged. No push or hosted dispatch.
