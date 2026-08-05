# ADR 0256: Generated-Fairy Buffer Failure Execution

- Status: Accepted
- Date: 2026-08-04

## Context

ADR 0246 published a move-only `std::bytes::Buffer` whose construction and mutation operations are
explicit fallible `Result` operations, and ADR 0134 proved failure-atomic behavior through a raw C
probe with a probe-only deterministic allocator. Product linkage intentionally exposes no
allocator-failure controls, so the `.err` arms produced from Fairy C were not executed from
generated source. Generated C containing an error construction is weaker evidence than executing
that construction, its source-level switch arm, and automatic Buffer cleanup together.

## Decision

1. The Windows host smoke compiles two bounded Fairy C workspaces through package loading, semantic
   Flow, verified MIR, and the x64 MSVC constrained-C backend. The first source requires
   `bytes::with_capacity(16)` to return `.err(BufferError)` and accepts only allocation status `1`.
   The second source constructs a canonical empty `Buffer` with `bytes::with_capacity(0)`, requires
   `bytes::resize_zeroed(ref mut buffer, 16)` to return `.err(BufferError)` with allocation status
   `1`, and requires both `len` and `capacity` to remain zero.
2. Each test harness renames only the generated C entry point and adds a C-only outer entry point.
   The generated function body, Result construction, and automatic Buffer cleanup remain unchanged
   compiler output.
3. Each outer entry links the existing probe-only SystemHeap allocator closure. It arms the initial
   allocation failure before the generated program and requires result `42` plus zero live probe
   allocations. A zero-capacity `with_capacity` performs no allocation, so the same armed failure
   reaches the subsequent `resize_zeroed` growth allocation.
4. Both low and optimized MSVC profiles compile each path as separate translation units, pass COFF,
   PE32+, and AMD64 unwind inspection, and execute the forced failure. This yields four forced
   generated-Fairy Buffer `.err` executions across four native images.
5. Failure injection remains test-only. No probe symbol enters the public SDK, runtime symbol
   contract, ordinary product linkage, checked-in user example, or generated Fairy source.

## Consequences

- The native evidence chain now covers source-level `with_capacity` allocation-failure transport,
   source-level `resize_zeroed` growth-failure preservation of `len` and `capacity`, exact status
   `1`, and zero retained probe allocations after automatic Buffer cleanup.
- The raw C Buffer probe remains responsible for byte-level zeroing, copied bytes, exact pointer and
   metadata preservation, overlap append, overflow rollback, malformed-state rejection, and detailed
   unchanged-output checks. The new proof does not replace that matrix.
- This is test and evidence hardening only. Language remains `fairyc-language/8`, SDK remains
   `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim.
