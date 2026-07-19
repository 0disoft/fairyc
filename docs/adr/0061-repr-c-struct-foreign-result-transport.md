# Repr(C) Struct Foreign Result Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`

## Context

The source backend can already define, construct, assign, and read nonempty acyclic `@repr(C)`
records and can pass their addresses to C. It nevertheless rejects the same record as a foreign
result. This blocks real standard-library functions such as owned-text operations whose C ABI
returns a small status record by value.

Opening every aggregate call position at once would blur two different ABI risks. A result is
written into one typed destination, while aggregate parameters add evaluation, copying, register
exhaustion, and ownership questions. They remain separate gates.

## Decision

1. A foreign C result may be a nonempty acyclic `@repr(C)` struct already admitted by the constrained
   record field and layout rules. A non-`@repr(C)` struct result remains rejected explicitly.
2. Foreign aggregate parameters remain rejected. This decision does not admit by-value resources,
   enums, arrays, `Option`, `Result`, String, Buffer, or other language-owned representations.
3. MIR retains the nominal result type on the call destination. Generated C emits the exact public
   struct tag in the prototype and assigns the returned value directly to the typed local.
4. The target layout engine remains the source for generated size, alignment, and field-offset
   assertions. No width-based alias or flattened surrogate replaces the source record.
5. The native source fixture receives two results from a separately compiled hand-written C unit:
   an 8-byte two-`u32` record classified into a return register and a 20-byte five-`u32` record using
   the SysV AMD64 hidden-memory result path.
6. The generated Fairy caller reads every field and exits 42 only when both results retain their
   exact values. Clang-caller/GCC-callee and the reverse each run at low and optimized profiles.
7. `fairyc.native-c-report/v25` adds `foreignCStructResultCallExecutions`, expected to be four only
   after all compiler directions and profiles execute successfully.
8. The interpreter remains fail-closed on the foreign call. This transport does not establish a
   public C ABI support level, automatic standard-library linkage, foreign aggregate parameters,
   callbacks, exports, or owned-text source semantics.

## Consequences

- The next owned-text slice can receive its exact two-`u32` C status record without inventing an
  integer wrapper or changing the production helper signature.
- Register and hidden-memory results now have generated-Fairy crossing coverage, but mixed
  INTEGER/SSE results, nested result interaction, and aggregate parameters remain separate tests.
- Native execution remains configured but unobserved on this Windows host.
