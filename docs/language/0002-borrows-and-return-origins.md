# Borrows and Return Origins

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Borrow kinds

`&T` is a non-null shared read-only borrow. `&mut T` is a non-null exclusive mutable borrow. Neither kind owns the referent. Nullability is expressed with `Option<&T>` or a raw pointer.

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
a by-value parameter with `&parameter` borrows its callee-local stack storage, so its origin is
`Local(parameter)` and returning it is `E2102`.

`Static` may be shortened to an expected parameter or region origin. Origins are never lengthened.

UTF-8 string, byte-string, and C-string literals produce immutable compiler-owned storage with `Static` origin. The source language has no explicit lifetime spelling for that origin; it is tracked as the same hidden fact used for other program-static storage.

## Local loans

Stage 0 loans are local and end after the last proven use in the current function. A shared loan prevents mutation or move of the borrowed root. A mutable loan prevents every competing access to that root.

Place overlap is deliberately coarse. Borrows of different fields of the same root are treated as borrowing the whole root. Field-disjoint borrowing and two-phase borrows are deferred until real APIs justify their diagnostics and analysis cost.

Passing an existing mutable borrow to a call creates a call-bounded reborrow when the parameter is `&mut T`. The original borrow becomes usable again after the call's last use. Returning, storing, or assigning the borrow to a longer-lived destination is not a call-bounded reborrow and must satisfy the ordinary origin and escape rules.

## Return rule

A function may return a safe borrow only when every successful return derives from one origin:

- one declared borrow parameter;
- one declared region parameter; or
- static storage.

Selecting between borrows from two parameters and returning the selected value is rejected even when each branch is individually safe. The caller must split the function, return an index or tag, copy a value, or use another explicit ownership design.

In Stage 0, a complete return type contains at most one safe-borrow leaf. This applies through `Option` and `Result` payloads.

## Storage and function values

Named `struct` and `resource` aggregates cannot contain safe borrow fields. A safe borrow may exist in a local temporary and in the restricted return shapes above, but it cannot become durable aggregate state.

`str` and `CStr` are compiler-known unsized view types, not ordinary structs containing raw pointers. Their safe forms are `&str` and `&CStr` and retain an ordinary borrow origin.

A function that returns a safe borrow cannot be converted to a function pointer, stored in an interface or dynamic-dispatch table, or used as a callback. Those forms would need the return origin to become part of a first-class function type, which Stage 0 intentionally does not model.

## Self-reference and interior mutability

Safe self-referential aggregates are rejected because moving the aggregate could invalidate its internal reference. Use indices, generation handles, a lexical region with external references, or a narrow unsafe representation.

User-defined interior mutability is not available in Stage 0. `&T` therefore preserves its read-only meaning. `Region` is a compiler-known capability with dedicated rules, not a user-definable exception.

## Raw and foreign boundaries

Raw pointers do not acquire a safe origin automatically. Foreign signatures expose raw pointers and scalar ABI values. A safe wrapper may construct a borrow only after it establishes non-nullness, extent, alignment, aliasing, and lifetime for the duration promised by its API.

The C backend must not emit `restrict`, LLVM `noalias`, or related alias promises merely because a pointer came from safe-looking source syntax. Such attributes require proof from the final borrow analysis and are disabled for the Stage 0 reference backend.

## Required diagnostics

Diagnostics must identify the conflicting access, the active loan and its last use, an escaping local origin, a multiple-origin return, a stored safe borrow, and a borrow-returning function-value conversion. When possible, the explanation should recommend copying, splitting the API, returning an index, or using a region.
