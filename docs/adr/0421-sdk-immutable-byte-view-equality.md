# ADR 0421: SDK immutable byte-view equality

- Status: Accepted; not implemented
- Date: 2026-10-02
- Depends on: ADR 0218, ADR 0414
- Workload evidence: [byte-view equality assessment](../architecture/byte-view-equality-assessment.md)

## Context and availability

The assessment identifies byte equality helpers in the nine preserved round4
projects and an identical loop with actual command/header call sites in the
maintained task-manager example. This is repeated source code, not a compiler
defect or a performance result. A local loop remains a valid alternative.

Accept the small ordinary library helper below. No language feature is needed.
SDK `0.0.0-stage0.76` does not currently provide it. Acceptance defines the future
SDK contract; it does not make the function available, reserve a version, change
the compiler or revalidate the historical applications. Implementation and
application adoption remain separate units.

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

This documentation-only acceptance leaves SDK stage0.76, language `/20`, runtime
ABI `/0`, grammar and target maturity unchanged. It does not create an SDK source
export or change the runtime/stdlib availability documentation.

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

No new implementation or native execution is claimed by this ADR. The
assessment's source hashes establish the inspected workload only. Implementation
completion requires its own current evidence before changing this ADR's status.

## Rollout and deferred scope

After implementation, consider the maintained task-manager example as the first
separate adoption unit. Replace only its equality loop/calls and verify the
existing command names, FYTASK1 header, field-validation ordering, diagnostics,
limits and storage/publication/cleanup behavior with the then-current compiler.
Its historical stage0.72 receipt is not current adoption proof. Preserve every
external round4 source, compiler and receipt.

Lexicographic comparison, stable sorting, full-range usize parsing, generic
binary codecs, constant-time comparisons, text normalization and field/row
validators remain separate proposals. Equality does not establish valid text,
canonical input, file identity, checksum integrity or authorization.
