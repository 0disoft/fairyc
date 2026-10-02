# Digits-only i64 parsing assessment after signed parser adoption

Status: Assessment concluded; defer a separate SDK API; no API accepted.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `feb03bbb814c50c6a1c51780f609d08e9b39835a`.
Bundled SDK: `0.0.0-stage0.76`; language `/20`; runtime ABI `/0`.

## Recommendation and decision boundary

Keep one signed decimal SDK parser. For a future digits-only caller, reject an
ASCII minus at the first byte before calling `std::num::parse_i64_decimal`, then
map its existing parse errors to caller diagnostics. This removes duplicated
numeric accumulation without requiring another public function, error enum or
syntax/options family. The remaining lexical-policy check is small and explicit.

Round4 ledger and access-log sources demonstrate real `[0-9]+` fields. They do
not establish that an additional SDK entry point is necessary now: both belong
to frozen external experiments, and the two maintained numeric quickstarts
require signed grammar, including `-0`. They already use the signed SDK parser.
Do not count a hypothetical rewrite of frozen experiments as an adopted consumer.
This assessment records design guidance only; it changes no product source,
public API, SDK version, compiler behavior, roadmap commitment or external file.

| Option | Decision | Benefit, cost and evidence |
| --- | --- | --- |
| Signed SDK parser plus a leading-minus caller guard | Apply as design guidance for future callers | Removes the numeric loop while keeping digits-only syntax and caller diagnostics; no new SDK surface. Not applied to frozen round4 projects. |
| Dedicated digits-only i64 SDK entry point | Defer | Would centralize the small lexical guard, but currently has no maintained adopter beyond the inspected frozen workloads; requires another public contract, version activation and execution evidence. |
| Signed parsing followed only by a nonnegative-value check | Reject | Accepts `-0` and negative zero spellings, and can report range instead of syntax for an oversized negative spelling. |
| Full-field digit prescan before signed parsing | Reject as an error-preserving replacement | Can prioritize a later invalid byte over an earlier overflowing prefix. |
| Unsigned integer return, generic radix/options, implicit parsing | Defer | These i64 fields need none of those capabilities; digits-only syntax does not establish a usize/u64 value contract. |
| Rewrite external round4 sources to use the current SDK | Outside scope | Their pinned SDK is stage0.73; changing source/toolchain would invalidate the historical experiment identity. |

## Inspected evidence

SHA256 identifies the actual inspected bytes, including checkout line endings.
Compiler-relative paths and external round4 paths are distinguished below.
The external root is `C:\Users\cherr\Downloads\fairy_c_test_projects\round4`.

| Source | Relevant behavior | SHA256 |
| --- | --- | --- |
| External `01-ledger-reconciler/src/ledger.fy` | `amount` rejects empty/nondigit input and guards positive i64 accumulation before multiplication. Whole-file UTF-8 validation and row checks precede it. | `e9b5afccb0b0da265f04b13e6070490f0d8103c02ff3f340b5f3707895f95488` |
| External `02-access-log-analyzer/src/format.fy` | The bytes field uses the same positive accumulation guard, after row UTF-8, controls, fields, status and path checks. Status separately requires three ASCII digits in 100..599. | `385503eeadf2e4b08058ae3647488fd2dfa253873b0d5c8c507cd2e2b2b75475` |
| `internal/sdkstdlib/sources/std/num.fy` | The signed byte-view parser implements exact i64 bounds and left-to-right first-failure order; a digits-only parser is absent. | `09aa0d45f075aa4212049527ded7bb48d99ceb6ffed56b310e267421bae20bb3` |
| `examples/checked_offset_quickstart/src/numbers.fy` | A caller-error adapter delegates parsing to the SDK, then separately converts to usize; its argv limit remains 20 bytes. | `b8a847a6ab464ab11d809f5f11cd037b34da8f8720ded0c02fb16bba9a5d2546` |
| `examples/checked_byte_codec_quickstart/src/numbers.fy` | Another caller-error adapter delegates signed parsing; byte/width/high-bit policies are outside the parser. | `81bf86f77a6c87ccac273898854d4e0d4053bf501a8472fb78f9380c24435fd9` |

The round4 reports identify compiler revision
`c34718bd60ef08596629113e554321357cf2c3fd`, EXE SHA256
`7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917`,
SDK stage0.73, language `/20` and runtime `/0`. Those are historical reported
identities, not a freshly executed compiler check in this assessment. The current
SDK parser cannot be imported under that old toolchain unchanged.

[ADR 0420](../adr/0420-sdk-checked-i64-decimal-parsing.md) owns the implemented
signed parser. Its [retained evidence](../ops/evidence/checked-i64-decimal-20261002.json)
records 157 mathematical-oracle inputs, MIR/interpreter/C and two Windows native
profiles. The [offset adoption receipt](../ops/evidence/checked-parser-adoption-offset-20261002.json)
and [codec adoption receipt](../ops/evidence/checked-parser-adoption-codec-20261002.json)
record maintained signed callers. These prior receipts do not execute the
digits-only composition proposed here; no new native proof is claimed.

## Minimal composition and exact policy

Digits-only means ASCII `[0-9]+`, with no sign, whitespace, separator, Unicode
digit or radix prefix, and exact i64 values from 0 through MAX. Leading zeros
remain valid. Empty input maps to `Empty`, other syntax to `InvalidDecimal`, and
an overflowing digit to `OutOfRange`, in the same first-failure order.

The following is an unimplemented application-helper sketch, not a new SDK API
or a runnable project command:

```text
fn Result<i64, num::ParseI64Error> digits_only(ref u8[] raw) {
    if (raw.len > 0) {
        if (raw[0] == 45) { return .err(.InvalidDecimal); }
    }
    return num::parse_i64_decimal(raw);
}
```

Empty input reaches the SDK's existing empty check. A leading minus is rejected
as syntax before any signed magnitude work. For every other first byte, the
signed parser cannot choose its negative branch: its admitted success spellings
are digits only, with the same 0..MAX bound as the inspected positive loops.
Plus and other invalid bytes are already rejected by the SDK. The caller need
not copy the accumulator, scan all bytes first or inspect the resulting sign.

| Input | Digits-only field result | Why a shortcut can differ |
| --- | --- | --- |
| Empty | `Empty` | Retain each caller's empty-field message. |
| `0`, `000256` | 0, 256 | Leading zeros are syntax, not a fixed-width representation. |
| `-0`, `-0000` | `InvalidDecimal` | Checking only whether the parsed value is negative admits both. |
| `-1` | `InvalidDecimal` | The field forbids a sign; this is not an integer-conversion error. |
| `-9223372036854775809` | `InvalidDecimal` | Direct signed parsing reaches `OutOfRange`; rejecting the sign afterward is too late. |
| `+1`, `1-0`, `1 0`, `0x10` | `InvalidDecimal` | Existing SDK syntax rejection is sufficient after the leading-minus guard. |
| `9223372036854775807` | i64 maximum | Same exact bound in both accumulation formulations. |
| `9223372036854775808` | `OutOfRange` | A valid digit exceeds MAX. |
| `9223372036854775808x` | `OutOfRange` | A full-field syntax prescan would incorrectly prefer the trailing `x`. |
| `922337203685477580x8` | `InvalidDecimal` | Invalid byte appears before range overflow. |

These are source-derived expectations and arithmetic reasoning, not new executed
SDK/helper tests. The positive loops' `value > (MAX - digit) / 10` and the SDK's
negative preflight enforce the same prefix magnitude bound for unsigned
spellings. On both paths, classify each byte before that digit's range check.

## Caller-owned envelopes and diagnostics

Composition replaces only the numeric field loop, after existing validations.
Ledger validates whole-file UTF-8 before parsing rows. Log validates each row's
UTF-8 before controls/fields/status/path and the bytes field. NUL and bare CR
have earlier caller diagnostics. Those envelope checks can deliberately win
over a numeric prefix overflow and must not be removed or reordered.

Preserve `amount is empty`, `amount must be nonnegative decimal i64`, and
`amount exceeds i64 maximum`; log retains the corresponding `bytes` messages.
Path/line context, error effects, field slicing, streaming assembly, aggregate
overflow and all-or-nothing output remain caller responsibilities. Numeric
parsing alone establishes neither row UTF-8 validity nor application bounds.

Log status is not a general digits-only i64 field: accepting leading zeros or
an arbitrary width through a shared parser cannot replace its exact three-byte
and 100..599 checks. Likewise, a digits-only amount is not automatically a usize,
u8, safe allocation length or valid field width. Do not copy the quickstarts'
20-byte argv limit into the SDK or another application's row parser.

## Reopening and verification boundary

Reconsider a dedicated SDK wrapper if maintained applications repeat the lexical
guard and adapter, if real sign/error-order mistakes recur, or if actual API
adoption demonstrates a meaningful usability benefit over this composition.
Two frozen numeric loops establish the workload but do not settle that benefit.
No fixed consumer count or performance target is imposed.

Any later API proposal needs its own accepted contract under
[ADR 0328](../adr/0328-production-workload-capability-admission-gates.md), with
exact syntax/range errors and first-failure order. Implementation would require
SDK source/interface/version activation and independent full/subview, ownership,
Result, MIR/interpreter and Windows native evidence. A caller-only adoption would
instead verify its own diagnostics and envelope order without a new SDK version.

This task validates document links, whitespace, inspected source hashes and exact
commit scope only. It does not execute the helper sketch, rerun external round4
suites, repeat signed-parser acceptance, build native images or run performance
experiments. All external sources, toolchains, reports and receipts are preserved.

## Executed composition follow-up (2026-10-02)

The separate [digits-only quickstart](../../examples/digits_only_quickstart/README.md)
now executes the application-helper composition. Its
[retained receipt](../ops/evidence/digits-only-example-20261002.json) records 31
fixed inputs cross-checked with an arbitrary-precision oracle, direct and
`?`-propagated Results on full and nonzero-offset subviews, unchanged padded
owners, workspace check, MIR codec round trip, interpreter execution, and low
plus optimized Windows MSVC native execution. Each input has four Result checks.
The raw corpus includes invalid UTF-8, NUL, BOM and a Unicode digit. Both native
images return 0 with empty UTF-8 stdout/stderr; their hashes and raw invocations
are retained with source, SDK and in-process compiler identities.

This follow-up verifies the proposed composition and resolves its earlier
unexecuted-helper boundary. It does not demonstrate adoption in a maintained
application, admit another SDK API, change stage0.76, impose the fixture's
128-byte storage capacity on callers, or validate ledger/log envelope policies.
The frozen round4 sources and receipts remain unchanged; a future application
still needs its own line/field diagnostics and validation-order evidence.
