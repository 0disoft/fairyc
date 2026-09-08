# ADR 0391: Shared field borrow projections

- Status: Accepted
- Date: 2026-09-09
- Contract: `docs/language/0002-borrows-and-return-origins.md`

## Decision

Implement the existing coarse-root field-borrow contract for shared references.
`ref owner.field` and nested field paths may start at a local/by-value parameter
place or at an existing shared aggregate reference. A borrowed field is an address
projection, not a copied field followed by a borrow of that temporary. Resource
field visibility remains declaration-module-only; no resource field is moved out.

Direct places retain a loan of the entire original root. Projection through an
existing shared reference retains its exact origin and loans. Borrow-return
summaries therefore preserve a single input origin through field access, while
returning a field of a callee-local/by-value owner is rejected. Moving, replacing
or exclusively borrowing a directly loaned root before the field reference's last
use remains rejected. Different fields do not acquire disjoint loans.

Owned call-result temporaries remain unaddressable. Mutable field borrows and
projection through an existing mutable aggregate reference are not admitted by
this slice: they require an explicit exclusive reborrow lifetime contract. The
current token-storage prerequisite uses a shared `ref Token`, so it does not require
those extensions.

## Lowering and evidence

Typed HIR carries `field-borrow` with its borrowed aggregate receiver. MIR retains
the receiver and field identity and verifies a shared owner/result, declared field
type, local operand and non-overwriting destination; malformed ownership-transfer
metadata is rejected. C emission takes the actual owner's field address, including
nested inert aggregate fields, without copying an intermediate aggregate. The
interpreter retains the existing aggregate state and changes only the reference
value's type. Source lifetime checks remain the permission authority; MIR shape
validation is not an independent lifetime proof.

`TestAnalyzeSharedFieldBorrow*` covers direct, nested, returned and last-use paths,
root move/replacement/exclusive conflicts, local escape, temporary rejection and
resource-field privacy. `TestVerifyFieldBorrow` mutates result/owner mutability,
field/type/operand identity and transfer metadata. Interpreter and C-address tests
check projections separately. `TestWindowsSharedFieldBorrow` executes nested
projections and a borrow-returning helper through ordinary CLI dispatch under low
and optimized MSVC. These two configurations share a compiler lineage.

Grammar and the existing hidden-origin/coarse-root rules do not change. Language
14, SDK stage0.58 and runtime ABI 0 remain unchanged; this is not a new public MIR
transport or a portable-layout promise. ADR 0392 subsequently supplies nested
resource C layout/cleanup and the owned-cooked-token storage probe without removing
its Buffer field. The complete lexer workload and self-hosting admission remain
unfinished.
