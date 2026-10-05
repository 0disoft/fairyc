# ADR 0469: Direct array field views

- Status: implemented; Windows dev/release native matrix and standalone public CLI validated; compiler round-trip, interpreter and both Windows MSVC and Linux C11 emission profiles pass; Linux actual execution deferred to the owner's batch; the same direct field view mechanism over an admitted plain-struct element field is extended by ADR 0472
- Date: 2026-10-05
- Related: ADRs 0415, 0455, 0457, 0461, 0462, 0463, 0464, 0465, 0467, 0468, 0470, 0471, 0472

## Decision

Admit direct element access and explicit `ref` / `ref mut` views over the bounded inline
scalar array fields admitted by ADR 0468 on plain non-`@repr(C)` structs. The element set
(`bool`, `u8`, `i32`, `i64`, `u32`, `usize`) and length (`1..=4096`) are unchanged, and the
field still receives no implicit initialization.

```fairy
struct Packet { u8[4] bytes; usize length; }
struct Envelope { Packet packet; }
fn void fill(ref mut u8[] bytes) {}
var Packet packet = Packet { .bytes = [1, 2, 3, 4], .length = 4 };
packet.bytes[0] = 1;                            // direct element store
packet.bytes[0] += 1;                           // checked compound element store
let ref u8 single = ref packet.bytes[0];        // element borrow
assert(*single == 2);
let ref mut u8 slot = ref mut packet.bytes[1];  // exclusive element borrow
*slot = 20;
let ref u8[] all = ref packet.bytes;            // whole-field view
let ref u8[] tail = ref packet.bytes[2..];      // half-open field subview
let ref u8[] empty = ref packet.bytes[4..];     // empty field subview
var Envelope envelope = Envelope { .packet = packet };
fill(ref mut envelope.packet.bytes[1..3]);      // nested owner path
```

The place root must resolve to a named local or to a by-value, `ref` or `ref mut` parameter,
and only through field and group projections. Call, temporary, raw-pointer, tagged,
indexed-aggregate and nested-slice roots stay closed, and ordinary array-field slicing without the
explicit `ref` / `ref mut` stays closed. A direct array field as a `for` source was closed in
this record and is superseded by ADR 0470, which admits direct array field iteration over this
same bounded field; binding a view first remains a valid alternative.

Whole-field reads and copies, like whole-struct copies, keep independent values (ADR 0468) and
may copy the array contents, so their cost scales with the field. Direct element access and
explicit views select the original field storage and reuse the existing field, array-view and
view operations, so no array copy or allocation is introduced. Nested arrays remain closed and
`@repr(C)` keeps rejecting array fields, and the same element access, view, subview and borrow
mechanism is extended to the ADR 0471 inline plain-struct array field by ADR 0472.

## Evaluation and lowering

`internal/semantic/array_field_views.go` recognizes the place root, unwraps grouping to the
field path, and requires the resolved owner to be a local or parameter whose type is a plain
view struct once a `ref` / `ref mut` layer is removed. It lowers to the existing
`array-view-borrow` operation whose child is the resolved field address.

Element reads become the existing `view-index` operation over that view; element borrows become
`view-index` plus the existing `view-element-borrow`; element stores and checked compound
stores rebase the existing mutable `view-index` target on the synthesized field view;
whole-field views are admitted where the `ref T[]` / `ref mut T[]` type is already expected,
as in a `let ref u8[] all = ref packet.bytes;` binding or a `ref mut u8[]` call argument;
field subviews run through `checkSliceRange` and the existing `view-slice` operation with
explicit, half-open, omitted or empty bounds. A whole-field view enforces the same named
plain-struct root, so an indexed-owner or tagged-owner full view is rejected rather than
silently borrowing a copy.

A matching shared or mutable fixed-array reference owner is now accepted by
`hir.ArrayViewBorrow`, and the C emitter and interpreter unwrap that owner type before
computing the element address. The emitted MIR is the same `array-view-borrow` plus view
operations used by named view bindings, direct element references and slice iteration
(ADRs 0464, 0465, 0467). No new grammar production, MIR operation, wire field, SDK export or
runtime helper is added.

## Borrows

The root owner is the whole struct, and neither the field name nor the index or range proves
disjointness. A shared view or element reference coexists with shared reads, shared borrows and
sibling shared views of the same root, but an owner write or exclusive borrow while it is live
is `E2101`, and an exclusive view or element reference blocks owner reads and writes and every
competing borrow. An exclusive view needs a mutable owner, so only a `var` local or a
`ref mut` parameter is accepted and everything else is `E1008`. Views and references whose
root is a local or by-value parameter cannot escape their owning frame, so returning them is
`E2102`; retained references and `Option<ref T[]>` aliases keep the owner loan alive through
their last use.

The index or bounds evaluate exactly once, and the original field storage is selected before the
index, bounds or right-hand side run. The runtime traps `bounds` before any element is accessed,
compound assignment uses the checked arithmetic of ADR 0466, and an index or bound that returns
early skips the remaining work through the terminating guards in `lowerIndexedAssignment` and
`lowerViewSlice`.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or
host flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, MIR/wire schemas, public
ABI and target maturity stay unchanged, and this is an additive admission over ADR 0468. Exact
compiler revision/executable identity separates cached generations.

## Verification and limits

Semantic boundary tests cover the immutable store and view (`E1008`), the width and
non-`usize` bound mismatches (`E1006`), local, by-value and element escapes (`E2102`), the
closed roots, and the `E2101` owner, sibling, bound, index, right-hand-side and returned or
tagged-alias conflicts. The compiler round-trip test drives the MIR codec, interpreter execution
and both the Windows MSVC and Linux C11 emission profiles, and rejects MIR mutations that drop
the `view-slice` bounds check, change the borrow owner type, or launder an immutable view into
a mutable one.

The semantic boundary tests, the compiler round-trip test and the MIR mutation rejections all
pass. The related suite passes 427 tests with no failures or skips; the Windows native suite passes 10
test entries with no skips (8 cases in dev and release, 16 outcomes, plus cache reuse); and the
standalone public CLI `check`/`build`/`run` recording passed 2 normal outcomes for dev and
release with the five rejected sources rejecting as expected (immutable `E1008`, width
`E1006`, owner loan `E2101`, local escape `E2102` and the unsupported temporary root).
Compiler build, `go vet`, formatting and diff checks pass. Linux actual execution remains
deferred to the owner's batch, so only C emission is verified for Linux. Evidence is recorded at
`docs/ops/evidence/array-field-view-20261005.json` with `.cli.json` and `.native.json`
companions.
