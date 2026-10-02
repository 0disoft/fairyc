# ADR 0420: SDK checked i64 decimal parsing

- Status: Implemented in SDK 0.0.0-stage0.76
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0328, ADR 0413, ADR 0418
- Workload evidence: [checked i64 decimal parsing assessment](../architecture/checked-i64-decimal-parsing-assessment.md)

## Context

Two maintained quickstarts duplicate the same signed decimal parser, including
minimum-value handling and range preflight. The round4 interval workload uses
an equivalent negative accumulator. Ledger and access-log fields instead have
digits-only syntax. These sources establish repeated library work, not a
compiler defect or a need for new grammar, casts or runtime symbols.

At decision time SDK `0.0.0-stage0.75` supplies checked integer conversions in
`std::num` but no decimal parser. The assessment retains inspected source hashes
and separates earlier application execution from future SDK acceptance evidence.
External round4 source, receipts and fixed compiler identities remain unchanged.

## Decision and exact contract

Accept one ordinary pure Fairy function and one separate inert error enum in
`std::num`:

```text
pub enum ParseI64Error { Empty; InvalidDecimal; OutOfRange; }
pub fn Result<i64, ParseI64Error> parse_i64_decimal(ref u8[] raw);
```

The parser consumes the entire immutable borrowed byte view on success with
ASCII grammar `-?[0-9]+`. An optional minus is permitted only at byte zero.
Leading zeros and `-0` are valid; no canonical spelling is required. Return the
exact value within `[-9223372036854775808, 9223372036854775807]`.

- Zero bytes return `Empty`.
- A lone minus, plus, repeated signs, whitespace, delimiter, decimal point,
  NUL, BOM, non-ASCII byte or other nondigit returns `InvalidDecimal` when
  encountered in the order below.
- A valid digit that would exceed the sign-specific i64 bound returns
  `OutOfRange` before performing unrepresentable arithmetic.

There is no trimming, locale, Unicode digit support, radix prefix, underscore,
partial-value success, saturation, wrapping, truncation, sentinel, trap,
console output or global error state. Invalid UTF-8 bytes are invalid decimal
syntax; this function does not validate text or return a UTF-8 error. Callers
requiring separate text diagnostics own that validation.

Keep `ConversionError` unchanged: parsing failure and representability conversion
are distinct contracts, and a negative signed number is valid input. Results
require ordinary mandatory handling through compatible `?` propagation or
exhaustive switching. No language or cast allowlist entry is added.

## Error order

Use left-to-right first-failure order: check empty input, inspect an optional
minus and require a following digit, then classify each byte before range
preflight for that digit. Return immediately at the first failure. Do not scan
a suffix after overflow solely to prefer a later syntax error.

| Input | Result |
| --- | --- |
| `0`, `-0`, `000256`, `-0001` | 0, 0, 256, -1 |
| `9223372036854775807` | i64 maximum |
| `-9223372036854775808` | i64 minimum |
| `9223372036854775808`, `-9223372036854775809` | `OutOfRange` |
| Empty byte view | `Empty` |
| `-`, `+1`, `--1`, `1 0`, `0x10` | `InvalidDecimal` |
| `9223372036854775808x` | `OutOfRange` |
| `922337203685477580x8` | `InvalidDecimal` |
| `-9223372036854775809x` | `OutOfRange` |

These normative expectations are covered by the implementation evidence below. Syntax-first
whole-input validation would be a different policy and is not admitted.

## Purity, limits and arithmetic bounds

Borrow the bytes without changing the view or its owner. Use constant scalar
storage, with linear work in bytes visited and no allocation, resource,
foreign call, intrinsic or hidden effect. Do not copy the examples' 20-byte
argv capacity into the SDK: arbitrary leading zeros can make a valid spelling
longer than its shortest representation. Argument, field, row and aggregate
budgets and streaming assembly remain caller-owned limits.

Accumulate negatively from zero. Select `limit = MIN` for negative spelling,
otherwise `limit = -MAX`. For each digit `d` in 0..9 reject if:

```text
value < limit / 10 || (value == limit / 10 && d > -(limit % 10))
```

Only then compute `value = value * 10 - d`. Division truncates toward zero;
`limit % 10` is -8 for MIN and -7 for -MAX, so remainder negation is safe.
On an admitted step, `value * 10` is between the chosen limit and zero, and
subtracting `d` stays at or above the limit. Both intermediates are representable.
Return the accumulator for negative spelling. Negate only positive spelling,
whose accumulator is bounded below by -MAX. Never form positive MIN magnitude
or negate MIN. Digit acquisition must use admitted value-preserving widening,
without adding a new narrowing cast.

The implementation uses this single guard; the arithmetic argument is
supplemented by the independent mathematical oracle and execution evidence below.
No performance, constant-time, narrower-architecture or target-promotion claim
accompanies this decision.

## Caller and adoption boundaries

The offset quickstart continues to own argv acquisition/capacity, checked usize
conversion, bounds and diagnostics. The byte codec owns byte and field width,
endianness and excess-high-bit policy. Interval callers own row context,
endpoint ordering and checked length calculations. A parsed i64 establishes none
of these application policies.

Digits-only ledger/log fields must reject a leading minus in the original
bytes, including `-0`, before using this signed parser. Checking only the
returned value would silently broaden their syntax. Avoid a whole-field syntax
prescan that changes first-failure ordering; preserve field-specific messages.
Log status retains its exact three-digit/range rule. Migrate maintained
quickstarts separately, preserving their argv/output expectations; do not
rewrite external round4 programs or historical evidence.

## Compatibility and activation

The earlier acceptance-only change added no executable API and bumped no version.
Implementation advances the bundled SDK from `0.0.0-stage0.75` to
`0.0.0-stage0.76` with the acceptance evidence below. Language `/20`, runtime ABI `/0`, MIR and artifact
schemas stay unchanged for an ordinary-library addition. Source/module digest,
SDK interface and cache identities change normally; workspace SDK locks refresh
under the [versioning contract](../compatibility/versioning-v0.md).

Existing source stays valid. Historical identity fixtures and receipts remain
intact. Implementation and acceptance evidence land together.

## Implementation acceptance evidence

Compare exact values and exact error variants against an independent
mathematical oracle, not another copy of the chosen guard. Cover MIN/MAX and
adjacent failures, powers-of-ten boundaries, long leading zeros, signed zero,
empty input, lone/repeated signs, plus, whitespace, NUL/BOM/non-ASCII bytes,
and mixed syntax/overflow order. Use immutable full views and field subviews;
confirm owner bytes remain unchanged. Bound test input independently without
inventing an SDK length cap.

Verify successful and error `?` propagation, exhaustive error switching,
unhandled Result rejection and incomplete enum matching rejection. Exercise
the interpreter after verified MIR artifact-codec round trip, C generation for
current x64 target descriptions, and actual Windows MSVC low and optimized
native execution. Retain exact argv/cwd/stdout/stderr/exit and source, SDK,
compiler/toolchain identities and EXE SHA256; native text readers explicitly
use UTF-8. C generation alone is not Linux native execution.

Run targeted SDK discovery, workspace loading/compilation and cache-identity
checks appropriate to the addition. Do not repeat unrelated workload suites or
performance experiments. The implementation evidence below records completion
for the stated scope.

## Implementation evidence

`internal/sdkstdlib/sources/std/num.fy` implements the parser with the admitted
negative accumulator and preflight guard. The maintained
`internal/cli/checked_i64_decimal_test.go` computes expectations from positive
arbitrary-precision Go `math/big` prefix magnitudes, independently of that guard.
Its 157 inputs cover signed power-of-ten neighbors through 10^19, MIN/MAX and
adjacent failures, signed zero, 96 leading zeros, empty/sign-only input,
whitespace, separators, NUL/BOM/non-ASCII/malformed UTF-8 and competing
syntax/range errors. These are bounded fixtures, not a parser length limit.

Each input checks direct and `?`-propagated results through immutable full views
and bounded subviews; empty input uses an empty subview of a real owner. Every
owner byte is compared with its saved bytes afterward. Exhaustive error matching
executes; unhandled Results and incomplete error matches are rejected.

The verified MIR executes after artifact-codec round trip, and C emission passes
for both current x64 target descriptions. Actual Windows MSVC low and optimized
images execute the entire fixture with exit zero and empty stdout/stderr.
The [retained receipt](../ops/evidence/checked-i64-decimal-20261002.json) records
the fixture hash, exact byte inputs/results, native argv/cwd/exit, streams decoded
explicitly as UTF-8, SDK source identity, compiler/toolchain identities and EXE
SHA256. The in-process CLI compiler identity is the actual Go test executable
hash/build information plus source-base and source hashes, not a standalone
`fairyc.exe` build. SDK discovery, loader, package-cache and targeted workspace
identity regressions pass. Linux native execution and performance were not tested.

## Rollout and deferred scope

Implementation and acceptance evidence are complete for this scope. The two
maintained quickstarts now use `std::num::parse_i64_decimal` through exhaustive
caller-error adapters. Their 20-byte argv budgets, diagnostics and application
policy order remain unchanged. Example package versions advance to `1.0.1`;
SDK/language/runtime versions do not change for adoption. The new
[offset receipt](../ops/evidence/checked-parser-adoption-offset-20261002.json)
and [codec receipt](../ops/evidence/checked-parser-adoption-codec-20261002.json)
record 37 and 40 CLI cases, respectively, in both Windows native profiles.
They add competing syntax/range cases to the original corpora. Historical
receipts remain unchanged. Other
integer types, digits-only wrappers, radix/options, byte offsets, text overloads,
prefix parsing, streaming parser state, FFI, intrinsics and performance
experiments are deferred. Compiler grammar, runtime symbols, external round4
product source and storage behavior remain unchanged.
