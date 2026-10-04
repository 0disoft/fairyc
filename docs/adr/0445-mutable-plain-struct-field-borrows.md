# ADR 0445: Mutable plain struct field borrows

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0391, 0401, 0414, 0415, 0441-0444

## Decision

Admit `ref mut owner.field` and nested/grouped field places from named mutable
locals or existing `ref mut` struct references. The root must satisfy ADR 0441's
nonempty acyclic plain struct graph; leaves are bool/u8/i32/i64/u32/usize.
A projected field may itself be a plain struct or one of those scalars.

```fairy
let ref mut Span span = ref mut token.position;
normalize(span);
let ref mut i64 start = ref mut span.start;
*start = 4000000000;
```

Select the actual stored field address, without copying an intermediate struct.
Scalar writes reuse admitted named-reference dereference assignment; aggregate
references use existing field stores. Whole-aggregate dereference replacement and
compound dereference assignment remain outside their existing contracts.

Immutable/shared roots, owned call results, indexed/pointer/dereference roots,
resources and nonplain graphs remain rejected. Shared field projection keeps
ADR 0391's separate boundary: explicit `ref mutable_reference.field` remains
closed. Reading a projected mutable reference through a shortened shared call
reborrow follows ADR 0414.

## Loans and origin

A field child retains its original whole-owner origin and exclusive loan, plus
an exclusive child generation. Every parent/parent-alias/root/sibling access
conflicts until the child's last proven use. A child alias or descendant retains
that generation, including through borrow-returning helpers and Option/Result.
Repeated siblings remain rejected even when fields are distinct. After the last
retained use, ordinary parent access and reference rebinding become usable again.
Return summaries propagate parameter provenance and child-generation status;
local/by-value owner escape remains rejected. No field-disjoint loan analysis,
new lifetime syntax, interior mutability or alias optimization promise is added.

## Implementation and compatibility

Semantic checking constructs nested `field-borrow` HIR with matching mutable
owner/result kinds. Flow reuses ADR 0415's child-generation checks and return
summaries. MIR reuses the existing `field-borrow` operation; the verifier checks
matching reference kinds, actual receiver local types, exact field declarations
and the mutable plain graph. C emission takes each real field address. The
interpreter retains a live scalar field cell for dereference loads/stores and
shares the stored struct referent for aggregate field stores. Owned copies remain
independent. Codec/cache transport uses existing shapes.

No grammar, MIR opcode/wire schema, SDK export, runtime C symbol, target maturity,
CI, dependency or runner change. SDK stage0.84, language /20 and runtime /0 stay.
Revision/executable identity separates compiler cache generations.

## Verification

`internal/compiler/testdata/mutable_field_borrow.fy` asserts direct/nested/grouped
projection, all six scalar types, >i32 values, writes through functions and scalar
aliases, descendant release, returned/tagged children, independent copied owners,
parent reuse and subsequent receiver rebinding. Fresh and codec-decoded MIR have
the same asserted outcome. Source rejection tests require E2101 for parent/root/
sibling/alias/call conflicts and E2102 for local escape. MIR mutations reject
forged receiver kinds/types, field types, paths and nonplain graphs. Windows
MSVC dev/release run the successful fixture and a checked overflow trap; source
session cold/warm artifact reuse is exercised. Linux runtime verification remains
in the owner's deferred batch. Review evidence is
`docs/ops/evidence/mutable-field-borrow-20261004.json`.
