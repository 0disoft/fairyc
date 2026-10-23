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

Source checking carries these fields as `TransportOnlyArray`, a source helper spanning the
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

## Delivery and evidence

Source typing, backend emission and focused tests are committed at `36dc9dcc`. The expanded shared
probe is committed at `2199a87d` in
[`floating_arrays.go`](../../internal/foreignstructprobe/floating_arrays.go). It adds a 48-byte mixed
record with length-3 `f32` and `f64` arrays, plus length-1 records for both families. Independent C
checks every element with the existing floating helpers across five rotating value classes. The
probe observes whole-field reads, inferred and typed locals, internal calls, nested reads and
references, direct and referenced replacement, and unchanged caller copies after record and C
by-value mutation. No new runtime, workflow or target-maturity surface was added.

Local pre-push validation passed: the full HIR, semantic, MIR, MIR-codec, moduleinterface and packageartifact
suites; focused `TestFloatingArray*` emission for the Linux and Windows profiles, MIR round-trip and
length/boundary rejection including bare foreign arrays and `Option`, `Result`, enum and resource
payloads; the shared probe's Linux-profile `TestForeignCStructResultCallSmokeSourceReachesCEmission`
and `Generate` tests; affected-package vet; and full `cbackend`, `cbackendfixture`, `cbackendnative`
and `foreignstructprobe` tests (38.239s, 2.583s, 3.098s and 0.047s).

A temporary local `TestLocalFloatingArrayNativeCrossing` invoked `runFairyStructResultProbe` and was
removed after execution. Local Windows MSVC passed C11/C17 low/optimized, all four executions, with
five rotating floating-array cases each and expected 42 with empty stdout and stderr. The exact
MSVC version was not rechecked. The full repository suite and the full Windows smoke were not rerun
locally. The [hosted F2-6 receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23)
at `9e1500e1` covers Linux native and hosted execution; the previous `f79e55c6` F2-5 receipt
remains historical for this slice. Phase 2 remains partial and all target maturity flags remain
false.
