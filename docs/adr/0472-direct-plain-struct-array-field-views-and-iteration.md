# ADR 0472: Direct plain-struct array field views and iteration

- Status: implemented; interpreter, MIR codec and both Windows MSVC and Linux C11 emission profiles pass; Windows dev/release native matrix and public CLI `check`/`build`/`run` with direct EXEs pass; Linux actual execution deferred to the owner's batch
- Date: 2026-10-05
- Related: ADRs 0415, 0455, 0462, 0463, 0464, 0465, 0467, 0468, 0469, 0470, 0471

## Decision

Admit direct element access, explicit `ref` / `ref mut` views and subviews, and `for` iteration
over the bounded inline plain-struct array fields admitted by ADR 0471 on plain non-`@repr(C)`
structs. The element is the admitted scalar-leaf plain struct (ADR 0462, ADR 0471): `bool`, `u8`,
`i32`, `i64`, `u32` and `usize` leaves with nested ordinary structs and no array field anywhere in
its field graph. The length bound (`1..=4096`), exact nominal identity and absent implicit
initialization are unchanged.

```fairy
struct APage { ZToken[2] tokens; usize used; }
struct ASection { APage page; }
struct ZToken { u32 kind; u8 flags; }

fn void fill(ref mut ZToken[] tokens) {}

var APage page = APage { .tokens = [ZToken { .kind = 1, .flags = 0 }; 2], .used = 2 };
page.tokens[0].kind = 2;                              // direct scalar field store
page.tokens[1] = ZToken { .kind = 3, .flags = 1 };    // whole element replacement
let ref ZToken[] all = ref page.tokens;               // shared whole-field view
let ref mut ZToken[] mut_all = ref mut page.tokens;   // exclusive whole-field view
let ref ZToken[] tail = ref page.tokens[0..];         // shared field subview
let ref ZToken first = ref page.tokens[0];            // shared element reference
let ref mut u32 kind = ref mut page.tokens[1].kind;   // exclusive scalar field reference

var ASection section = ASection {
    .page = APage { .tokens = [ZToken { .kind = 0, .flags = 0 }; 2], .used = 0 },
};
fill(ref mut section.page.tokens[0..2]);              // nested plain parent path

for (let ZToken item in page.tokens) { assert(item.kind >= 0); }         // copied item
for (let ref ZToken item in page.tokens[..]) { assert(item.kind >= 0); }  // shared item
for (let ref mut ZToken item in page.tokens) { item.kind += 1; }          // exclusive item
```

The place root must resolve to a named local or to a by-value, `ref` or `ref mut` parameter whose
type is a plain non-`@repr(C)` struct, and only field and group projections are allowed on the
path, so a nested plain parent path such as `section.page.tokens` is supported. Call, temporary,
raw-pointer, tagged, indexed-aggregate and nested-slice roots stay closed. The whole-field borrow
must be the unsized `ref T[]` / `ref mut T[]`, so a sized `ref T[N]` / `ref mut T[N]` field
binding stays closed, and ordinary array-field slicing without the explicit `ref` / `ref mut`
stays closed outside the explicit field view and the direct `for` source.

A whole-field value read keeps the independent copy of ADR 0471 and may copy the array contents,
so its cost scales with the field. A direct element value read selects the original field
descriptor and then copies only the selected element value, so it copies that element but never
reads or copies the whole array. Element field projections, element references, explicit views and
subviews likewise select the original field storage and reuse the existing array-view,
`view-index`, `view-element-borrow` and `view-slice` operations, so they add no whole-array copy or
allocation, and iteration builds its descriptor from the original storage without a whole-array
copy or allocation. A copied `let T item` loop item still copies the selected element value, as the
existing copied iteration does. Nested arrays, cyclic element graphs and `@repr(C)` array fields
stay rejected.

## Evaluation and lowering

`internal/semantic/array_field_views.go` recognizes the place root, unwraps grouping to the field
path, and requires the resolved owner to be a plain non-`@repr(C)` struct once a `ref` / `ref mut`
layer is removed. It lowers to the existing `array-view-borrow` operation whose child is the
resolved field address.

Element reads and element field projections become the existing `view-index` operation over that
view; element references become `view-index` plus the existing `view-element-borrow`; element
stores, whole-element replacement and checked compound scalar field stores rebase the existing
mutable `view-index` target on the synthesized field view; whole-field views are admitted where
the `ref T[]` / `ref mut T[]` type is already expected, as in a `let ref ZToken[] all = ref
page.tokens;` binding or a `ref mut ZToken[]` call argument; field subviews run through
`checkSliceRange` and the existing `view-slice` operation with explicit, half-open, omitted or
empty bounds. A whole-field view enforces the same named plain-struct root, so an indexed-owner or
tagged-owner full view is rejected rather than silently borrowing a copy.

Iteration reuses the implicit descriptor generation of the named-array and named-view loops
(ADR 0457, 0465, 0470) over the explicit field view, so the loop selects the original field
storage without a whole-array copy or allocation and never copies the whole field into a temporary;
a copied `let T item` still copies that one element value.

No new grammar production, MIR operation, wire field, artifact schema, SDK export or runtime
helper is added.

## Borrows

The root owner is the whole struct, and neither the field name nor the index or range proves
disjointness, so there is no field or sibling disjointness. A shared view or element reference
coexists with shared reads, shared borrows and sibling shared views of the same root, but an owner
write or exclusive borrow while it is live is `E2101`, and an exclusive view or element reference
blocks owner reads and writes and every competing borrow. An exclusive view, element reference or
loop item needs named mutable storage: a `var` local, a `ref mut` parameter, or a named local or
parameter holding a `ref mut` owner; everything else is `E1008`. Views and references whose root
is a local or by-value parameter cannot escape their owning frame, so returning them is `E2102`;
retained references and
`Option<ref T[]>` aliases keep the owner loan alive through their last use.

The index or bounds evaluate exactly once, in written left-to-right order, and the original field
storage is selected before the index, bounds or right-hand side run. The runtime traps `bounds`
before any element is accessed, compound assignment uses the checked arithmetic of ADR 0466, and
an index or bound that returns early skips the remaining work through the terminating guards in
`lowerIndexedAssignment` and `lowerViewSlice`. The iteration item set is exactly the field's plain
struct element, copied `let T item`, shared `let ref T item` or exclusive `let ref mut T item`,
with explicit, half-open or omitted bounds; `break`, `continue`, `return` and local or by-value
escape rules are inherited unchanged.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or
host flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, MIR/wire schemas, public ABI and
target maturity stay unchanged, and this is an additive admission over ADR 0471 that reuses the
ADR 0469 and ADR 0470 mechanism on the plain-struct element. Direct element value indexing now
retains the owner while evaluating the index: an index expression that writes that same owner is
rejected with `E2101`, rather than indexing the prior whole-field copy from the ADR 0471 unit.
Exact compiler revision/executable identity separates cached generations.

## Verification and limits

Verification is implemented and passing. The source matrix
`internal/compiler/testdata/nominal_array_field_views.fy` covers a nested scalar-only `Token` and
`Meta`, a `Page` field and an `Envelope` parent, original/copy independence, ordered once-only
selectors, shared and mutable views and element access, an explicit leaf reference, calls, return
and `Option` aliases, copied/shared/exclusive loops, empty ranges, and
`break`/`continue`/`return`/Result propagation; it drives the interpreter, the MIR codec and both
the Windows MSVC and Linux C11 emission profiles.
`TestNominalArrayFieldViewStorageIdentity` verifies the field-address descriptor path and rejects a
whole-array field read or an array-index copy. The semantic boundary suite contributes 39 negative
cases and confirms the fixed raw field references and the other closed forms stay rejected.

The Windows native matrix passes 14 native Go test entries (24 dev/release outcomes), including
`bounds`, reversed range, checked overflow and index or bound early return. The public CLI
`check`/`build`/`run --locked` passes in dev and release with the direct EXEs, and six public
diagnostics reject as expected: `E1006`, `E1008`, `E2101`, `E2102`, the unsupported fixed field
reference and the indexed owner. The selected unit suite passes 253 test entries with 0 failures,
skips or build failures, the public raw commands' success and diagnostic assertions pass, `go vet`
passes with an empty log, and formatting and diff checks pass. Actual Linux execution stays deferred
to the owner's batch, so Linux is verified by C emission only.

Evidence is recorded at `docs/ops/evidence/nominal-array-field-views-20261005.json`. No whole-suite
result is claimed here.
