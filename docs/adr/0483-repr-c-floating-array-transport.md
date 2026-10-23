# ADR 0483: Repr(C) floating-array transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0482's array transport, ADR 0479's floating field transport

## Decision

Admit fixed-array fields of `f32` and `f64` in nonempty acyclic `@repr(C)` records, with lengths
`1..=4096`. They are actual named C array members whose element types are `float` and `double`,
respectively. The target layout engine owns size, alignment and offsets; generated assertions check
those facts against the C providers. The fields remain distinct Fairy array types rather than
pointers or ordinary executable array storage.

Source checking will carry these fields as `TransportOnlyArray`, a source helper spanning the
existing `IntegerTransportArray` families and the new floating-array families. Raw C members and
whole-array values bridge by element copies. C-produced values support whole-field reads, inferred
and explicitly typed array locals, internal by-value array arguments and results, record copies,
explicit reconstruction and direct whole-field replacement through a mutable record or record
reference. Nested record reads and copies retain value semantics. This admission adds no nested
array field-store path.

Floating literals, arithmetic, comparisons and casts; indexing, element stores, iteration and
views; bare foreign array parameters and results; pointer elements or pointer-based array transport;
`c::float` or `c::double` arrays; direct arrays in private records; resource, enum, `Option` or
`Result` payload arrays; and record-element arrays remain outside executable admission. Existing
`f32`/`f64` scalar-field rules, integer-array transport and admitted pointer forms are otherwise
unchanged.

There is no new opcode, cache schema, SDK export, CLI contract, workflow or target promotion.
Phase 2 remains partial and all target maturity flags remain false.

## Validation

Source typing, backend emission, focused compiler checks and native C probe execution for this
admission are pending. This documentation slice makes no delivery or validation claim. Existing
ADR 0479 floating-field evidence and ADR 0482 integer-array evidence do not attest floating arrays.
