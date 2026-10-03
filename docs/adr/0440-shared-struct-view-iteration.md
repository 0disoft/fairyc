# ADR 0440: Shared struct view for-in iteration

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0431, 0436, 0438, 0439

## Decision

ADR 0439 added the shared nominal struct view `std::vec::view<T>(ref Vec<T>) -> ref T[]`
but admitted no iteration. This ADR adds `for` over a named immutable shared nominal
struct slice, as in `for (let Token item in view)`.

The iterable must be a named immutable shared reference to a nominal struct slice,
`ref Token[]`, reached through a `let` binding or a function parameter. The declared
item type must match the view element exactly: an actual nonempty struct whose fields
are all plain language scalars (`bool`, `u8`, `i32`, `i64`, `u32`, `usize`), per
ADR 0439. Module-qualified names such as `tokens::Token` resolve to the same nominal
identity. Iteration copies one whole struct value per element, so `item` is an
immutable `let` whole-value copy; the body may make a mutable local copy of it.

## Implementation and semantics

The view descriptor length is snapshotted once at loop entry and every element is read
through a checked indexed load, so short readers and subviews keep the ADR 0415
reborrow rules. The loop holds the whole-owner shared loan for its full duration,
through `break`, `continue`, `return` and error propagation, and after `item` has been
copied; the loan is released after the loop exits. Nested, scalar and mixed loops lower
the same way, and the integer element rules are unchanged.

The following forms remain rejected: no `ref` or `ref mut` iteration item
over a slice, no mutable descriptor iterable (`var ref Token[]`), no call or
slice-expression iterable, and no direct `Vec` iteration promotion. The SDK keeps its
separate preexisting reference-item `Vec` path. Struct elements with nested aggregate or nonscalar fields
remain outside the scalar-field view contract.

## Identities and compatibility

SDK stage0.84, language `/20` and runtime ABI `/0` are unchanged, as are the grammar,
MIR and interface schemas. Lowering reuses the existing shared nominal descriptor, so
no new runtime C symbol appears. The compiler executable identity separates cache
generations, so a rebuilt compiler does not reuse stale loop lowerings.

## Verification and limits

The `examples/struct_vec_view` analyzer and first-semicolon discovery now use struct
for-in. Windows development/release each pass seven iteration fixtures: empty,
singleton copy, parameter/nested/mixed loops with continue/break, early return and
owner release, returned/tagged subviews, error propagation, and checked sum
overflow. The token analyzer passes twelve cases per profile, including the
module-qualified item type. Twenty source rejection cases cover owner mutation
and move even before early return or propagation, wrong nominal/scalar items,
mutable descriptors, expression iterables, direct Vec and reference items.
Owning AST/semantic/generic tests, private MIR codec roundtrip, warm all-hit cache,
public CLI check/locked build/run, and Go vet pass. Raw outcomes and compiler,
SDK, source and EXE hashes are in
`docs/ops/evidence/struct-for-20261004.json`.

Linux execution is deferred to the owner's batch, so emitted Linux C is not runtime
proof, and this is not a released-compiler, target-maturity or performance claim. No
broad tests, benchmarks, dependency installation, CI runner, PowerShell runner or push
is added.
