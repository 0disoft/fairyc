# ADR 0419: SDK checked i64-to-byte conversion

- Status: Implemented in SDK 0.0.0-stage0.75
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0413, ADR 0418
- Workload evidence: [checked byte conversion assessment](../architecture/checked-byte-conversion-assessment.md)

## Context

Round4 pack and KV encoders reconstruct bytes from an i64 using the same nested
eight-bit loop. The assessment identifies repeated application code, not a
compiler defect. At decision time SDK `0.0.0-stage0.74` had only the two size
conversions in `std::num`; byte narrowing was not available. The existing `as` allowlist
rejects i64-to-u8 and u32-to-u8 casts, including in unsafe blocks.

The binary patch program already decodes hex nibbles using u8 arithmetic. Its
earlier rejected u32 cast is weaker evidence for an unsigned conversion API.
Keep that workload and the external round4 source and receipts unchanged.

## Decision and exact contract

Accept exactly one ordinary pure Fairy function in `std::num`:

```text
pub fn Result<u8, ConversionError> u8_from_i64(i64 value);
```

Reuse the existing inert `ConversionError { Negative; OutOfRange; }` unchanged.
Inputs from 0 through 255 return their exact u8 value. Any negative input,
including i64 minimum, returns `Negative`; check the sign without negating the
input. Inputs above 255, including i64 maximum, return `OutOfRange`.
No invalid input traps, wraps, saturates, truncates, reinterprets sign bits,
returns a sentinel or changes global error state.

Results require ordinary mandatory handling through compatible `?` propagation
or exhaustive switching. No error variants, grammar, scalar types, implicit
conversions or `as` allowlist entries are added. Unsafe does not enable a
rejected narrowing cast. `u8_from_u32` is deferred; a caller can explicitly use
the already admitted value-preserving u32-to-i64 widening before this function.

## Purity and arithmetic bounds

Use existing Fairy arithmetic and Result/enum transport. The function has no
allocation, I/O, foreign symbol, resource, intrinsic or hidden effect.
After the sign and upper-bound checks, decompose the nonnegative i64 remainder
with `% 2` and `/ 2`. Start u8 result at zero and u8 power at one. Add a power
only for a set bit, divide the remainder, and double the power only while more
input remains. There are at most eight rounds, power is at most 128 and every
partial sum is at most 255. Zero skips the loop. Never construct 256 in u8.

The implementation below follows these arithmetic bounds. Fixed-width operands
avoid a usize-width assumption, but admit
no target, narrower architecture or Linux native execution. No speed, image-size
or constant-time claim accompanies the decision.

## Codec caller boundary

Conversion establishes byte representability. Converting the full value 256
must return `OutOfRange`. A caller intentionally serializing a nonnegative
multi-byte value instead extracts `remaining % 256`, converts that digit and
then divides `remaining` by 256. This may replace the application's inner bit
loop; the library still performs conversion work.

Signed encoding, endianness, field width, excess high bits, format limits,
Buffer append/write errors and publication atomicity remain caller contracts.
Do not silently discard high bits, change error ordering or infer whole-encoder
failure atomicity from this pure conversion. Existing codecs are not migrated
as part of accepting the API.

## Compatibility and activation

The accepted decision was recorded before implementation. Implementation
advances the bundled SDK from `0.0.0-stage0.74` to `0.0.0-stage0.75` after
the acceptance evidence below passes.
Language `/20`, runtime ABI `/0`, MIR and artifact/cache schemas remain unchanged
for this ordinary-library addition. Source/module digest and current SDK
interface/cache identities change normally; workspace SDK locks refresh.
Existing source remains valid. Preserve historical identity fixtures and round4
receipts. The earlier acceptance-only commit did not bump a version.

## Implementation acceptance evidence

Verify all 256 success inputs against independent expected values, including
exact u8-to-i64 widening round trips. Verify exact `Negative` for -1 and i64
minimum and `OutOfRange` for 256 and i64 maximum. Observe successful `?`
propagation and propagation of both errors, exhaustive enum switching and
continued rejection of unhandled Results. Retain i64-to-u8 and u32-to-u8 `as`
rejection, including inside unsafe blocks.

Cover the interpreter, artifact-codec round trip and generated C for the current
x64 target descriptions, plus actual Windows MSVC low and optimized execution.
Retain exact results, source/SDK/compiler/toolchain identity and EXE SHA256.
C generation alone is not Linux native execution. Run targeted SDK discovery,
workspace loading/compilation and cache-identity regressions appropriate to the
added function; do not repeat unrelated workload suites.

A small maintained codec fixture/example must distinguish full-value 256
rejection from little-endian extraction of `0x0100` as `00 01`. An eight-byte
i64-maximum encoding must yield `ff ff ff ff ff ff ff 7f`. Validate caller
negative-input, width and excess-high-bit policies separately. No example or
acceptance test was claimed as implemented by the earlier acceptance decision.

## Implementation evidence

`internal/sdkstdlib/sources/std/num.fy` implements the function using guarded
pure Fairy bit accumulation. The [retained receipt](../ops/evidence/checked-byte-conversion-20261002.json)
records all 256 exact success inputs and four exact errors. The maintained
`internal/cli/checked_byte_conversion_test.go` fixture uses independent literal
byte expectations and Go `encoding/binary` for the maximum-value byte sequence.
It checks widening round trips and successful/error `?` propagation.

Nine caller codec cases cover two successful encodings and seven failures:
negative -1/i64 minimum, width zero/nine, output capacity and excess high bits
for 256 in one byte and i64 maximum in seven bytes. Each failed encoding leaves
every sentinel byte unchanged. This fixture defines its own field policy and is
not a new public codec API or migration of the external round4 programs.

The verified MIR executes after artifact-codec round trip. C generation passes
for both current x64 target descriptions; actual Windows MSVC low and optimized
images each execute the full fixture once with exit zero and empty stdout/stderr.
Source and SDK identities, EXE SHA256 and MSVC identity are retained. The
compiler identity is the in-process CLI compiler's actual Go test executable
SHA256 and Go build information, plus the modified source base and source hashes;
it does not claim a standalone `fairyc.exe` build. Narrowing cast rejection in
ordinary/unsafe source, unhandled Result rejection and non-exhaustive enum match
rejection pass. SDK discovery, loader, package-cache and targeted workspace
session checks pass. Linux native execution and performance were not tested.

## Rollout and deferred scope

The implementation and acceptance evidence are complete for the stated scope.
The unsigned wrapper, other integer pairs, generic casts,
wrapping/saturating conversions, intrinsics, FFI, new codec/parser APIs and
performance experiments remain outside this decision. Product source, runtime
and storage behavior are unchanged in this acceptance commit.
