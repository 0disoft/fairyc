# ADR 0479: Repr(C) floating field transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)

## Decision

Admit distinct Fairy `f32` and `f64` value types for floating fields in nonempty acyclic repr(C)
records. A record received from C can supply typed local values, internal value arguments/results,
and explicit construction of another record. Whole-record copies retain value semantics. A nested
repr(C) record may contain these fields. Exact source identities remain distinct from `c::float`
and `c::double`; there is no implicit conversion between them or between floating widths.

This slice supplies value transport, not floating literals, arithmetic, comparisons, casts,
arrays/views, direct floating-field stores, resource, tagged or private enum payload storage. Direct float
fields in private structs and standalone foreign float signatures also remain closed. The pure
interpreter continues to reject foreign calls rather than fabricate a floating producer.

Target layout owns sizes and alignments. C transport uses `float` and `double`, with generated
checks for size/alignment and binary32/binary64 precision and exponent limits. This does not
introduce fast-math, floating-environment controls or a promise about NaN payload bits. Ordinary
record/value copies must preserve signed zero and infinities and retain quiet NaNs as NaNs on the
tested target toolchains; signaling NaN behavior remains outside this admission.

The builtin identities travel through verified MIR, the existing MIR codec, public-interface
hashing and stable package contexts/rebinding. There is no new opcode or serialized field, SDK
export, language revision, workflow or target maturity promotion.

## Delivery

Commit `db47dc2f` adds type identities and package preservation. The next implementation connects
target field layout, scalar value emission and the existing shared native record probe. Emission
includes `<float.h>` only when a floating Fairy value is used and rejects its standard macro names
on public C records, members and foreign symbols before expansion can corrupt emitted declarations.

The shared probe checks a 4-byte scalar record, a 16-byte double pair, a 16-byte mixed integer/float
record and a 24-byte record. Each configured execution checks finite values, negative zero,
infinities, quiet NaNs and the smallest positive subnormals. Independent C code observes every
floating field, including after Fairy field reads, internal calls, reconstruction and C by-value
mutation; caller copies remain unchanged. NaN checks require quiet NaN classification, not a
specific payload.

MSVC 14.51.36231 passes all four C11/C17 low/optimized combinations, each with five floating cases.
Both Linux and Windows C emission pass focused compiler tests. HIR/semantic, interface/context and
MIR codec tests pass, including rejection of unsupported operations and exact type mismatches.
Backend/emission-fixture tests and affected-package vet pass. The full repository suite and full
Windows smoke were not repeated; the changed native crossing was run directly.

Old C-scalar floating receipts do not attest the distinct Fairy types. Hosted Linux execution remains
pending until the expanded source runs under GCC and Clang in both configured directions and
profiles. No existing CI job, report counter, product runtime or target maturity flag changes.
