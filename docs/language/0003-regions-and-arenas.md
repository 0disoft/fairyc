# Regions and Arenas

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`
- Region fixtures: `spec/fixtures/region/manifest-v1.json`

## Region capability

`Region` is a compiler-known lexical capability. It is not equivalent to `ref mut Arena`, because
repeated allocation through an ordinary exclusive borrow would otherwise create overlapping-borrow
problems or require user-defined interior mutability.

A region capability:

- is created only by a lexical region construct;
- can be used repeatedly for allocation within that scope;
- cannot be copied, stored in an aggregate, returned, or captured;
- resets when fallthrough, `return`, Result `?` propagation, `break`, or `continue` exits the lexical region;
- assigns `RegionParam(i)` or the corresponding local region origin to derived borrows.

`region scratch in arena { ... }` requires `arena` to have type `ref mut std::alloc::Arena`. The
exclusive borrow covers the lexical block, while `scratch` is a distinct compiler-owned capability
so repeated allocations do not create overlapping ordinary borrows.

The Stage 0 allocation surface is limited to zero-initialized bytes, copied bytes, and `place(value)` for a fully initialized inert value whose type is inferred from the expression. There is no user generic `alloc<T>` syntax, safe uninitialized byte view, or resource placement.

## Allocation and escape

Allocation returns a safe borrow whose origin is the region. The referent cannot be moved out as an owned value unless the type and operation explicitly copy it. A region-derived borrow cannot be stored anywhere that outlives the region or returned unless the region is itself a declared input origin of the current function.

The compiler reports both the allocation span and the attempted escape span.

When the attempted escape is a return, `E3101` owns the rejection because the decisive fact is `LocalRegion(name)`. A generic `E2102` for the same returned expression is a cascade and is suppressed rather than reported as a second independent error.

Region fixtures expose the compiler-known allocation operation, hidden origin, optional last use, and return escape decision. They do not expose addresses, arena offsets, Flow block numbers, or MIR locals. Region v1 resolves the lexical escape source through `pass.package.stdlib-arena-region`, whose explicit standard-library instance owns an opaque `std::alloc::Arena` resource. The fixture does not choose a constructor, capacity model, or public representation.

## Plain-region element rule

A plain region may contain only types with `NeedsDrop = false`. Bulk reset releases storage without visiting individual elements, so admitting resources would silently skip cleanup.

Inert aggregates placed in a region must also contain no safe borrow field. Internal graphs use indices, offsets, or raw pointers under an unsafe abstraction rather than named safe-reference fields.

## Resource-aware arenas

A future resource-aware arena is a different abstraction. It would need:

- a registered drop list;
- a rule for partial construction failure;
- deterministic reverse-order destruction;
- ownership transfer and removal semantics;
- reset behavior when a close operation can fail;
- panic, trap, and foreign-control-flow behavior;
- tests for reentrancy and double registration.

It is deferred rather than being treated as a hidden mode of `Region`.

## Reset and traps

Normal lexical exit resets the region after ordinary inner-scope resource cleanup has completed. A trap is non-unwinding and performs no region reset as an observable language action; the process terminates. Foreign `longjmp` or exception transfer across the region scope is forbidden.

The current executable prototype proves fallthrough, direct `return`, Result `?` propagation,
nearest-loop `break` and `continue`, and nested Region exits with explicit MIR enter, place, drop,
and reset operations. Its trace records the exit kind, placement count, and how many resource drops
completed before reset without exposing addresses or offsets. The verified `region-enter` retains
that exact Arena owner local, owner resource type, and mutable-borrow type; it does not retain or
reconstruct a raw address. Executable lowering currently requires the Arena source to be a direct `ref mut` local; a
side-effecting borrow-producing expression is rejected rather than skipped.

## Concurrency

Stage 0 has no cross-thread region capability and no `Send` or `Sync`-style user traits. A region and its borrows are thread-local. Later concurrency support needs an explicit memory model and cannot be inferred from these rules.

## Required diagnostics

Diagnostics must distinguish a region borrow escape, stored region capability, returned local region, resource allocation in a plain region, and unsupported cross-thread transfer.
