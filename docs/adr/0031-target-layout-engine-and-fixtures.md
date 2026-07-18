# ADR 0031: Target layout engine and versioned fixtures

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Target source: `internal/target/`
- ABI source: `docs/abi/c-abi-v0.md`
- Fixture source: `spec/fixtures/layout/manifest-v0.json`
- Implementation source: `internal/layout/` and `internal/layoutfixture/`

## Context

ADR 0030 made the first target facts executable, but it did not compute aggregate layouts. Resource
and `@repr(C)` lowering need deterministic size, natural alignment, declaration-order offsets, and
padding without asking Go or whichever C compiler happens to be installed. A table of scalar sizes
alone cannot catch offset arithmetic overflow, duplicate fields, empty aggregates, zero-length
arrays, nested alignment, or trailing padding mistakes.

The source frontend does not yet implement general inert structs, fixed arrays, explicit C enums,
or foreign declarations. Binding layout directly to HIR now would mix target mechanics with
unfinished source semantics. The first engine therefore consumes a closed implementation type
description and is tested through an independent versioned manifest.

## Decision

1. `internal/layout` owns host-independent layout execution bound to one validated `target.Spec`.
   It never reads Go reflection or `unsafe`, runs a C compiler, inspects libc, or accepts a local
   path.
2. The closed v0 input families are language scalar, C scalar, data pointer, function pointer,
   explicitly based C enum, positive-length fixed array, and nonempty declaration-ordered struct.
   Listing one family here does not implement its Fairy C source syntax or ABI call lowering.
3. Arrays retain the element's natural alignment and multiply size with checked `uint64`
   arithmetic. Length zero is rejected; size overflow is a named failure.
4. Struct fields are placed in declaration order. Each field offset is rounded up to its natural
   alignment, the aggregate alignment is the maximum field alignment, and final size is rounded up
   to that aggregate alignment. The result records exact top-level internal and trailing padding
   ranges. Nested field-internal padding remains owned by the nested layout.
5. Empty structs, duplicate or empty field names, unknown scalars, irrelevant type metadata,
   invalid alignment, checked size/alignment overflow, and nesting beyond the bounded internal
   computation depth fail closed. Fixture-visible failures use the closed codes `invalid-type`,
   `unknown-scalar`, `duplicate-field`, `empty-aggregate`, `invalid-array-length`, `overflow`, and
   `depth-limit`.
6. `fairyc.layout-fixtures/v0` owns repository observations, not a public serialized compiler IR.
   Its runner uses strict Go JSON decoding, rejects unknown fields and trailing documents, verifies
   the exact local schema reference, target identity, nonempty unique ledgers, strictly sorted
   unique case IDs, expectation shape, and exact layout or error code.
7. The initial manifest contains 14 cases covering scalar, pointer, explicit-enum, array, padded
   and nested struct success plus empty, duplicate, zero-length, unknown-scalar, and overflow
   rejection.
8. Native Clang/GCC layout probes, SysV AMD64 argument/return classification, bidirectional C ABI
   calls, source semantic lowering, resource representation, and backend consumption remain
   separate gates. `TargetSpec` therefore replaces its broad missing `layout-fixtures` marker with
   the narrower `c-layout-probes` marker.

## Consequences

- Future resource and C-layout lowering can ask one target-bound engine instead of repeating offset
  formulas in each backend.
- Exact padding is executable repository evidence, while agreement with real Clang and GCC remains
  visibly unproven.
- Malformed compiler-created layout inputs fail with a stage-local code rather than wrapping around
  an integer or leaking host behavior into output.
- The recursive fixture schema and Go data model must change together when a new layout family or
  failure code is introduced.

## Rejected alternatives

- Use C `sizeof` and `offsetof` as the layout engine: makes compilation depend on a local toolchain
  and cannot serve target-independent analysis or cross-host inspection.
- Use Go struct layout: measures the bootstrap compiler process and is wrong by construction for a
  different target.
- Emit C structs first and infer offsets later: generated code would already have committed to a
  representation before verification.
- Add resource layout directly: resource drop state, active tagged payloads, and user drop bodies
  need additional contracts beyond inert field placement.
- Treat integer overflow as an impossible compiler bug: hostile or malformed dimensions can reach
  layout before a future frontend has every bound, so the stage must fail closed itself.
