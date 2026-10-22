# ADR 0482: Repr(C) integer-array transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0477's integer array fields, ADR 0480's integer field transport

## Decision

Admit fixed-array fields of `i8`, `i16`, `u16`, `u64` and `isize` in nonempty acyclic
`@repr(C)` records, with lengths `1..=4096`. They are actual named C array members whose
element types are `int8_t`, `int16_t`, `uint16_t`, `uint64_t` and `intptr_t`, respectively.
`isize` follows target pointer width. The target layout engine owns size, alignment and offsets;
generated assertions check those facts and the C providers. The fields remain distinct Fairy
array types rather than pointers or ordinary executable array storage.

C-produced values support whole-field reads, inferred and explicitly typed array locals, internal
by-value array arguments and results, record copies, explicit reconstruction and direct whole-field
replacement through a mutable record or record reference. Nested record reads and copies retain
value semantics. This admission adds no nested array field-store path.

Source checking carries these fields as `IntegerTransportArray`, a new source helper that
distinguishes them from ordinary executable arrays. Raw C members and whole-array values bridge by
element copies. Private structs with the new direct arrays, element literals, indexing, stores,
views or iteration, scalar arithmetic, comparisons or casts, bare foreign array parameters/results,
new scalar pointers, and direct arrays in resource, enum or tagged payloads remain outside
executable admission. The existing `i8`, `i16`, `u16`, `u64` and `isize` transport scalars remain
under ADR 0480's restrictions. Existing arrays and scalar transport are otherwise unchanged.

There is no new opcode, cache schema, SDK export, CLI contract, workflow or target promotion.
ADR 0480 remains the existing scalar decision and is unchanged; this ADR only adds the
corresponding array fields.

## Planned evidence

The existing shared `internal/foreignstructprobe` source will supply and observe every element
from independent C declarations. Coverage includes zero, signed minima and maxima, unsigned
maxima, `u64` values above the signed 64-bit range, and callee copy mutation with the caller's
originals unchanged. Target-layout assertions will cover the array fields and their C providers.

The implementation type slice has passed full HIR, semantic, MIR, MIR codec, moduleinterface and
packageartifact tests, along with focused compiler MIR round-trip and boundary tests. Native
evidence for this admission remains planned and pending.
