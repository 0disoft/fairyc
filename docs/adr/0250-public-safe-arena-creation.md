# ADR 0250: Public safe Arena creation

- Status: Implemented local compiler and two-profile Windows native gate
- Date: 2026-07-31
- SDK source: `internal/sdkstdlib/sources/std/alloc.fy`
- Runtime source: `internal/cstdlib/stage0_arena.c`
- Example: `examples/std_alloc_quickstart/`
- Depends on: ADR 0042, ADR 0059, ADR 0086, ADR 0133, ADR 0147, ADR 0249

## Context

The target library already had failure-atomic Arena initialization, Region allocation, reset, and
drop, but the compiler-bundled SDK exported a one-field placeholder Arena with no constructor. Its
storage was too small for the selected C implementation, so connecting lexical Region lowering to
that placeholder would corrupt adjacent memory. Users also had no safe way to obtain the
`ref mut Arena` required by `region` syntax.

## Decision

1. SDK `0.0.0-stage0.10` replaces the placeholder with a move-only `std::alloc::Arena` resource
   whose private source storage is pointer, generation, and active-Region state compatible with the
   current x86_64 C providers. Cleanup calls `fy_std_arena_drop_v0` exactly once.
2. `std::alloc::create() -> Result<Arena, AllocError>` initializes canonical empty storage through
   `fy_std_arena_initialize_v0`. Failure returns the exact stable `u32` allocation status and leaves
   no live allocation or partially initialized resource.
3. `std::alloc::error_status(AllocError) -> u32` exposes that stable status without exposing the
   target allocator or Arena internals.
4. Arena fields remain module-private. Their shape is an SDK-to-runtime storage bridge, not a public
   C ABI, construction literal, capacity promise, growth policy, node format, or serialization
   contract.
5. The language revision remains `fairyc-language/7` and the runtime ABI remains
   `fairyc-runtime/0`. The SDK version and source-derived package identities change.
6. This decision is a prerequisite for native lexical Region lowering. It does not make the
   current infallible source spelling of `Region.place` acceptable, expose byte allocation methods,
   or claim that Region MIR is emitted to C.

## Evidence and limits

The bundled SDK is parsed, package-resolved, semantically checked, lowered, and verified with a
safe `alloc::create` caller. The checked-in quickstart requires exact initialize and drop symbols,
then builds and executes with result 42 under low and optimized MSVC profiles. The complete smoke
finishes with 61 structurally inspected PE images. Existing raw Arena probes retain the
allocation-failure, rollback, generation, stale-use, active-drop, and cleanup evidence.

Only the current x86_64 target profiles have a proven source-storage match; adding a narrower
pointer or `usize` target requires a fresh layout and native-crossing proof rather than reusing
this record by name.
