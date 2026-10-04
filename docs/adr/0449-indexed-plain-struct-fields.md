# ADR 0449: Indexed plain struct field places

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0441, 0445, 0446, 0448

## Decision

Named plain nominal views support `items[i].field = value`, checked scalar
compound assignment, nested field paths and exact nominal nested replacement.
A mutable descriptor is required for stores. `ref items[i].field` projects a
shared address from a shared or mutable descriptor; `ref mut items[i].field`
requires a mutable descriptor. These addresses refer to stored elements rather
than copied indexed values. Ordinary `items[i]` still copies the whole value.

```fairy
let ref mut Token[] items = vec::view_mut<Token>(ref mut owner);
items[0].value = 5000000000;
items[0].position.start += 3;
let ref mut Span position = ref mut items[0].position;
rebase(position, origin);
let ref i64 number = ref items[0].value;
assert(*number == 5000000000);
```

The owner remains held. Mutable field children freeze the parent descriptor and
its aliases using ADR 0415 generations; fields and indices do not prove
separation. Shared children prevent writes while live. Returned and Option
wrapped children retain origin and loan information. Last use releases the
hold; local escape and shared-to-exclusive conversion remain rejected.

The place root must be a named nominal view, possibly grouped. Temporary view
calls, raw pointers, fixed-array struct conversion, owning slices, direct
`ref items[i]` element borrows and mutable nominal for-in remain closed.
The admitted graph is unchanged: actual nonempty acyclic structs with scalar
leaves, with no resource, pointer, array, borrowed or tagged field.

## Evaluation and lowering

Receiver selection precedes index evaluation. The descriptor is saved before an
effectful index; the index runs once and its bounds are checked before taking an
address. Field stores capture that address before the RHS. Compound stores also
read the prior scalar before RHS evaluation without reevaluating the index.
An in-range terminating RHS has no store; an out-of-range index traps before the
RHS. Structural owner mutation remains blocked throughout bounds and RHS.
Descriptor rebinding preserves the earlier selected owner and address.

HIR/MIR add `view-element-borrow`: a checked nominal descriptor/index produces
an exact nominal shared or mutable reference. It requires a dominating matching
`view-index-check` and its successful branch. Actual receiver and usize index
local definitions are checked, independently of operand annotations. The
nominal graph, result referent, mutability and closed operation shape are checked
before codec or backend consumption. Existing field-borrow/store operations
then use the selected reference; no aggregate copy becomes a writable place.

C emission takes a typed pointer to the indexed stored element. Existing
Windows mutable-view reads/stores retain their ADR 0433 policy; no warning
suppression, runtime symbol or optimization flag is introduced. Interpreter
array-backed nominal descriptors borrow the stored aggregate itself while
ordinary indexed loads stay independent copies. Host-storage Vec allocation
interpretation is not added.

## Verification and compatibility

`internal/compiler/testdata/indexed_struct_fields.fy` exercises direct/nested
stores, nested replacement and copy independence, scalar compound assignment,
shared/mutable field references, function and Option returns, last use and
single index evaluation. Windows dev/release run that fixture, bounds before
RHS, RHS assertion, checked overflow, end/huge borrowed indices and independent
index/RHS descriptor rebinding. Rejections cover parent, alias, sibling field
and index, shared children, owner mutation, returned/tagged holds, effectful
bounds/RHS, shared-to-exclusive conversion, temporary views and local escape.

MIR mutations reject missing checks, wrong nominal/result kinds, forged receiver
and index definitions, nonplain graphs and missing fields; codec rejection is
also checked. Both Windows and Linux C emission, interpreter reference/copy
separation and cold/warm cache reuse are exercised. Related tests and Go vet
pass. Evidence: `docs/ops/evidence/indexed-struct-fields-20261004.json`.

The operation vocabulary grows within existing MIR/2 record fields and private
codec envelopes; no wire fields, grammar, SDK exports, runtime ABI or target
maturity change. Compiler revision/executable identity separates cache
populations. SDK stage0.85, language /20 and runtime /0 stay unchanged.
Linux actual execution remains deferred to the owner's batch. No full-repository
suite, benchmark, installation, CI/runner change, push or release is included.
