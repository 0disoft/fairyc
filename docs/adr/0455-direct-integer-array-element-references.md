# ADR 0455: Direct integer array element references

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0453, 0454

## Decision

Admit `ref data[i]` and `ref mut data[i]` for named fixed arrays of u8,
i32, i64, u32 and usize. The reference points into the original array storage,
not an indexed value copy. Grouped places and explicit call arguments work.
Mutable references require a mutable array binding; immutable arrays can yield
shared references. Element types must match exactly.

```fairy
var i64[3] data = [1, 4000000000, 3];
let ref mut i64 item = ref mut data[1];
*item = *item + 1;
assert(data[1] == 4000000001);
```

The direct receiver is a named local or by-value parameter with an admitted
fixed-array shape. By-value parameters occupy local storage, so their references
cannot be returned. Call/literal/slice roots, bool arrays, struct arrays, array
fields, raw pointers and empty arrays remain outside this admission. Direct
array for-in iteration remains closed; use a named view for that operation.

## Lifetime and evaluation

The whole array is the loan owner. Exclusive element references freeze direct
owner reads/writes and view aliases until last use; shared references allow
shared reads but block mutation while retained. Different indices do not prove
disjointness. Reference copies, returned references from borrowed scalar
parameters and tagged payloads retain their selected origin and generation.
Direct references into local arrays cannot escape, including through returns.

Select the original storage before evaluating the index exactly once. During
index evaluation, hold the selected owner against replacement, element writes
and competing borrows. Ordinary reads such as its fixed length are allowed
before the element reference is created. Checked bounds dominate address
creation. A failing index creates no reference or continuation. Scalar stores
retain the existing receiver snapshot before RHS effects; rebinding a reference
in its RHS does not retarget the write. RHS traps perform no store.

## Implementation

`internal/semantic/indexed_place.go` admits the exact named array place and
prepares an internal `array-view-borrow` followed by `view-element-borrow`.
This conversion lends original storage rather than evaluating the array index
as a value. Source Flow establishes the named array's local origin and loan
before effectful indices; exclusive results carry the existing child generation
that freezes parent/aliases. Direct borrow creation uses the same owner-conflict
check as explicit whole-array borrows. No second alias model is introduced.

MIR, codec, interpreter and typed C emission reuse the existing array descriptor,
checked view index and scalar address operations. The MIR verifier checks actual
local owner types for every array-view conversion, including u8. Forged operand
annotations cannot change the array element width. No new syntax node, MIR
operation, wire field, SDK API, C runtime helper or host flag is added.

## Verification and limits

Source tests cover the five types, shared/exclusive stored updates, grouped
places, explicit arguments, shared read coexistence, independent copied values,
last-use release, view aliases, copied-reference aliases, retained call/tagged
origins, competing siblings, RHS owner access, and index effects. Negatives
cover immutable mutable-borrows, mismatched widths, bool/literal/call roots,
shared writes and local/by-value-parameter escape with E2102.

The fixture `internal/compiler/testdata/array_element_borrow.fy` runs through
verified MIR codec round-trip and the interpreter, and emits through Windows
MSVC and Linux C profiles. It checks the five types, large values, immutable
shared borrows, function returns, tags, one-time index effects and RHS reference
rebinding. Verifier/codec/backend mutations reject missing bounds, forged array
owners/widths, shared-to-exclusive receivers and wrong item types.

Windows dev/release native execution covers that matrix, first/last elements,
end/huge out-of-range indices, index/RHS assertions and checked i64 overflow.
Public CLI check/build/run also exercises the matrix, rejected parent/index
mutation and local escape, plus the `examples/integer_array_element_refs`
byte-histogram example. Related regressions and Go vet pass. Evidence and raw
execution/artifact hashes: `docs/ops/evidence/array-element-borrow-20261004.json`.

SDK stage0.85, language /20, runtime /0 and target maturity remain unchanged.
Linux actual execution remains deferred to the owner's batch. No benchmark,
installation, CI change, push or release is included.
