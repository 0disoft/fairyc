# ADR 0255: Generated-Fairy Region Failure Execution

- Status: Accepted
- Date: 2026-08-04

## Context

ADR 0252 proved native Region placement success from checked-in Fairy C source and proved both
allocation rollback stages through a separate raw C probe. It did not execute the `.err` arm
produced from Fairy C because product linkage intentionally exposes no allocator-failure controls.
Generated C containing an error construction is weaker evidence than executing that construction,
its source-level switch arm, lexical Region reset, and automatic Arena cleanup together.

## Decision

1. The Windows host smoke compiles a bounded Fairy C workspace through package loading, semantic
   Flow, verified MIR, and the x64 MSVC constrained-C backend. Its source performs one
   `Region.place(i32)`, requires `.err`, and accepts only allocation status `1`.
2. The test harness renames only the generated C entry point and adds a C-only outer entry point.
   The generated function body, Result construction, Region reset, and Arena cleanup remain
   unchanged compiler output.
3. The outer entry links the existing probe-only Arena/SystemHeap closure. It independently fails
   the payload allocation and the tracking-node allocation, invokes the generated Fairy program
   after each failure is armed, and requires result `42` plus zero live probe allocations.
4. Both low and optimized MSVC profiles compile as separate translation units, pass COFF, PE32+,
   and AMD64 unwind inspection, and execute both failure schedules. This yields four forced
   generated-Fairy `.err` executions across two native images.
5. Failure injection remains test-only. No probe symbol enters the public SDK, runtime symbol
   contract, ordinary product linkage, checked-in user example, or generated Fairy source.

## Consequences

- The native evidence chain now covers source-level error selection, exact status transport,
  payload-failure non-publication, node-failure rollback, lexical reset, automatic Arena drop, and
  zero retained allocations.
- The raw C Arena probe remains responsible for byte-level zeroing, copied placement, stale Region
  rejection, active-drop preservation, and detailed unchanged-output checks. The new proof does not
  replace that matrix.
- This is test and evidence hardening only. Language remains `fairyc-language/8`, SDK remains
  `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim.
