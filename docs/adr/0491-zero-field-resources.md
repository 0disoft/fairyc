# ADR 0491: Zero-field resources

- Status: Accepted for prototype
- Date: 2026-11-01
- Language owner: [Values and resources](../language/0001-values-and-resources.md)

## Decision

Admit `resource Token { cleanup {} }` and zero-field resources with executable
nonempty cleanup. Construction is `Token {}` in the declaring module. A complete
root is owned even with no source fields: local/parameter/return moves, borrows,
reassignment, nested direct resource fields and active Option/Result payloads use
the existing ownership rules. An empty cleanup body still leaves a move-only resource.
Source branch-dependent roots remain rejected; no conditional ownership flag is added.

MIR construction accepts zero operands only when the resource definition also has
zero fields. Missing definitions, extra fields, missing initializers for nonempty
resources, invalid cleanup and forged ownership still fail verification. The
interpreter retains a distinct resource state and invokes cleanup exactly once.

ISO C has no empty structs. The private C representation uses one initialized u8
member, size/alignment 1, with the normal layout assertions. This byte is not a
source field, discriminant, drop flag, public layout or user-addressable property.
Its storage is preserved inside direct resource parents and tagged payloads.
Empty initialization emits `{ 0 }`; no extension, uninitialized marker or erased
cleanup responsibility is used. StorageLayout reports this private representation.

## Validation

Focused compiler checks cover moves through returns and parameters, nested direct
resources, inactive tags, exact interpreter drop order, MIR codec round trips,
both-target emission and private storage size/alignment. Forged construction/drop
mutations reject. The zero-field native probe observes the exact cleanup sequence
`1,9,1,1` through independent C functions under Windows MSVC low/optimized.
Linux native execution remains deferred to the batched queue. Nested owning tags
are the next slice and are not admitted by this decision alone.

No source grammar, foreign ABI, SDK/CLI API, runtime, opcode/codec envelope,
dependency, DB, CI/runner or repository hygiene policy changes.
