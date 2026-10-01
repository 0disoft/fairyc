# Checked integer conversion assessment after round4

Status: Candidate assessment; no API admission or implementation claim.
Date: 2026-10-02 (Asia/Seoul).

This assessment records the pre-implementation reasoning. The subsequent
accepted SDK contract and current implementation status are owned by
[ADR 0418](../adr/0418-sdk-checked-integer-conversions.md).

## Recommendation

Prioritize two explicit fallible SDK functions for `usize -> i64` and
`i64 -> usize`. Start with a pure Fairy implementation for the existing x64
targets. Do not expand `as`, implicit conversions, or the integer type family
to remove application boilerplate.

The workload evidence establishes repeated code, not a compiler defect.
It does not yet establish that an ordinary SDK library is materially worse
than a compiler feature. [ADR 0218](../adr/0218-modern-language-concept-admission-boundary.md)
requires that distinction before admitting a language feature. The functions
below are proposed spellings, not available APIs or reserved names.

## Evidence and identity

Source contracts were inspected at compiler repository commit
`dfba3b17a2a8ca0dc402c1e79514b9e130f940e3`. Round4 used the separate pinned
compiler revision `c34718bd60ef08596629113e554321357cf2c3fd`, unmodified,
SDK `0.0.0-stage0.73`, language `/20`, runtime `/0`. Its EXE SHA256 was checked:
`7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917`.

The local application evidence root is
`C:\Users\cherr\Downloads\fairy_c_test_projects\round4`.
Those applications are external evidence, not maintained compiler fixtures.
Their relevant source was read directly; hashes identify the inspected bytes.
This assessment did not rerun the application suites or performance experiments.

| Evidence path relative to round4 | Observed repeated code | SHA256 |
| --- | --- | --- |
| `03-binary-patch-cli/src/patches.fy` | `record` parses the offset into i64 while separately accumulating an input-length-bounded usize; counts hex pairs again for an i64 length. | `2e0b25805d7129f2a82e5f7650aeb88cd3ae22b93e6e195a42fd027a20de519d` |
| `05-resource-pack-cli/src/codec.fy` | `small` converts bounded usize values by binary accumulation; `decode` builds both i64 and a separately bounded usize. | `afb09ec8c09d47f896079d30f60ade4129e9f3a72d6dfa8c0e5092841660a5c8` |
| `09-persistent-kv-store/src/codec.fy` | Same conversion pattern, with a different application bound; `encode` separately assembles u8 values bit by bit. | `0cd44acb5c24b04ee8eb0d211c0d631912137ae4b2c038b6f7ca9ea75f9270d7` |
| `09-persistent-kv-store/src/snapshot.fy` | `size` decodes four bytes directly into usize after snapshot validation. This is a bounded format decoder, not itself a general scalar conversion. | `78a429920f20997c238daab4917ceca2672ed65cb5e66474664a2024c4ba33d7` |

The pack and KV `small` helpers explicitly rely on caller bounds of 1,083,924
and 1,048,576 respectively. They must not be copied as general checked APIs.
The binary patch report records rejected `usize -> i64`, `i64 -> usize`, and
`u32 -> u8` attempts as expected executable-subset limitations.

A representative retained source pattern is:

```text
var usize remaining = input; var i64 value = 0; var i64 power = 1;
while (remaining > 0) {
    if (remaining % 2 != 0) { value += power; }
    remaining /= 2;
    if (remaining > 0) { power *= 2; }
}
```

This source demonstrates duplication and caller assumptions. It supplies no
timing, code-size improvement, or general-range correctness proof.

## Current owning contracts

- [ADR 0413](../adr/0413-executable-signed-64-bit-integers.md) admits checked
  i64 arithmetic and explicit value-preserving widening from u8, i32 and u32.
- [Typing v0](../language/0007-name-resolution-and-typing-v0.md) rejects
  usize-to-i64 casts and leaves other integer pairs unsupported until an
  explicit fallible conversion contract exists. Read its i64 amendment together
  with the older closed `as` list.
- `internal/hir/hir.go::ValuePreservingIntegerCast` and
  `internal/semantic/typecheck.go::checkCast` implement that closed subset.
  Neither direction under review is admitted. An `unsafe` block does not
  make an excluded integer cast executable.
- `internal/target/spec.go` defines unsigned eight-byte usize for both current
  x64 target descriptions. Host Go integer width is not the source contract.
  Target descriptions alone do not promote either target's maturity.
- The bundled SDK currently has alloc, bytes, text, fs, io and process modules;
  no numeric conversion module is present. `std::bytes::append_i64` is a
  formatter, not a conversion API. Its private `signed_decimal_digit` handles
  a small digit domain and does not justify a universal narrowing cast.
- [ADR 0328](../adr/0328-production-workload-capability-admission-gates.md)
  keeps production capabilities evidence-gated. These small CLI workloads do
  not establish production-target or storage-durability readiness.

## Proposed SDK contract

Possible module: `std::num`, subject to SDK naming review. Proposed ordinary
functions, with a closed inert error enum for `negative` and `out_of_range`:

```text
pub fn Result<i64, ConversionError> i64_from_usize(usize value);
pub fn Result<usize, ConversionError> usize_from_i64(i64 value);
```

These declarations are design sketches only.

| Direction | Success | Recoverable error |
| --- | --- | --- |
| usize to i64 | Return exactly the input if it is at most 9,223,372,036,854,775,807. | `out_of_range` above i64 maximum. |
| i64 to usize | Return exactly the input if it is nonnegative and fits the target's usize range. | `negative` below zero; `out_of_range` above the target maximum. |

The caller must handle `Result` through the existing result-handling contract.
No wrapping, saturation, truncation, sign reinterpretation, sentinel value,
allocation, I/O, hidden trap on bad input, or global error state is proposed.
Existing `as` rejection remains intact.

Representability is separate from application validity. After conversion,
callers still check file length, offsets, buffer capacity and format limits.
For ranges, check `offset <= length` then `count <= length - offset` rather
than computing a potentially overflowing `offset + count` first. Decimal and
binary parsers retain their own parse-overflow checks before conversion.

On existing x64 targets, every nonnegative i64 fits usize, while usize's upper
half does not fit i64. A pure library can reject the upper half before binary
accumulation and reject negative i64 before constructing an unsigned value.
Bound the loop to at most 63 value bits after those guards. It must not multiply
a signed power after the last bit or construct an out-of-range intermediate.
This is an implementation approach to prove, not a tested implementation.

Initially claim only the existing x64 widths. If a narrower target is later
admitted, the library must obtain a target-owned bound or receive a separate
reviewed implementation. A Go-host-derived bound and unconditional 64-bit
usize assumption cannot establish portability. No new target is proposed here.

## Small implementation units and acceptance evidence

1. Accept a separate SDK feature decision with exact names, errors, target
   scope, ownership/effects and compatibility treatment. Check in a minimal
   application-shaped offset/length example derived from the repeated pattern.
   The external round4 files remain unchanged.
2. Implement both directions as ordinary pure Fairy functions using the admitted
   arithmetic and Result surface. Keep the C ABI and scalar-cast allowlist
   unchanged. Record the SDK source/identity change under the current release
   rules; decide any cache impact before activating the SDK source.
3. Verify 0, 1, i32 maximum plus 1, i64 maximum, negative 1, i64 minimum,
   and usize values i64 maximum plus 1 and usize maximum. Verify exact payloads
   and errors, successful round trips, caller error propagation, and continued
   rejection of the two `as` conversions. Use semantic/interpreter checks and
   actual development/release generated-Fairy native execution on each target
   for which availability is claimed. Review terminal arithmetic bounds so
   implementation bugs cannot replace the promised error with a trap.
4. Rewrite the maintained offset/length example to parse once, convert once,
   then apply its original application bound. Compare source duplication and
   failure ordering. Stop here unless concrete evidence shows that the pure
   library creates an unacceptable cost or target limitation. Only then assess
   a compiler-private lowering or intrinsic, with its own exact identities and
   semantic/MIR/backend evidence.

Each implementation unit needs its own local commit and owning validation.
No performance threshold or zero-cost claim follows from this static review.

## Deferred scope and review limits

General `try_cast<T>`, traits, overloads, user-defined implicit conversions,
u64/isize admission, floating conversions, foreign ABI conversions, wrapping
and saturating families remain separate proposals. u32/i64-to-u8 narrowing has
codec evidence, but is deferred to keep the first unit about size/index values.
Byte-format decoding and SDK parser APIs are not automatically replaced.

This document changes no grammar, diagnostics, runtime, API availability,
language/SDK version, application source, or storage behavior. New source and
native tests are implementation acceptance work, not completed evidence here.
