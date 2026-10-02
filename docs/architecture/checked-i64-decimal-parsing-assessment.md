# Checked i64 decimal parsing assessment after round4

Status: Assessment concluded; API accepted by [ADR 0420](../adr/0420-sdk-checked-i64-decimal-parsing.md); implementation pending.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `3250a448`.
Bundled SDK: `0.0.0-stage0.75`; language `/20`; runtime ABI `/0`.

## Recommendation and decision boundary

The assessment recommended one pure, fallible signed decimal parser in
`std::num`; ADR 0420 now accepts that contract. Two maintained quickstarts
duplicate the same negative accumulator, and round4 interval parsing uses an equivalent guard. This is
repeated library work, not evidence of a compiler defect or a new language need.
Keep grammar, runtime symbols, casts and external round4 projects unchanged.

The following assessment sketch is now accepted by ADR 0420, but is not yet
an available API:

```text
pub enum ParseI64Error { Empty; InvalidDecimal; OutOfRange; }
pub fn Result<i64, ParseI64Error> parse_i64_decimal(ref u8[] raw);
```

Use a separate parse error rather than extending `ConversionError`: a negative
number is valid signed input, while missing digits and invalid bytes are not
conversion failures. `ref u8[]` accepts argv buffers and row-field subviews
without allocation or a preceding whole-input UTF-8 conversion.

| Option | Assessment | Reason |
| --- | --- | --- |
| One signed decimal SDK parser | Advance to an ADR candidate | Exact duplication in two maintained examples and a third signed workload. |
| Keep all parsers local | Valid fallback | No SDK commitment, but repeats minimum-value and overflow reasoning. |
| Add digits-only/nonnegative wrappers now | Defer | Unsigned field syntax is real, but one shared parser plus a caller syntax check can be assessed first. |
| Generic radix/options API, implicit parsing or intrinsic | Defer | No need established by the inspected workloads; increases the contract surface. |
| Error byte offsets, text overload or prefix parsing | Defer | Current callers attach field/line context; a public offset or another entry point needs stronger evidence. |

## Inspected workload evidence

The compiler checkout was clean at the inspected revision. SHA256 identifies
the actual read bytes, including checkout line endings. External paths below
are relative to `C:\Users\cherr\Downloads\fairy_c_test_projects\round4`;
the examples are relative to the compiler repository.

| Source | Observation | SHA256 |
| --- | --- | --- |
| `examples/checked_offset_quickstart/src/numbers.fy` | `parse` accepts `-?[0-9]+`, uses a negative accumulator, distinguishes empty/syntax/range; argv and usize conversion are separate. | `7c583edbc8726244cdb6041be4949066453514976aa34e6c0abe9fa3c97e8f0f` |
| `examples/checked_byte_codec_quickstart/src/numbers.fy` | Same signed parsing body; its argv adapter returns i64 without conversion. | `990547c1d44018bb5bf1bb313570856b077284171d4cbae42addf74becae778a` |
| `08-interval-merger/src/numbers.fy` | Signed negative accumulation with equivalent `(limit + digit) / 10` guard; errors perform caller-owned console diagnostics. | `c56b6ac2e6d1948f9198db4195561a1fb9f8d5e7661866f0ab90828bb8be8be9` |
| `01-ledger-reconciler/src/ledger.fy` | `amount` accepts digits only, with a positive accumulator and pre-arithmetic i64 overflow guard. | `e9b5afccb0b0da265f04b13e6070490f0d8103c02ff3f340b5f3707895f95488` |
| `02-access-log-analyzer/src/format.fy` | The bytes field accepts digits only after row UTF-8/field/path checks; status has a separate three-digit/range contract. | `385503eeadf2e4b08058ae3647488fd2dfa253873b0d5c8c507cd2e2b2b75475` |

`internal/sdkstdlib/sources/std/num.fy`, SHA256
`1f786295d03f9f648e43adc8b2dbfa62bd671a82a057ee58f5ece771783c892e`,
currently supplies three checked integer conversions and no numeric parser.
The inspected bundled SDK sources expose no decimal parsing entry point.
`std::text` validation and numeric byte-buffer append helpers do not supply one.

The prior [offset example receipt](../ops/evidence/checked-offset-example-20261002.json)
and [byte codec example receipt](../ops/evidence/checked-byte-codec-example-20261002.json)
retain executions of those application parsers. They are not verification of a
new SDK parser. This assessment did not rerun their suites or the external round4
programs. Their sources, reports, compiler identities and receipts remain intact.

## Proposed exact input and result contract

Consume the entire borrowed byte view with grammar `-?[0-9]+`. An optional
ASCII minus is allowed only at byte zero. Accept leading zeros and `-0`, both
as ordinary decimal values; no canonical spelling is imposed. Success returns
the exact value within `[-9223372036854775808, 9223372036854775807]`.

- Zero bytes: `Empty`.
- A lone `-`, a plus, repeated signs, whitespace, a separator, a decimal point,
  NUL, BOM, non-ASCII byte or any nondigit: `InvalidDecimal` when encountered.
- The next valid digit would exceed the sign-specific i64 bound: `OutOfRange`.
- No trimming, locale, Unicode digits, radix prefixes, underscores, saturation,
  wrapping, truncation, sentinel, trap, partial-value success or console output.

The bytes are inspected as ASCII syntax. Even malformed UTF-8 is an invalid
decimal byte; the parser does not claim UTF-8 validation or return a text error.
A caller requiring a distinct UTF-8 error must validate that contract separately.

The SDK should impose no copied 20-byte argv limit: leading zeros can make a
valid number longer than its shortest representation. Process argument capacity,
row/field size, aggregate input budget and streaming assembly remain caller
limits checked before parsing. Work is linear in bytes visited, with constant
scalar storage and no allocation, resource, foreign call or hidden effect.
No performance, narrower-target or target-promotion claim follows from this design.

## Error order

Recommend the existing signed parsers' first-failure order: empty check, optional
minus/digit-presence check, then left-to-right byte classification followed by
range preflight for each digit. Return immediately on the first failure; do not
scan a suffix after overflow solely to prefer a later syntax error.

| Candidate input | Expected result | Reason |
| --- | --- | --- |
| `0`, `-0`, `000256`, `-0001` | 0, 0, 256, -1 | Leading zeros and signed zero are admitted. |
| `9223372036854775807` | i64 maximum | Exact upper boundary. |
| `-9223372036854775808` | i64 minimum | No positive magnitude or minimum-value negation. |
| `9223372036854775808`, `-9223372036854775809` | `OutOfRange` | First unrepresentable digit. |
| empty bytes | `Empty` | No sign or digit exists. |
| `-`, `+1`, `--1`, `1 0`, `0x10` | `InvalidDecimal` | Whole-input grammar, no prefix success. |
| `9223372036854775808x` | `OutOfRange` | Overflow is reached before the trailing invalid byte. |
| `922337203685477580x8` | `InvalidDecimal` | Invalid byte is reached before any overflow. |

These assessment expectations are now fixed by ADR 0420, not executed SDK
test results. Syntax-first whole-input validation would
be a different public policy and could change current callers' diagnostics.

## Arithmetic feasibility

Reuse ordinary checked i64 arithmetic. Accumulate negatively from zero and
choose `limit = MIN` for a negative spelling, otherwise `limit = -MAX`.
For each ASCII digit `d`, reject if `value < limit / 10` or if
`value == limit / 10 && d > -(limit % 10)`. Only then compute
`value = value * 10 - d`. Division truncates toward zero; the remainder is
bounded to -9..0, so its negation is safe. The guard bounds both the multiplication
and subtraction intermediates. Return the accumulator for negative input and
negate only the positive spelling, whose lower bound is `-MAX`.

This is source inspection and arithmetic reasoning, not a compiled SDK proof.
The interval workload's `(limit + d) / 10` guard is a second current formulation;
one admitted implementation should choose one guard and verify it independently.

## Caller policies and adoption limits

The offset example still owns argv acquisition/capacity, `usize_from_i64`, range
bounds, errors and output. The codec example still owns byte/width/high-bit and
endian policies. Interval parsing still attaches path/line context and validates
endpoint ordering and merged lengths separately.

Ledger/log fields forbid any sign, including `-0`. Checking only the returned
value for negativity would silently admit `-0`: reject a leading minus in the
original bytes before calling a signed parser, which still rejects plus and
other nondigits. Avoid a whole-field syntax prescan if it changes first-failure
order. Preserve syntax/range error ordering and field-specific messages. Do not route the
log status field through a permissive parser and lose its exact three-digit rule.
Parsing establishes an i64 value, not application bounds or valid UTF-8 for other fields.

## Later acceptance, compatibility and deferred work

[ADR 0218](../adr/0218-modern-language-concept-admission-boundary.md) and
[ADR 0328](../adr/0328-production-workload-capability-admission-gates.md)
keep admission evidence-based. This assessment itself accepts no API and changes
no version; the separate ADR 0420 now supplies the accepted contract before
implementation. If later implemented, advance the SDK and its source
digest under [versioning](../compatibility/versioning-v0.md); language `/20`,
runtime ABI `/0` and schemas need no change for an ordinary library addition.

Later acceptance must compare exact results with an independent mathematical
oracle: minimum/maximum and adjacent failures, powers-of-ten boundaries, long
leading zeros, signed zero, arbitrary invalid bytes and mixed syntax/overflow
order. Verify Result handling/propagation, interpreter after MIR codec round
trip, current x64 C generation and actual Windows low/optimized native execution.
Check SDK discovery and workspace/cache identity changes. Migrate maintained
examples in a separate bounded step with their original argv/output expectations;
do not rewrite external round4 evidence. C generation is not Linux native proof.

Other integer types, radix/options families, byte offsets, text overloads,
streaming parser state, FFI, intrinsics, application migrations and performance
experiments are deferred. Only source/contract inspection and document checks
belong to this assessment; no SDK build or native parser verification is claimed.
