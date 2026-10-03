# ADR 0437: Integer fixed-array views

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0430, 0433, 0436

## Decision

Admit shared and exclusive i32/u32/usize views of matching fixed arrays using
existing `ref T[]` spelling and array-view, length, slice, checked-index and
store operations. Lengths count elements. Empty subranges are valid; invalid
bounds trap before memory access. Exact element types cannot be reinterpreted.
Views can be parameters, returned derivatives and Option payloads. Short shared
reader calls reborrow exclusive views. Last use releases the owner loan.

Array size stays bounded at 4096. Unsized owned values, bool/aggregate views,
local-owner escape and shared stores stay rejected. Live derived/tagged views
block owner mutation/replacement. Exclusive parents and retained siblings
remain frozen according to ADR 0415, including disjoint ranges. There is no
new allocation, cleanup, foreign ABI or public raw-storage operation.

## Implementation

HIR owns the closed integer-view element set. Semantic unsized admission, MIR
actual-local operand checks, interpreter array-backed reads and C descriptor
selection use it. The MIR verifier matches owner/result element and borrow
kind, receiver definitions, usize bounds and actual store value types; relabeling
an integer view as bytes cannot bypass verification. C descriptors carry exact
const/mutable element pointers and uintptr_t length, emitted in fixed order.
The existing i64 MSVC workaround remains limited to i64.

This first unit admits fixed-array access only for the new types. Vec helpers
and for-in are the following unit. Existing u8/i64 behavior remains available.
As in ADR 0430-0436, language /20, SDK stage0.82, runtime /0, grammar and MIR
schemas remain unchanged for this additive executable-subset admission.
Compiler identities distinguish build/cache generations; no released compiler
compatibility or performance improvement is claimed.

## Verification

The Fairy example checks all three types, their extreme values, exclusive
subranges, shared reborrows, tagged returns, empty ranges, ordered bounds and
owner reuse. Compiler tests encode/decode MIR, run the interpreter, emit both
Windows/Linux C profiles and reject seven MIR mutations per new element type.
Semantic tests cover eight rejection families per type. Native execution covers
the example and end reads/stores, empty stores, reversed and past-end slices in
both Windows profiles. Public CLI check/build/run and identities are recorded in
[the receipt](../ops/evidence/integer-views-20261004.json).

Linux execution is deferred to the user's batch. No new PS1 runner, host
installation, performance experiment or whole-repository test is introduced.
