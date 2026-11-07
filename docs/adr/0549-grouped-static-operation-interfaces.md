# ADR 0549: Grouped static operation interfaces

- Status: Accepted for bounded prototype
- Date: 2026-11-07
- Extends: [ADR 0495](0495-explicit-static-operation-interfaces.md)

## Source and provider completeness

Existing argument-free `@interface` declarations retain their exact behavior.
`@interface(Arithmetic)` groups public operation declarations in one source
module. Its argument is a metadata label of 1..64 ASCII bytes matching
`[A-Za-z_][A-Za-z0-9_]*`; it introduces no value/type name. A group contains
2..16 operations. Each retains its own name, one type parameter, direct T
result, 0..4 immutable T parameters and empty safe body. Type parameter names
and operation arities may differ. Other declaration attributes remain closed.

Providers retain `@implements(operation, scalar)` and existing same-module,
exact-signature, duplicate-tuple and thirteen-scalar admission rules. A label
cannot replace the operation argument. For any scalar with at least one
provider in a group, every member must have exactly one provider. Structural
operation declarations are checked before provider/duplicate-tuple errors, then group
size, then completeness. A group with no providers is legal until requested;
a selected absent tuple retains E4502 at the original root request.

Malformed grouped declarations, invalid group size and partial implementations
fail with E4501 before specialization/publication, even for unused operations
or scalar tuples. Group size errors use the first source-order member's full
declaration span. Partial-provider errors use the earliest source-order provider
belonging to an incomplete group/scalar tuple. Group-related structural errors
use the offending declaration span. Existing ungrouped diagnostics are unchanged.

Calling an operation still uses its explicit generic name, e.g. `add<i32>(a,b)`;
ordinary selected-provider calls, argument order, effects, ownership and native
ABI are unchanged. There is no interface value, vtable, allocation, inheritance,
associated type, default/blanket/orphan implementation or dynamic dispatch.

## Identity and dependency closure

Ungrouped operation/provider identity bytes and domains stay unchanged. A
grouped member uses `source-static-operation-group-v1` over a JSON array of
original written AST functions: members sorted by operation name; for each
member its operation AST followed by every provider sorted by scalar name.
The array includes all members and all supplied scalar tuples, including unused
ones. Existing AST JSON, spans, annotation arguments and literal spellings are
retained; this does not add source-reordering/renaming invariance.

All members share this group definition digest. Existing distinct operation
capability IDs, per-provider `source-static-provider-v1` digests and each
member's `source-static-bound-operation-v1` binding remain. Thus changing any
sibling declaration/provider changes every member's definition and dependent
template/specialization identity, even when only one operation is requested.
Equal-width unused sibling edits must demonstrate this closure independently
of incidental source-span changes. Unrelated private edits preserve existing
dependent stability rules. No cache-key or artifact wire version is changed.

## Delivery

Go admission/planning, cache proof and Fairy metadata/native proof have disjoint
owners in an isolated checkout while existing main CLI validation continues.
The coordinator owns Fairy group admission, error-location integration, source
lock refresh, review and commits. Windows checks precede integration into main;
native Linux remains batched. Full CLI/cache and final self-build/fixed-point,
distribution and hosted CI are separate gates. No public API/ABI, DB, dependency,
runner, target promotion or remote action is introduced.
