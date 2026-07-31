# ADR 0251: Fallible Region Placement

- Status: Accepted
- Date: 2026-07-31

## Context

The public `std::alloc::Arena` constructor is fallible, but the compiler-known `Region.place`
operation still claimed to return `ref T` directly. A native implementation cannot keep that
promise: reserving storage and publishing the tracking node may fail. Treating either failure as a
trap would make ordinary allocation failure unrecoverable and would disagree with the rest of the
allocation surface.

The earlier direct-reference spelling also left MIR without an explicit failure type, encouraging a
backend to invent success or recover allocator policy from source-level context.

## Decision

1. Language revision `fairyc-language/8` changes the exact compiler-known signature to
   `Region.place(T) -> Result<ref T, std::alloc::AllocError>` for every fully initialized inert `T`.
2. A caller must handle the result with `switch`, propagate it with `?` from a compatible function,
   or state intentional loss with `discard`. There is no implicit unwrap and allocation failure does
   not trap.
3. The value expression is evaluated exactly once. Plain Region placement continues to reject every
   `NeedsDrop` value before executable lowering.
4. The success payload carries the Region origin through `Result`, locals, payload binding, calls,
   and `?`. It cannot outlive a local Region. The error payload carries no borrow origin.
5. MIR `region-place` retains both the placed owner type and the complete Result type. The verifier
   requires an immutable borrow of the owner type as the success payload and one concrete error type.
6. The bounded MIR interpreter models its abstract allocation as `.ok`; it does not claim a real
   allocator failure oracle. Native backends must derive `.ok` or `.err` from the target allocator
   status and must publish no borrow on failure.
7. SDK remains `0.0.0-stage0.10` and runtime ABI remains `fairyc-runtime/0`. The SDK already owns the
   public `std::alloc::AllocError` representation and exact status inspection; this decision changes
   the language operation, not the embedded SDK source bytes or public runtime ABI.

## Consequences

- Existing `let ref T value = region.place(input);` source must be rewritten to handle or propagate
  the Result.
- Package manifests, locks, whole-package cache keys, examples, and generated starter manifests use
  `fairyc-language/8`.
- Semantic, Region, MIR, verifier, interpreter, and workspace tests must agree on the tagged borrow
  provenance and the failure type.
- A C backend that cannot lower the complete Arena/Region lifecycle must reject the program rather
  than silently treating placement as infallible.
