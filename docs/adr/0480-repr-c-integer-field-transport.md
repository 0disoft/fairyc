# ADR 0480: Remaining Fairy integer field transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)

## Decision

Admit distinct Fairy `i8`, `i16`, `u16`, `u64` and `isize` values for fields of nonempty
acyclic repr(C) records. Values received from C support field reads, typed locals, internal
value arguments/results, whole-record copies and explicit reconstruction, including nested
repr(C) records. Source identities remain distinct from each other and from exact C scalars.

Use `int8_t`, `int16_t`, `uint16_t`, `uint64_t` and `intptr_t`, respectively. TargetSpec
language-scalar layouts own size and alignment; generated assertions check the C providers.
`isize` follows the target pointer width, not C `long` (which differs on LLP64).

This admission does not add literals, operators, comparisons, casts, arrays/views, direct
scalar-field stores, private scalar fields, scalar resource/tagged/enum payloads, bare foreign
integer signatures or raw pointers to these new scalar families. A nested repr(C) record can
still be used wherever that record was already admitted. Foreign calls remain unavailable in
the pure interpreter.

Builtin identities are preserved through verified MIR, the existing MIR codec, public-interface
hashes and stable package-context rebinding. There is no new opcode, serialized field, SDK
export, CLI contract, workflow or target maturity promotion. Phase 2 remains partial.

## Delivery

The first commit adds type identities and package preservation. The next connects C field
layout/emission and the shared native record probe. Independent C code must observe signed
minima/maxima, unsigned maxima (including values above signed 64-bit range), zero and ordinary
values after Fairy copies/reconstruction and by-value C calls. Caller records must remain
unchanged when the callee mutates its copy. Hosted Linux evidence is collected after an
authorized batched push; earlier receipts do not attest these new sources.
