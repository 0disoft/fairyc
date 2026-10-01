# Checked byte conversion assessment after round4

Status: Candidate assessment; no API admission or implementation claim.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `71b6cdb63e1c0fc21cebf068cce0401f9a0c8c83`.

This assessment records pre-implementation reasoning for byte conversion. It is
a sibling of the [checked integer conversion assessment](checked-integer-conversion-assessment.md),
whose two directions were accepted by [ADR 0418](../adr/0418-sdk-checked-integer-conversions.md)
and implemented in SDK `0.0.0-stage0.74`. That acceptance covers `usize <-> i64`
only. The names below are design sketches, not available APIs, accepted spellings
or reserved identifiers.

## Recommendation

Prioritize one explicit fallible SDK function for `i64 -> u8`. Consider
`u32 -> u8` as a small convenience wrapper over the already admitted `u32 -> i64`
widening and the same function, without a second conversion algorithm:

```text
pub fn Result<u8, ConversionError> u8_from_i64(i64 value);
pub fn Result<u8, ConversionError> u8_from_u32(u32 value); // optional wrapper
```

Repeated i64 byte reconstruction supports the first candidate. Evidence for the
unsigned wrapper is weaker: the binary patch report records a rejected cast,
but its current hex decoder uses u8 nibbles directly. Do not expand `as`, implicit
conversion or the scalar type family to remove this application boilerplate.

## Proposed exact contract

Reuse the inert `std::num::ConversionError { Negative; OutOfRange; }` from ADR 0418.

| Function | Success | Recoverable error |
| --- | --- | --- |
| `u8_from_i64` | Exact input for 0 through 255. | `Negative` below zero, checked first; `OutOfRange` above 255. |
| `u8_from_u32` | Exact input for 0 through 255. | `OutOfRange` above 255; `Negative` is unreachable. |

Reject negative input without negating it, including i64 minimum. Do not
reinterpret its low eight bits. Callers handle the ordinary mandatory Result
through exhaustive switching or compatible `?` propagation. No wrapping,
saturation, truncation, sentinel, trap or global error state is proposed.

## Workload evidence and identity

Round4 used the pinned compiler revision `c34718bd60ef08596629113e554321357cf2c3fd`,
unmodified, SDK `0.0.0-stage0.73`, language `/20`, runtime `/0`, with EXE SHA256
`7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917`.
The external evidence root is `C:\Users\cherr\Downloads\fairy_c_test_projects\round4`.
Source was read directly; hashes identify the inspected bytes. These applications
are not maintained compiler fixtures. This assessment did not rerun their suites.

| Evidence path relative to round4 | Observation | SHA256 |
| --- | --- | --- |
| `05-resource-pack-cli/src/codec.fy` | `encode` reconstructs each byte from i64 `remaining` in a nested eight-bit loop. | `afb09ec8c09d47f896079d30f60ade4129e9f3a72d6dfa8c0e5092841660a5c8` |
| `09-persistent-kv-store/src/codec.fy` | Same byte reconstruction loop; application size bounds elsewhere differ. | `0cd44acb5c24b04ee8eb0d211c0d631912137ae4b2c038b6f7ca9ea75f9270d7` |
| `03-binary-patch-cli/src/patches.fy` | Hex decoding uses u8 nibbles in 0..15; no wide value is narrowed. | `2e0b25805d7129f2a82e5f7650aeb88cd3ae22b93e6e195a42fd027a20de519d` |
| `03-binary-patch-cli/REPORT.ko.md` | Reports an earlier rejected u32-to-u8 cast and workaround, rather than a current unsigned conversion workload. | `0b07408cff382aaa4ed8501a049dc7eb8a91cbe52da7202d0658da2d892586e3` |

The pack and KV inner loop is repeated serialization code:

```text
var u8 value = 0; var u8 power = 1; var usize bit = 0;
while (bit < 8) {
    if (remaining % 2 != 0) { value += power; }
    remaining /= 2; bit += 1;
    if (bit < 8) { power *= 2; }
}
```

For nonnegative input, it extracts a low byte and leaves the higher digits for
the next iteration. A checked conversion instead rejects the full value 256.
A future caller can explicitly compute `remaining % 256`, convert that digit,
then divide `remaining` by 256. This removes the caller's inner reconstruction
loop, not the SDK's conversion work. It proves no speed or image-size gain.

The converter does not own field width, high-bit rejection, signed encoding,
endianness, format limits, Buffer append failure or publication atomicity.
Preserve those caller policies when adopting it. Current round4 sources and
receipts remain unchanged; this review does not diagnose an encoder defect.

## Owning contracts

- [ADR 0413](../adr/0413-executable-signed-64-bit-integers.md) admits checked
  i64 arithmetic and value-preserving widening from u8, i32 and u32.
- `internal/hir/hir.go::ValuePreservingIntegerCast` admits u32-to-i64 and u8
  widening, but no i64-to-u8 or u32-to-u8 cast.
- `internal/semantic/typecheck.go::checkCast` uses that allowlist; an unsafe
  block does not enable general narrowing.
- `internal/sdkstdlib/sources/std/num.fy` currently supplies only the two size
  conversions and ConversionError. Byte narrowing remains unavailable.
- [ADR 0218](../adr/0218-modern-language-concept-admission-boundary.md) requires
  evidence that a library solution is materially worse before adding a language
  feature. [ADR 0328](../adr/0328-production-workload-capability-admission-gates.md)
  keeps production capabilities evidence-gated.

The repeated code establishes a library candidate, not a compiler defect or a
need for a narrowing intrinsic. SDK `signed_decimal_digit` is a private decimal
alphabet mapping, not a general conversion contract; changing it is unnecessary.

## Candidate implementation and arithmetic bounds

Use an allocation-free pure Fairy function with no foreign call, resource,
hidden effect or target-width dependency. Reject negative input, then values
above 255. Decompose the remaining nonnegative i64 using `% 2` and `/ 2`, with
u8 result zero and power one. Add a power only for a set bit and double it only
while input remains. There are at most eight rounds, power reaches at most 128,
and every partial sum is at most 255. Never construct 256 in a u8 variable.
Zero returns immediately. The unsigned wrapper can return
`u8_from_i64(value as i64)` through the existing value-preserving widening.

This is a mathematical feasibility argument, not compiled or native execution
evidence. Fixed-width operands avoid usize-width assumptions, but do not promote
a target or establish Linux native availability.

## Future acceptance and compatibility

Keep an accepted SDK decision and implementation in separate future commits.
An additive ordinary-library implementation changes SDK version and embedded
source digest under the owning release rules. Language `/20`, runtime ABI `/0`
and cast admission remain unchanged; existing source stays valid and SDK locks
refresh normally. This assessment itself needs no version change.

Before claiming implementation, verify all 256 success values for both admitted
functions and exact widening round trips; signed errors for -1 and i64 minimum;
upper-bound errors for 256, i64 maximum and u32 maximum; Result propagation and
exhaustive switching; unhandled Results and both excluded `as` casts rejected.
Check the arithmetic bounds in the interpreter, artifact round trip and generated
C, plus actual Windows MSVC low and optimized execution. Do not count C generation
as Linux native execution or add a performance gate without workload evidence.

A maintained small codec example should prove that full-value 256 is rejected
while explicit little-endian digit extraction of `0x0100` yields `00 01`. An
eight-byte i64-maximum encoding must yield `ff ff ff ff ff ff ff 7f`. Verify the
chosen width/high-bit and negative-input policies separately, including overflow
and error ordering; conversion alone establishes none of them.

## Deferred scope and review limits

General `as`/`try_cast`, other integer pairs, wrapping/saturating families,
implicit conversions, new scalar types, intrinsics, FFI, parser APIs and generic
binary-codec APIs are separate proposals. This document changes no grammar,
diagnostic, runtime, public API, application source or storage behavior.
Only source/contract inspection and document checks were performed. Builds,
native tests and benchmarks belong to later implementation work. The remaining
question is whether reuse justifies both public entry points; unsigned convenience
has weaker workload evidence than the repeated i64 serialization pattern.
