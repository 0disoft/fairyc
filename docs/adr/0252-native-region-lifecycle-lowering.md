# ADR 0252: Native Region Lifecycle Lowering

- Status: Accepted
- Date: 2026-07-31

## Context

ADR 0251 made `Region.place(T)` explicitly fallible and preserved the success borrow's Region
origin through verified MIR. The constrained C backend still rejected `region-enter`,
`region-place`, and `region-reset`, so the checked-in safe Region workspace could run only in the
abstract interpreter. Recovering the Arena from source or HIR during emission would violate the MIR
boundary; treating allocation failure as success or a trap would violate the language `/8` Result
contract.

## Decision

1. The constrained C backend admits Region operations only after MIR verification and only when the
   selected target admits `fy_std_region_begin_v0`, `fy_std_region_allocate_zeroed_v0`, and
   `fy_std_region_reset_v0`.
2. `region-enter` uses the exact Arena owner local and mutable-borrow type retained by MIR. Each MIR
   Region symbol receives one zero-initialized private C Region record scoped to its function.
3. `region-place` admits only a fully initialized inert owner type and an exact
   `Result<ref T, std::alloc::AllocError>` destination. It requests zeroed storage using the emitted
   C type's `sizeof` and `_Alignof` and evaluates the already-lowered value operand once.
4. Status zero initializes the allocated object and constructs `.ok` with a const pointer carrying
   the verified Region origin. A nonzero status constructs `.err` with the exact allocation status
   and publishes no pointer.
5. `region-reset` calls the target reset helper on every verified lexical exit. Begin or reset
   rejection after verification is an internal invariant failure and uses the existing unreachable
   trap category; placement failure remains ordinary Result data.
6. Safe scalar and admitted struct/resource borrows use private C pointer transport. The current
   Region placement subset is canonical scalars plus supported inert structs; unsupported inert
   arrays fail as a backend capability boundary. This does not create a public borrow ABI, permit
   borrow retention, or allow a safe borrow to cross the emitted function-return boundary.
7. The checked-in `examples/regions` workspace is compiled and executed under development and
   release x64 MSVC profiles. Both must return 42 after Arena creation, Region entry, placement,
   dereference, reset, and automatic Arena drop.

## Consequences

- Target component selection follows the three used Region symbols and closes over Arena and
  SystemHeap through the existing standard-library contract.
- Generated C contains both `.ok` and `.err` construction. Product linkage deliberately exposes no
  allocator failure-injection control, so the safe workspace proves the native success path while
  the separate raw C Arena probe continues to prove both allocation-failure rollback stages.
- Language stays `fairyc-language/8`, SDK stays `0.0.0-stage0.10`, and runtime ABI stays
  `fairyc-runtime/0`; this decision adds backend capability and evidence rather than a new public
  version boundary.
- Other native backends must either implement the same verified lifecycle and fallible placement
  contract or reject Region operations before toolchain invocation.
