# Borrows and Return Origins

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Borrow kinds

`ref T` is a non-null shared read-only borrow. `ref mut T` is a non-null exclusive mutable borrow.
Neither kind owns the referent. Nullability is expressed with `Option<ref T>` or a raw pointer.

## Hidden origins

The compiler tracks a `BorrowOrigin` that is not part of source syntax:

| Origin | Meaning | May be returned safely |
| --- | --- | --- |
| `Static` | Static storage with program lifetime | Yes |
| `Param(i)` | Borrow derived from parameter `i` | Yes, as the function's single input origin |
| `RegionParam(i)` | Allocation or borrow derived from region parameter `i` | Yes, as the function's single region origin |
| `Local(scope)` | Borrow derived from a local owner | No |
| `RawUnknown` | Provenance entered through raw or foreign memory | No safe return without an explicit unsafe wrapper proof |

`Param(i)` is assigned only when an already-borrowed parameter is propagated. Directly borrowing
a by-value parameter with `ref parameter` borrows its callee-local stack storage, so its origin is
`Local(parameter)` and returning it is `E2102`.

`Static` may be shortened to an expected parameter or region origin. Origins are never lengthened.

UTF-8 string, byte-string, and C-string literals produce immutable compiler-owned storage with `Static` origin. The source language has no explicit lifetime spelling for that origin; it is tracked as the same hidden fact used for other program-static storage.

## Local loans

Stage 0 loans are local and end after the last proven use in the current function. A shared loan prevents mutation or move of the borrowed root. A mutable loan prevents every competing access to that root.

Place overlap is deliberately coarse. Borrows of different fields of the same root are treated as borrowing the whole root. Field-disjoint borrowing and two-phase borrows are deferred until real APIs justify their diagnostics and analysis cost.

ADR 0391 implements shared field-address projection from local/by-value places and
existing shared aggregate references, including nested fields. A local root keeps
its local origin; projection through a shared reference retains that reference's
origin and loans. Resource field privacy is unchanged, and no intermediate field
value is copied or moved. Owned call-result temporaries, mutable field borrows and
projection through mutable aggregate references remain outside this executable
slice.

ADR 0394 admits plain assignment to a direct `i32`, `u8`, `u32`, `usize`, or `bool` field of
a mutable local struct/resource or an existing `ref mut` aggregate receiver.
The right-hand side evaluates once and must have the exact field type. Resource
privacy, moved-owner checks and whole-root loan conflicts remain enforced.
Copying an inert aggregate retains value semantics, including when that value is
stored inside another aggregate; assigning through a borrow changes its referent.
ADR 0412 also admits checked compound assignment to these direct integer fields.
The receiver and its old field value are selected before the RHS runs once;
arithmetic failure traps before the field store. Exact types and whole-root loans
remain enforced. Boolean arithmetic and nested field paths are not admitted.
Nested field paths and general aggregate-field
replacement remain source-located unsupported lowering results without publishing
executable MIR. Direct plain replacement of a non-`repr(C)` one-`usize`-field
inert struct from a typed local is the bounded exception. Shared field reads and borrows retain the
separate projection boundary above; field stores do not admit mutable field borrows.

ADR 0401 admits plain `*slot = value` for a named `ref mut` to `bool`, `u8`,
`i32`, `u32`, or `usize`. The exclusive loan remains live while the right-hand
side is evaluated once; competing borrows, direct owner reads, and moves are
rejected. A read through the same reference is allowed. `Vec.at_mut<i32>` may
therefore update its selected element without copying or resizing the vector.
Aggregate replacement, compound dereference assignment, temporary reference
targets, and mutable field-address projection remain unsupported.

Passing an existing mutable borrow to a call creates a call-bounded reborrow when the parameter is
`ref mut T`. The original borrow becomes usable again after the call's last use. Returning, storing,
or assigning the borrow to a longer-lived destination is not a call-bounded reborrow and must
satisfy the ordinary origin and escape rules.

ADR 0414 additionally permits an existing `ref mut T` as an internal `ref T`
call argument or shared method receiver for the exact same referent. Earlier
arguments remain held through later argument evaluation; overlapping mutation
is E2101 in either order. A returned shared reference, its aliases and tagged
payload keep the original origin and freeze the parent and whole owner until
their last use. Afterwards the parent may be edited again. This call-only
conversion does not relax direct binding, assignment, return or escape rules.

ADR 0415 admits `view[start..end]` for an existing `ref mut u8[]`, producing
another `ref mut u8[]` with the same owner and origin. Bounds may be omitted;
the receiver and explicit bounds evaluate once in source order and trap unless
`start <= end <= len`. The child retains a distinct exclusive loan generation.
Its parent, parent aliases and original owner cannot be accessed before the
child's last use; the child and its descendants retain access. Returned and
tagged children preserve that restriction. Disjoint sibling intervals are not
proven, so retained siblings remain rejected even for numerically disjoint
ranges. Shared byte slicing keeps its existing result and lifetime behavior.

The first executable array-view bridge permits a direct shared borrow of an admitted `u8[N]`
place where `ref u8[]` is already expected. It retains the place's origin and loans the whole array.

ADR 0430 adds the corresponding shared `i64[N]` to `ref i64[]` bridge. Length,
indexing and half-open slicing count i64 elements and preserve the whole-owner
loan, including shared aliases/siblings, nested slices and parameter-derived
returns. Empty end slices are permitted. The receiver stays borrowed while
explicit indices/bounds evaluate. Owner mutation waits until the last retained
use; local/by-value-parameter escape remains E2102. Owned unsized slices remain
unsupported. ADR 0433 adds exclusive fixed-array i64 views below. No i64-to-byte
reinterpretation is implied.

ADR 0431 adds the SDK `std::vec::view_i64(ref Vec<i64>)` initialized-prefix
view with the same whole-owner loan. All mutating vector operations, replacement
and move are rejected until the last retained view/subview use, even when a
reserve or truncate would leave the allocation unchanged. Obtain a new view
after growth. Parameter-derived and tagged returns retain the vector origin;
local-owner escape remains rejected. There is no generic or mutable vector
slice conversion.

ADR 0432 admits `for (let i64 value in view)` for a named immutable `ref i64[]`
local or parameter. It snapshots length, visits checked indices in order and
copies an immutable item into body scope. The shared owner is held until loop
exit, including bodies ending with break/return/propagation. Mutation or move
inside the body is rejected; mutation after exit follows ordinary later-use
rules. Named array/Vec subviews are eligible. Direct collection values, mutable
items, mutable view bindings and expression-valued iterables are not admitted.

ADR 0433 admits `ref mut i64[]` from a mutable `i64[N]` place with explicit
`ref mut data`, including checked element stores/compound assignment and
exclusive half-open subviews. Lengths and offsets count elements. It reuses
ADR 0415's whole-parent freeze, alias generations, last-use release and
parameter-derived return summaries. Retained siblings remain rejected even
when disjoint; shared readers require a shortened shared reborrow. Vec mutable
view conversion and mutable for-in iteration are covered separately by ADR 0434-0435.

ADR 0434 adds `std::vec::view_mut_i64(ref mut Vec<i64>)` over the initialized
prefix. The root loan freezes other vector access until the last retained
view/subview/alias use, including structural changes that would not reallocate.
Derived and tagged returns preserve owner origin. Growth requires releasing the
old view and obtaining a fresh one. Generic mutable Vec slices remain unsupported;
ADR 0435 admits mutable u8/i64 view iteration below.

ADR 0435 extends the named-view for-in form to exact `ref mut u8[]` and
`ref mut i64[]` immutable bindings and parameters. Items are immutable scalar
copies; stores use explicit indices through the view. Length is snapshotted,
but each indexed load observes earlier stores. Loop-held exclusive loans block
owner mutation, reallocation, replacement and consuming moves through early
exits. Ordinary child/shared reborrow rules still apply inside the body; the
loop hold itself is not a shared reborrow. Mutable descriptors, reference items
and expression-valued iterables remain unadmitted. Both byte/i64 shared and
exclusive loops hold owners until exit, including before return/propagation.

ADR 0436 adds `std::vec::view_u8(ref Vec<u8>)` and
`std::vec::view_mut_u8(ref mut Vec<u8>)` over the initialized byte prefix.
Shared/exclusive owner holds, short reader reborrows, child/sibling generations,
returned/tagged origins and post-release growth follow the same i64 view rules.
The existing u8 for-in form applies to these named views. No generic vector
slice conversion or reinterpretation of other Vec element types is admitted.

ADRs 0437-0438 extend matching fixed-array and SDK initialized-prefix Vec views
to i32/u32/usize. Shared/exclusive forms retain the same owner, return-origin,
subrange and sibling rules. All five integer element types support named
immutable-descriptor for-in with copied items; the owner stays held through every
loop exit and indexed writes affect later loads. Owned unsized slices, bool
views, exclusive or owned aggregate slices, element reinterpretation and implicit
Vec conversion remain unsupported. A shared `ref T[]` exists for an actual
nonempty nominal struct whose fields are all plain scalars; a `bool` field is
admitted, but a bare `ref bool[]` view is not. Indices count elements. SDK
helpers are `view_i32`/`view_mut_i32`, `view_u32`/`view_mut_u32`,
`view_usize`/`view_mut_usize`.

ADR 0439 adds the shared `std::vec::view<T>(ref Vec<T>) -> ref T[]` over the
initialized prefix of `Vec<T>`. `T` may be one of the five existing integer
elements (u8/i32/i64/u32/usize), which aliases the current per-type integer view,
or an actual nonempty scalar-field nominal struct. For a nominal element the slice
cannot reinterpret a `Token` view as a same-layout `Other` view; a nested
aggregate/array/resource/borrow/pointer/enum field, a `ref mut` struct slice or an
owned slice is rejected. An index copies one struct value and `view[i].field`
reads are allowed, while element and field stores are rejected; a copied mutable
local survives an owner clear, and a `var ref Token[]` binding may still be
reassigned to another view. Owner holds, returned/tagged origins, subrange and
sibling rules reuse ADR 0415.

ADR 0440 admits copied-value `for (let Token item in view)` for a named immutable
shared struct-view descriptor or parameter. The exact scalar-field nominal item
must match the view; imported qualified names resolve normally. Length is
snapshotted once and each iteration performs a checked struct-value read. The
whole owner loan stays held through nested loops and break/continue/return/error
propagation, and is released at the written loop exit. A body may make a mutable
local copy of the item, but cannot modify the shared owner. Reference items,
mutable descriptor iterables, call/slice-expression iterables and implicit Vec
iteration remain outside this slice form; integer iteration rules are unchanged.

A mutable local containing a safe borrow may be reassigned. Flow evaluates the
right-hand side first and then replaces the target's origin and retained-loan set. Each direct
borrow creates a loan generation; copying a borrow preserves that generation, so replacing one
alias does not release an owner while another alias can still reach a later use.

The unsafe built-ins `str::from_raw_parts(owner, data, length)` and
`CStr::from_raw_parts(owner, data, storage_length)` require `owner` to be a shared safe borrow. The
returned view copies the owner's hidden origin and retained loans; the raw pointer contributes no
origin. A safe wrapper therefore returns `Param(i)` only when its owner argument already has that
origin. Moving the owner through a by-value resource call before the view's last use is `E2105`.

`if` and `switch` analyze each branch with an independent borrow environment and join only reachable
exits. Equal origins compact to one fact. Different possible origins remain a disjunction and a
return of that value is `E2103`; source order never chooses a winning branch. Loan liveness is also
path-aware, so a use in one branch does not keep an unrelated mutually exclusive branch's owner
borrowed.

For `while`, Flow joins the initial entry, reachable fallthrough, and nearest-loop `continue` edges
to a loop-header fixed point. A retained direct borrow uses one canonical static loan-site ID across
analysis rounds instead of creating an unbounded sequence. The converged header and nearest-loop
`break` environments form the post-loop state, including the zero-iteration path. Returning a
post-loop value with different possible entry and iteration origins remains `E2103`. Scalar
predicate correlation and at-least-once loop proofs are not part of this contract.

Every internal borrow-returning function has a compiler-only summary inferred from its reachable
returns: one exact `Param(i)`, `RegionParam(i)`, or static source. Summaries reach a fixed point
through a reverse-call dependency worklist before ordinary Flow checks, so forward declarations,
imported callees, argument reordering, and acyclic wrapper chains do not depend on source order.
At a call, only the selected actual
argument contributes its origin and retained loans; unselected borrow arguments remain
call-bounded. A static summary borrows no argument. A recursive dependency whose result origin
cannot be completely proven remains unsupported instead of inventing provenance.

The same proof follows the single safe-borrow leaf through `Option` and `Result`. Constructing
`.some`, `.ok`, or `.err` retains the selected payload's origin only when that payload contains the
leaf; a payload-free `.none` and the non-borrowing variant retain no loan. A local tagged value keeps
its owner borrowed until that value's last use. `switch` transfers the proof only to the matching
payload binding and removes the binding at the arm boundary. For `?`, the success proof continues
as the expression value while a borrowing failure proof is checked as the implicit early return.
No source-level lifetime or tag-specific annotation is required.

## Return rule

A function may return a safe borrow only when every successful return derives from one origin:

- one declared borrow parameter;
- one declared region parameter; or
- static storage.

Selecting between borrows from two parameters and returning the selected value is rejected even when each branch is individually safe. The caller must split the function, return an index or tag, copy a value, or use another explicit ownership design.

In Stage 0, a complete return type contains at most one safe-borrow leaf. This applies through `Option` and `Result` payloads.

## Storage and function values

Named `struct` and `resource` aggregates cannot contain safe borrow fields. A safe borrow may exist in a local temporary and in the restricted return shapes above, but it cannot become durable aggregate state.

`str` and `CStr` are compiler-known unsized view types, not ordinary structs containing raw
pointers. Their safe forms are `ref str` and `ref CStr` and retain an ordinary borrow origin.

A function that returns a safe borrow cannot be converted to a function pointer, stored in an interface or dynamic-dispatch table, or used as a callback. Those forms would need the return origin to become part of a first-class function type, which Stage 0 intentionally does not model.

## Self-reference and interior mutability

Safe self-referential aggregates are rejected because moving the aggregate could invalidate its internal reference. Use indices, generation handles, a lexical region with external references, or a narrow unsafe representation.

User-defined interior mutability is not available in Stage 0. `ref T` therefore preserves its
read-only meaning. `Region` is a compiler-known capability with dedicated rules, not a user-definable
exception.

## Raw and foreign boundaries

Raw pointers do not acquire a safe origin automatically. Foreign signatures expose raw pointers and scalar ABI values. Four exact owner-bound built-ins are the only current raw-parts reconstruction path: the two text forms return shared `str` or `CStr` views, while the two byte-slice forms return shared or exclusive `u8[]` views. They take provenance from their separate safe owner argument. The exclusive form additionally requires exclusive owner provenance and mutable data. Every unsafe wrapper must still establish the permitted null-and-zero case, extent, alignment, aliasing, initialization, text invariants where applicable, and lifetime for the duration promised by its safe API.

The C backend must not emit `restrict`, LLVM `noalias`, or related alias promises merely because a pointer came from safe-looking source syntax. Such attributes require proof from the final borrow analysis and are disabled for the Stage 0 reference backend.

## Required diagnostics

Diagnostics must identify the conflicting access, the active loan and its last use, an escaping local origin, a multiple-origin return, a stored safe borrow, and a borrow-returning function-value conversion. When possible, the explanation should recommend copying, splitting the API, returning an index, or using a region.
