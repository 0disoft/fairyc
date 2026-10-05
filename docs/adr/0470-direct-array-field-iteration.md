# ADR 0470: Direct array field iteration

- Status: implemented; Windows dev/release native, related suite and standalone public CLI verified; Linux C emission only, actual execution deferred to the owner's batch
- Date: 2026-10-05
- Related: ADRs 0415, 0457, 0458, 0461, 0462, 0465, 0467, 0468, 0469

## Decision

Admit `for` iteration whose source is a bounded inline scalar array field admitted by
ADR 0468, reached from a named plain non-`@repr(C)` struct local or parameter through field
and group projections only:

```fairy
struct Packet { u8[4] bytes; usize length; }
struct Envelope { Packet packet; }
fn void scan(ref Packet packet) {
    for (let u8 byte in packet.bytes) { assert(byte >= 0); }
    for (let ref u8 byte in packet.bytes[1..3]) { assert(*byte >= 0); }
}
var Envelope envelope = Envelope { .packet = Packet { .bytes = [1; 4], .length = 4 } };
for (let ref mut u8 byte in envelope.packet.bytes[..]) { *byte = 0; }
```

The item set is exactly the field's scalar element type: `bool`, `u8`, `i32`, `i64`,
`u32` and `usize` (ADR 0468), copied `let T item`, shared `let ref T item` or exclusive
`let ref mut T item`. Bounds may be explicit, half-open or omitted, matching the field subview
form of ADR 0469, and the source stays a single field subview.

The place root must resolve to a named local or to a by-value, `ref` or `ref mut` parameter
whose type is a plain non-`@repr(C)` struct, and only field and group projections are allowed on
the path. Temporary, call, indexed-aggregate, tagged, raw-pointer and nested-slice roots stay
closed. Ordinary array-field slicing stays closed outside the direct `for` source and the explicit
`ref` / `ref mut` field view of ADR 0469, so `packet.bytes[lo..hi]` is admitted only as this
loop's source or through an explicit field view. A non-`var` local or a by-value parameter supports
copied and shared items only; exclusive items
require a mutable owner, so only a `var` local or a `ref mut` parameter is accepted and
everything else is `E1008`.

The loop reuses the implicit descriptor generation of the named-array and named-view loops
(ADR 0457, 0465) over the explicit field view of ADR 0469, so it selects the original field
storage without a copy or allocation. The whole struct is the loan owner and neither the field
name nor the range proves disjointness: a shared loop holds a shared loan that blocks owner writes
and exclusive borrows through exit, and an exclusive loop holds an exclusive loan that blocks
owner reads and writes and every competing borrow, freezing its parent descriptor and aliases for
the written loop's duration under ADR 0415. `break`, `continue`, `return`, Result propagation
and local or by-value escape rules are inherited unchanged from the existing loops.

## Evaluation and lowering

The written slice evaluates its receiver, explicit `start` and explicit `end` exactly once, in
that order, before the length snapshot and every element access; an omitted `start` is zero and an
omitted `end` reads the selected descriptor's length. The runtime checks `start <= end <= len`
before the loop body, so omitted bounds and empty subviews are valid and an out-of-range slice
traps with `bounds` before any element is addressed. Length is captured once and indices advance
in ascending order, and an index or bound that returns early skips the remaining work through the
existing terminating `lowerViewSlice` guard.

The resolved field path lowers to the existing `array-view-borrow` over the field address, the
written slice to the existing `view-slice`, and the loop to the shared `prepareForDescriptor`
header generation and element address operations. No new grammar production, MIR operation, wire
field, artifact schema, SDK export, runtime symbol or host flag is added.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or
host flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, MIR/wire schemas, public ABI
and target maturity stay unchanged, and this is an additive admission over ADR 0468 and ADR 0469.
Exact compiler revision/executable identity separates cached generations.

## Verification and limits

Implementation and validation pass. The related suite passes 313 test entries with no failures or
skips (`.cache/array-field-for-related-final.jsonl`); the Windows native suite passes 9 test
entries with no failures or skips (7 cases in dev and release for 14 actual outcomes plus a cache
test); and the standalone public CLI `check`/`build`/`run` recording passed 2 normal outcomes
for dev and release and rejected five sources: an immutable exclusive source (`E1008`), a width
mismatch (`E1006`), an owner write (`E2101`), a local escape (`E2102`) and an unsupported
temporary root.

The matrix covers all six scalar element types, whole-field and grouped/nested field paths, bounds
that evaluate once with a changing end variable, empty slices, `break`/`continue`/function
`return`, a `ref` parameter item returned beyond the loop, `?` propagation and original owner
access after the loop; the native cases cover past-end, reversed, start-past-end, checked i64
overflow, start-return and end-return. Compiler round-trip tests confirm retained-AST deterministic
encoded MIR, MIR codec and interpreter round-trip, and both the Windows MSVC and Linux C emission
profiles; a MIR mutation that drops the bounds check is rejected by the verifier, codec and backend.
`go build`, `go vet`, formatting and diff checks pass. Linux actual
execution remains deferred to the owner's batch, so only C emission is verified for Linux. No
benchmark, installation, CI change, push or release is included. Evidence is recorded at
`docs/ops/evidence/array-field-for-20261005.json` with `.cli.json` and `.native.json`
companions.
