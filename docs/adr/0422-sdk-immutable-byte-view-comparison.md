# ADR 0422: SDK immutable byte-view comparison

- Status: Implemented in SDK 0.0.0-stage0.78
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0414, ADR 0421
- Workload evidence: [byte-view comparison assessment](../architecture/byte-view-comparison-assessment.md)

## Context and availability

The assessment identifies repeated unsigned byte/prefix ordering in the ledger,
resource pack, persistent KV and maintained lexer. Pack and KV share an identical
comparison body; the ledger and lexer express the same ordering as boolean
helpers. The shared need is byte ordering, not their application sorting policy.
A local loop remains valid; no compiler defect or performance claim motivates
this decision.

Accept one ordinary safe Fairy library helper. The inspected SDK stage0.77 had
`bytes::equal` but no public `bytes::compare`. SDK stage0.78 implements the
accepted contract; application adoption remains a separate unit.

## Decision and exact contract

Add this function to `std::bytes`:

```text
pub fn i32 compare(ref u8[] left, ref u8[] right);
```

For valid immutable byte views, return exactly `-1`, `0` or `1`. Compare
corresponding bytes from index zero as unsigned `u8` values. The first unequal
pair determines the order. If the common prefix is equal, the shorter view sorts
first; equal lengths and equal bytes return zero.

| Inputs | Result |
| --- | ---: |
| Two empty views | 0 |
| Empty left, nonempty right | -1 |
| Nonempty left, empty right | 1 |
| `61` versus `61 00` | -1 |
| `61 FF` versus `62 00` | -1 |
| `80` versus `7F` | 1 |
| `00 FF` versus `00 FF` | 0 |
| `31 30` versus `32` | -1 |

Equality depends only on viewed bytes and logical lengths: zero is equivalent
to `bytes::equal(left, right)`. Swapping arguments negates the result; the strict
less-than relation is transitive. Owner identity, capacity, surrounding bytes
and implicit terminators do not participate. NUL and invalid UTF-8 are data.

There is no text validation, Unicode normalization, case folding, locale or
numeric interpretation. Callers retain their input validation and work limits.
Return `i32` to match the existing tri-state consumers without a new public
`Ordering` type. A boolean-only helper cannot distinguish equality from order
in one call. Do not return byte or length differences: their magnitude is not
the contract and subtraction introduces unnecessary overflow/underflow risk.

## Source algorithm, ownership and cost

Use an ordinary guarded Fairy loop over the common prefix, explicit byte
less/greater branches and then length branches. Return constant `i32` values;
index only while both lengths permit access and use checked `usize` increment.
For valid views, the loop needs no new ordinary-trap path. Worst-case work is
linear in the common-prefix length with constant auxiliary storage. Early exit
is permitted; this is not a constant-time or cryptographic comparison.

No allocation, I/O, mutation, retained borrow, error type, Result, unsafe scope,
raw pointer, foreign `memcmp`, compiler intrinsic or new runtime symbol is
needed. Keep `bytes::equal` and its length-first implementation unchanged.

Distinct owners, identical same-owner views and overlapping shared subviews are
permitted. Inspect only logical view extents and preserve complete owner bytes.
Existing bounds checks apply before the call; this is not a slice constructor
or pointer validator. [ADR 0414](0414-call-shared-reborrows.md) governs shared
reborrowing, earlier-argument protection and last-use release. The scalar result
carries no borrow; other live caller loans retain their restrictions. Existing
conflicting mutation/movement rejection remains. This does not admit mutable
sibling views excluded by [ADR 0415](0415-exclusive-byte-subviews.md).

## Implementation and verification

The documentation-only acceptance reserved no SDK version. Implementation
activates SDK stage0.78. Synchronize SDK export discovery, source/public-interface
identity and relevant lock/cache fixtures. Older bundles must not resolve the
new export accidentally. Grammar, language `/20`, runtime ABI `/0`, private
view transport and target maturity remain unchanged.

Use an independent unsigned byte-order oracle such as Go `bytes.Compare`,
normalized to the exact three-value contract. Cover both argument orders,
empty/proper-prefix pairs, first/interior/last mismatch, all byte values, NUL,
BOM, malformed UTF-8, different Unicode spellings and long common prefixes.
Check equality consistency, antisymmetry and transitivity over bounded data.

Exercise distinct owners, identical/overlapping shared subviews, nonzero
offsets, zero/end empty views and surrounding sentinels. Check every owner byte
after comparison. Cover ordinary scalar returns/branches, call reborrowing,
last-use release and existing mutation/move/argument-evaluation rejections.

Check source, verified-MIR codec round trip, interpreter, C generation and
Windows MSVC low/optimized native execution. Retain exact source/SDK/compiler
and toolchain identities, argv/cwd/stdout/stderr/exit and EXE SHA256. Decode
native streams explicitly as UTF-8. C generation alone is not Linux native
execution. No benchmark or physical-failure experiment is needed.

## Adoption and non-goals

Adoption follows implementation verification. In the maintained lexer, replace
only token-byte ordering; retain start/end/code/exit-repair priority and final
`ai < bi` tie-breaking. Compare full diagnostic output and deterministic tie
cases. Preserve external round4 projects, pinned compilers and past receipts.

No sorting API, callback protocol, generic collection comparator, text overload,
numeric parser, binary codec or timing guarantee is admitted. No grammar,
compiler behavior, database or runner configuration changes are required.

## Implementation evidence

Implemented on 2026-10-02 as a guarded safe loop in
[`std::bytes`](../../internal/sdkstdlib/sources/std/bytes.fy). SDK discovery is
stage0.78; language `/20`, runtime ABI `/0`, the seven-module bundle and artifact
schemas are unchanged. `equal` retains its existing length-first source body.

[`bytes_compare_test.go`](../../internal/cli/bytes_compare_test.go) checks 22
oracle pairs in both directions, through direct calls and scalar-returning
wrappers. Cases include long prefixes, first/interior/last differences, binary
data, text spellings and proper prefixes. A separate array of all 256 byte
values is compared against byte 128 using Go-derived expectations. All 56
increasing triples from an independently ordered eight-sequence domain check
transitivity. Equality consistency, shared overlap, zero/end empty views,
different sentinels and complete owner preservation are checked.

Source check, verified-MIR codec round trip, interpreter, Linux/Windows C
generation and Windows MSVC low/optimized native execution pass for these
byte-view inputs. `.cache/bytes-compare/receipt.json` and `main.fy` retain the
oracle inputs, source/SDK identities, actual in-process compiler executable
SHA256, toolchain identity, EXE SHA256 and raw UTF-8 execution records. Native
exit is zero with empty stdout/stderr. Linux native execution was not run.

Five negative cases retain `E2101` mutation/reborrow diagnostics and `E2105`
resource-move rejection. The existing shared-reborrow/exclusive-subview semantic
tests, equality checks and SDK discovery tests pass. Cold/warm cached comparison
execution passes. Removing only `compare` in a modeled prior bundle changes
source/interface/lock identities, misses both cached packages and rejects the
export. This is a synthetic fixture, not a historical compiler execution.

## Historical compiler limitation found during verification

Fixed on 2026-10-03 by [ADR 0423](0423-lossless-mir-static-bytes.md). The minimal
source below now passes codec round trip and native execution; comparison
verification also includes high-byte literals. The following records the
original diagnosis and preserved evidence.

The previous JSON MIR codec serialized raw byte-literal data as a Go string.
An invalid UTF-8 literal byte is replaced during encoding: `80` becomes
`EF BF BD`. This occurs independently of `compare`; the minimal source below
passed direct checked-MIR interpretation but trapped after encode/decode:

```text
module main;
import std::bytes as bytes;
fn i32 main() {
    let u8[1] expected = [128];
    assert(bytes::equal(b"\x80", ref expected));
    return 0;
}
```

The diagnostic source, encoded MIR, diagnostic Go test source and exact
compiler/SDK/outcome evidence are retained under `.cache/bytes-compare/` as
`codec-byte-literal-repro*`. The temporary diagnostic test is not installed as
a passing regression for broken behavior. The subsequent fix and regression
tests are recorded in ADR 0423.

The original comparison verification represented arbitrary raw bytes with numeric arrays
and subviews, including malformed UTF-8. Thus the helper's byte-order contract
was verified, but that unit did not establish preservation of invalid UTF-8
byte literals across MIR serialization. ADR 0423 supplies that separate evidence.
