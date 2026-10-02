# ADR 0421: SDK immutable byte-view equality

- Status: Implemented in SDK 0.0.0-stage0.77
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0414
- Workload evidence: [byte-view equality assessment](../architecture/byte-view-equality-assessment.md)

## Context and availability

The assessment identifies byte equality helpers in the nine preserved round4
projects and an identical loop with actual command/header call sites in the
maintained task-manager example. This is repeated source code, not a compiler
defect or a performance result. A local loop remains a valid alternative.

Accept the small ordinary library helper below. No language feature is needed.
SDK `0.0.0-stage0.76` did not provide it. SDK `0.0.0-stage0.77` implements the
accepted contract as ordinary Fairy source in `std::bytes`. Application adoption
remains a separate unit; historical applications are not revalidated here.

## Decision and exact contract

Add one pure Fairy function in `std::bytes`:

```text
pub fn bool equal(ref u8[] left, ref u8[] right);
```

For valid immutable byte views, return true if and only if the logical lengths
are equal and every corresponding `u8` value is equal. Otherwise return false.
The result depends on the viewed bytes and lengths, not backing-owner identity,
pointer identity, capacity, bytes outside a subview or an appended terminator.

| Inputs | Result |
| --- | --- |
| Two empty views, including empty subviews at different valid offsets | true |
| Empty and nonempty, in either argument order | false |
| Equal bytes from separate owners or from the same shared view | true |
| Same prefix but different lengths | false |
| Equal-length views differing at the first, interior or last byte | false |
| Equal byte sequences containing NUL, 128..255 or malformed UTF-8 | true |
| Different UTF-8 encodings or Unicode-normalization spellings | false whenever their bytes or lengths differ |

There is no UTF-8 validation, case folding, normalization, trimming, locale,
numeric conversion, hashing or text interpretation. NUL is an ordinary byte.
The function supplies no ordering or tri-state comparison result.

Return an ordinary `bool`; introduce no error enum, Result, special must-use
rule or sentinel encoding. Callers may use the result in existing expressions,
branches, assertions and returns. The function does not mutate input, retain or
return a borrow, allocate, perform I/O, call a foreign comparison symbol or create
a runtime-global state. It adds no heap failure path or new runtime ABI symbol.

## Source algorithm and cost

The accepted source implementation first compares the lengths. Unequal lengths
return false before any element indexing. For equal lengths, scan from index
zero, compare corresponding bytes, return false at the first mismatch, and return
true after the entire view matches. An empty pair performs no element read.

Worst-case work is linear in the equal-length input size; auxiliary storage is
constant and there is no library input-length cap. Callers retain their own work
and input budgets. Do not subtract bytes or lengths to encode a comparison result.
Index only while the index is less than the shared logical length; advance with
ordinary checked usize arithmetic. No unsafe scope, raw-pointer conversion,
native memcmp import, intrinsic or speculative out-of-view load is needed.

This source algorithm does not freeze generated instructions or promise timing.
Early return is deliberately permitted: this API is not a constant-time secret
comparison, authentication check or cryptographic verification operation. Any
timing-resistant API needs a separate name, contract and execution evidence.

## Borrowing, aliasing and bounds

Inputs are existing valid immutable views. Distinct owners, identical same-owner
views and overlapping shared subviews are permitted; all access is read-only.
The source must inspect only each argument's logical extent and preserve all
owner bytes, including surrounding sentinels.

Keep the existing view-construction bounds checks and borrow rules. Invalid
slicing can trap at the caller before this function runs; `equal` is not a
fallible slice-construction or arbitrary-pointer-validation API. Valid input
views need no new ordinary-trap path in the comparison loop.

Call-bounded mutable-to-shared reborrowing is governed by
[ADR 0414](0414-call-shared-reborrows.md), including earlier-argument protection
while later arguments are evaluated. The bool result carries no borrow; the
helper's call access ends on return. Other live caller views still retain their
own last-use restrictions. Mutation or movement conflicting with those loans
must remain rejected. Do not infer a new ability to hold mutable sibling views:
[ADR 0415](0415-exclusive-byte-subviews.md) retains that exclusion.

No source lifetime annotation, implicit slice equality operator, generic
collection comparison, text overload, mutable-input overload or public byte-view
C ABI is added. Existing private view transport remains compiler-owned.

## Versioning and implementation gate

The initial documentation-only acceptance left SDK stage0.76 unchanged.
Implementation activates SDK stage0.77 and updates the runtime/stdlib
availability documentation. Language `/20`, runtime ABI `/0`, grammar and
target maturity remain unchanged.

Implementation must add the function in the owning `std::bytes` source, activate
the appropriate next SDK version, and synchronize SDK discovery, public-interface
and lock/cache identity fixtures. An old bundled SDK must not resolve the new
export accidentally. No new foreign component or runtime ABI version is needed
for the accepted pure source implementation. Target maturity does not advance
because an ordinary library function was added.

## Required verification

Use an independent byte-sequence equality oracle instead of a second copy of
the Fairy loop. Cover both argument orders; empty/nonempty pairs; distinct
owners; same-owner identical and overlapping shared subviews; unequal lengths
and proper prefixes; first/interior/last mismatches; all 256 byte values;
embedded NUL, BOM and malformed UTF-8; nonzero offsets and boundary sentinels.
Verify original owners remain unchanged and bytes outside the view do not affect
the result. Bound fixture sizes independently of the API.

Exercise ordinary bool expressions and wrapper returns, shared call reborrowing,
last-use release and existing rejection of conflicting mutation/movement,
including argument-evaluation order. Do not weaken existing borrow diagnostics
to admit the new helper. Retain SDK discovery, loader, public-interface and
cache-identity checks appropriate to the export/version change.

Check the source, round-trip verified MIR through its artifact codec, execute
the interpreter, generate C for current target descriptions, and run Windows
MSVC low and optimized native images. Retain exact source/SDK/compiler/toolchain
identities, argv/cwd/stdout/stderr/exit and EXE SHA256. Decode native streams
explicitly as UTF-8. C generation alone is not Linux native execution. No
performance experiment or timing guarantee is required by this contract.

The assessment's source hashes establish the historical inspected workload only.
Current implementation evidence is recorded below.

## Implementation evidence

Implemented on 2026-10-02 as a safe, ordinary Fairy loop in
[`std::bytes`](../../internal/sdkstdlib/sources/std/bytes.fy).
SDK version/discovery is synchronized at stage0.77; the seven-module bundle,
language `/20`, runtime ABI `/0` and artifact schemas are unchanged.

[`bytes_equal_test.go`](../../internal/cli/bytes_equal_test.go) checks 16 pairs
against Go `bytes.Equal`, in both argument orders and through a bool-returning
wrapper. The pairs cover all 256 byte values in one binary vector,
first/interior/last mismatches, length/prefix differences, NUL, BOM, malformed
UTF-8 and distinct Unicode spellings. Distinct owner subviews have different
surrounding sentinels; every owner byte is checked after comparison. Additional
fixed cases cover identical and overlapping same-owner views, empty views at
zero/end offsets, literals, ordinary bool expressions, call-bounded shared
reborrowing, mutation after last use and a bool result surviving mutation.

Five negative cases retain exact `E2101` mutation/reborrow diagnostics and
`E2105` resource-move rejection. Existing shared-reborrow and exclusive-subview
semantic tests pass without compiler changes. SDK discovery and exact bundle
loader checks pass. Cold/warm cache execution passes. A modeled older bundle
with only this export removed changes source/interface/lock identity, misses
both cached packages and rejects a new `equal` call. This is a synthetic
compatibility fixture, not an execution of a historical compiler.

Source check, verified-MIR codec round trip, interpreter, C generation for
Linux GNU and Windows MSVC, and Windows MSVC low/optimized native execution pass.
Both native images return 0 with empty stdout/stderr; byte pipes are explicitly
validated/decoded as UTF-8. Toolchain: MSVC `14.51.36231`; compiler is the
in-process CLI compiler in the Go `go1.27.1` test executable, not the preserved
round4 compiler. Native argv/cwd/exit/output bytes and identities are retained in
`.cache/bytes-equal/receipt.json`; generated source and both EXEs are alongside it.

| Artifact | SHA256 |
| --- | --- |
| SDK `std/bytes.fy` | `2fdb25f2bc59de2bf9c9af150fe4cb81a9bd30fc2a18601616d8feb451df30f8` |
| SDK package source digest (`fysrc1`) | `a58fda1cc53ded24238d05e5a96b45ece30fd8947841a3f4ca18647eb2dee03d` |
| Generated Fairy input | `26ac096d3de441f29b2b7c56dc2876e9607fb79a85a0cbe376c7ecb68d7e8031` |
| Go test executable containing compiler | `210b0d8092017f22668c968ed1b046f3a56b170c6fec798d0e185870b3eff3a1` |
| Low EXE | `5eb826f7eafde6f6013081c8209e0b16f187c45dd356004a324d3ec5385718d2` |
| Optimized EXE | `c4ecc65a608f9491d864353a23d29cdf814bdc07a4057e6c6f34adf6837cc9dc` |

This SDK implementation evidence does not claim Linux native execution,
performance measurement or constant-time behavior. The separate application
adoption evidence follows below. Historical round4 sources, toolchains,
receipts and identity fixtures are preserved.

## Rollout and deferred scope

The maintained [task-manager example](../../examples/task_manager_cli/README.md)
completed the first separate adoption unit on 2026-10-02 in commit `cd2bee15`.
Example version `0.1.2` removes its local equality loop and calls `bytes::equal`
for the four command names, `--help` and FYTASK1 header. CLI arguments, field
validation order, diagnostics, limits and storage/publication/cleanup code are
preserved. Verification now requires an explicit compiler path and reviewed
SHA256; its historical stage0.72 receipt is not current adoption proof.

Fresh isolated-copy verification passed `check`, locked check, development and
release Windows native builds, and 126 exact-output/file-state cases (118 existing
cases plus eight observations of header first/last-byte mismatches, command
prefixes and case differences). Cases include Korean titles, restart, malformed
stores, limits, reserved sidecars and real OS replacement/read denial. An
incorrect compiler pin is rejected before fixture/build writes. Native readers
explicitly decode UTF-8; each case retains executable, argv, cwd, stdout, stderr,
exit code and before/after file states. Separate injected-storage-fault tests were
not repeated because storage code did not change; no new Linux, power-loss,
performance or constant-time verification is claimed.

The adoption compiler embeds revision `cd2d6a6e433ac3bcba3d55e21cf2597bcd26eac4`
with `modified: true` (the adoption edits were present when it was built), SDK
`0.0.0-stage0.77`, language `/20` and runtime `/0`. Exact source hashes and raw
compiler/build execution records are retained under
`.cache/task-byte-equality-20261002/`; the isolated project's
`verification-results.json` contains SDK/source digests and all 126 cases.

| Adoption artifact | SHA256 |
| --- | --- |
| Compiler EXE | `5c948d1c665b506cd10600db3f04ed11823e07bb2c7e97a1321dc4dc34bfa7e9` |
| Development EXE | `5a012ea4f9aaacbace6068e3a912803f2fa2ddaa90f29f477925f17188ba114f` |
| Release EXE | `651a6f04def6772778896f92f707b398f927db0cfcfec1542c8a9b779ee98d7e` |
| Structured verification results | `c4f2b88d85a54a2f02142fc2bdebb0499079c179945e83b125f63498d949e8a8` |

Every external round4 source, compiler and receipt remains preserved. This
adoption does not revalidate or update those historical projects.

Lexicographic comparison, stable sorting, full-range usize parsing, generic
binary codecs, constant-time comparisons, text normalization and field/row
validators remain separate proposals. Equality does not establish valid text,
canonical input, file identity, checksum integrity or authorization.
