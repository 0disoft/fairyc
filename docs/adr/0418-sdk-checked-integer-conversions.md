# ADR 0418: SDK checked integer conversions

- Status: Accepted; implementation evidence pending
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0413
- Workload evidence: [round4 assessment](../architecture/checked-integer-conversion-assessment.md)

## Decision

Add an allocation-free, pure Fairy module `std::num` with these ordinary
functions and one inert public error enum:

```text
pub enum ConversionError { Negative; OutOfRange; }
pub fn Result<i64, ConversionError> i64_from_usize(usize value);
pub fn Result<usize, ConversionError> usize_from_i64(i64 value);
```

The spellings are accepted for the SDK implementation, not new grammar.
`i64_from_usize` returns the exact mathematical value through i64 maximum
(9,223,372,036,854,775,807); larger inputs return `OutOfRange`.
`usize_from_i64` returns `Negative` for any negative input, including i64
minimum, without negating that input. Every nonnegative i64 is representable
in the existing x64 usize range and returns its exact value.

Only the current eight-byte unsigned usize target descriptions are admitted
for this implementation. A narrower target needs a separately verified bound
and conversion implementation before claiming availability. `OutOfRange` is
not reachable from `usize_from_i64` on x64; its presence does not claim a
validated 32-bit implementation. Existing target maturity is unchanged.

Results retain ordinary mandatory handling, exhaustive switching and compatible
`?` propagation. Invalid input is a recoverable error, never a trap, wrapped
value, saturated value, truncated value or sign reinterpretation. Both functions
have no imports, allocation, I/O, unsafe boundary, foreign symbol, resource or
hidden effect. No compiler intrinsic is needed for this first admission.

## Arithmetic bounds

Reject values above i64 maximum before unsigned-to-signed accumulation and
reject negative signed inputs before signed-to-unsigned accumulation. Decode
at most 63 bits by division/remainder with matching operand types. Add powers
only when the corresponding bit is set, and double a power only when more
input remains. Thus the highest power is 2^62 and every intermediate and
successful result is representable; the final iteration never creates 2^63
in an i64 variable. Zero succeeds without entering the loop.

This implementation prioritizes one reusable SDK contract over repeated
application loops. It makes no speed, code-size or constant-time claim.

## Caller boundaries

Conversion establishes representability, not a valid offset or allocation size.
Callers retain format parsing/overflow guards and application limits. Check
`offset <= length`, then `count <= length - offset`, before constructing a
range. The existing closed `as` allowlist remains unchanged, including inside
an unsafe block. General narrowing, wrapping/saturation, floating conversions,
new scalar types and generic conversion syntax are excluded.

## Compatibility and acceptance

Advance the bundled SDK from `0.0.0-stage0.73` to `0.0.0-stage0.74` when the
implementation is activated. Language `/20`, runtime ABI `/0`, MIR and
artifact schemas remain unchanged: the module uses existing checked arithmetic,
calls and user-enum Result transport. Existing source remains valid. Importing
workspaces refresh SDK locks; embedded source digest and current interface/cache
identities capture the additional module and source without migrating historical
identity fixtures or round4 receipts.

Required evidence is exact results for 0, 1, values beyond i32, powers-of-two
boundaries and i64 maximum, exact errors for -1/i64 minimum and i64 maximum
plus 1/usize maximum, successful round trips, compatible error propagation,
and retained cast/unhandled-Result rejection. Verify SDK discovery and workspace
loading, interpreter and artifact round trip, generated C and actual Windows
MSVC development/release execution. Do not claim Linux native execution from
source or C-emission checks alone.
