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

## Delivery and evidence

Source typing is committed at `fe4beed8`; backend and native proof are committed at `af181d65`.
`internal/foreignstructprobe/integer_arrays.go` extends
the shared probe. Independent C supplies length-3 arrays for all five families in a 72-byte mixed
record and nested envelope, then observes internal array argument/result copies, reads through a
borrowed nested record, and direct whole-field replacement through mutable records and references.
It checks reconstruction, C by-value mutation and replacement while saved originals stay unchanged.
Values cover zero, signed minima/maxima, unsigned maxima, `u64` above signed 64-bit range and
ordinary values. Individual 1/2/8-byte records with `[1]` arrays cover every family. Generated
assertions cover target size, alignment, offsets and signedness.

Local validation passed: focused compiler `TestIntegerArray*` on both target profiles, boundaries
and lengths 1/4096; full `cbackend` (65.557s), `cbackendfixture`, `cbackendnative` and
`foreignstructprobe`; affected backend/probe vet; HIR, semantic, MIR, codec, moduleinterface and
packageartifact suites; and Linux-profile `TestForeignCStructResultCallSmokeSourceReachesCEmission`.
MSVC 14.51.36231 passed C11/C17 low/optimized, all four executions, expected 42 and empty output,
alongside prior probe checks.

Hosted GCC/Clang native evidence remains pending; the `82b760cb` hosted receipt covers F1–F3 only.
The full repository suite, full Windows smoke and local Linux native run were not executed. Scope
remains whole-array and record value transport; all element and payload restrictions in the decision
stand. Phase 2 remains partial and all maturity flags remain unchanged.
