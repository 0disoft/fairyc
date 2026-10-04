# ADR 0446: Shared fields from mutable struct references

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0391, 0414, 0415, 0441, 0445

## Decision

Admit explicit `ref parent.field` and nested/grouped stored field places when
`parent` is a named `ref mut` to an ADR 0441 nonempty acyclic plain struct graph.
The field is a plain struct or bool/u8/i32/i64/u32/usize leaf. Existing ADR 0391
shared projection from owned places and shared references is unchanged.

```fairy
fn void adjust(ref mut Token token) {
    let ref Span position = ref token.position;
    inspect(position);
    token.position.start += 1;
}
```

This is an explicit shared borrow of a selected field. Implicit whole-reference
binding/assignment/return conversion, resources and nonplain mutable roots,
owned or borrowed call-result roots, indexed/pointer/dereference roots, shared
to mutable conversion, and disjoint exclusive sibling loans remain closed.
Resource field privacy remains enforced. No intermediate struct is copied.

## Lifetime and access

The shared child retains the selected whole-owner origin and root loans. While
it, its alias, its descendant or a returned/tagged child remains live, conflicting
writes through the parent, parent aliases, siblings and the original owner are
E2101. Shared reads and shared sibling projections may coexist. Last-use release
allows parent referent writes again. Earlier call arguments remain held during
later argument evaluation; an exclusive argument conflicts in either order.
Local escape remains E2102 and multiple parameter origins remain E2103.

Reference descriptor rebinding keeps the existing contract: a `var ref mut`
may select another owner without retargeting an already-created child. The child
continues protecting the original referent and origin. Access through an existing
exclusive field child also retains ADR 0415's ancestor-generation restrictions.
A function requiring `ref mut` still requests exclusive call access, even if its
body merely returns a shared field; obtain simultaneous shared fields explicitly
or call a helper taking `ref T` instead.

## Implementation and compatibility

Semantic checking first applies the legacy shared projection. For the new
admitted root it rebuilds the HIR field path with an exact `shared-reborrow` at
the named mutable root, then projects shared field addresses. Original mutable
HIR children/type facts stay unchanged. Source Flow uses the existing shared
parent-loan retention and reference-origin conflict checks. Return summaries
preserve the selected argument origin through helpers and Option/Result.

MIR remains `shared-reborrow` followed by shared/shared `field-borrow` operations.
Existing verifiers still reject mismatched reference kinds, forged local types
and wrong fields. Existing C emission and the interpreter already transport these
addresses and live field cells. No backend/verifier rule is weakened. Codec and
package cache transport retain existing operation shapes.

No grammar, MIR opcode/wire schema, runtime C symbol, SDK source/export, target
maturity, CI, runner or dependency change. SDK stage0.84, language /20 and runtime
/0 remain; exact compiler revision/executable identity separates cache generations.

## Verification and remaining limitation

`internal/compiler/testdata/shared_mutable_field_borrow.fy` asserts direct,
nested/grouped projections, all six scalar leaves and >i32 values, shared siblings,
parent reads, returned/tagged children, alias release, loop use, original copy
independence and old-referent identity after descriptor rebinding. Fresh and
codec-decoded MIR return the same asserted value. Four MIR mutations reject
exclusive kind relabeling, missing fields and wrong referents. Source rejections
cover parent/alias/sibling/owner writes, later arguments, exclusive children,
shared writes, local and multiple-origin escape, and excluded root forms.
Windows MSVC dev/release execute the success fixture and a runtime-dependent
checked overflow after release; source-session cold/warm all-hit is verified.

An initial always-overflowing function containing an assertion fails release
build with MSVC C4702/C2220 (CLI exit 6). A version using the already-supported
shared-call reborrow reproduces on the clean prior `251d15b0` compiler without
this feature. This pre-existing C reachability limitation remains unresolved;
the successful dynamic fault test does not claim it fixed. Initial failure and
baseline sources/tool identities/raw logs are kept with the current review in
`docs/ops/evidence/shared-mutable-field-borrow-20261004.json`.
Linux actual execution stays deferred to the owner's batch; there is no whole-
repository suite, performance experiment, warning suppression or cache bypass.
