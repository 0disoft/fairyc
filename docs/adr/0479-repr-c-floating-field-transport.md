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
arrays/views, direct floating-field stores, resource or private enum payload storage. Direct float
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

Implement type identities and package preservation first. Connect C field layout and value emission
in the next slice, then extend the existing shared native record probe. Old C-scalar floating
receipts do not attest the distinct Fairy types. Hosted Linux execution remains pending until the
expanded source runs under GCC and Clang in both configured directions and profiles.
