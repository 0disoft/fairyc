# ADR 0259: Generated-Fairy CString Failure Execution

- Status: Accepted
- Date: 2026-08-05

## Context

ADR 0247 published move-only `std::text::CString` whose construction is an explicit fallible
`Result` operation, and the raw C probe proved CString allocation rollback at the byte level with a
probe-only deterministic allocator. Product linkage intentionally exposes no allocator-failure
controls, so the `.err` arm produced from Fairy C for `cstring_from_text` allocation failure was not
executed from generated source. ADR 0255, ADR 0256, and ADR 0257 extended generated-Fairy
allocation-failure execution to Region, Buffer, and `String.append`; CString remained the public
owned-text construction path without a generated-Fairy failure execution.

## Decision

1. The Windows host smoke compiles one bounded Fairy C workspace through package loading, semantic
   Flow, verified MIR, and the x64 MSVC constrained-C backend. The source calls
   `text::cstring_from_text("Fairy")` and switches on the returned `Result<CString, TextError>`.
2. The C harness renames only the generated C entry point and adds a C-only outer entry point. The
   generated function body, Result construction, and automatic CString cleanup remain unchanged
   compiler output.
3. The outer entry arms the existing probe-only SystemHeap allocator with
   `fy_std_system_heap_probe_fail_after_allocations_v0(0)`, so the first CString storage
   allocation fails. The generated `.err(TextError)` arm requires `error_status == 1` and
   `error_allocation_status == 1`, then returns 42. After the generated main returns, the harness
   requires `fy_std_system_heap_probe_live_allocations_v0() == 0`, proving the automatic CString
   drop released the failed owner.
4. Both low and optimized MSVC profiles compile the probe as separate translation units, pass
   COFF, PE32+, and AMD64 unwind inspection, and execute the forced failure. This yields two forced
   generated-Fairy CString `.err` executions across two native images.
5. Failure injection remains test-only. No probe symbol enters the public SDK, runtime symbol
   contract, ordinary product linkage, checked-in user example, or generated Fairy source. The
   existing `generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)` closure is
   reused without a new linker or closure abstraction.
6. ADR 0258 narrows emission to the entry-reachable closure, so this probe's
   `unit.ForeignSymbols` is the exact minimal set `fy_std_cstring_from_text_storage_v0` and
   `fy_std_cstring_drop_storage_v0`, and the host-smoke unit test freezes that set.

## Consequences

- The native evidence chain now covers source-level `CString` construction-allocation failure
  transport with exact statuses 1/1 and automatic CString drop back to zero live probe
  allocations. The input `"Fairy"` is an interior-NUL-free five-byte UTF-8 value that reaches the
  allocation path.
- The raw C probe remains responsible for interior-NUL rejection, the exact NUL terminator byte,
  empty and overflow CString storage, malformed-state negatives, and repeated drop. The new proof
  does not replace that byte-level matrix; it executes only the generated-Fairy `.err` surface.
- `text::copy` initial-allocation failure remains deferred to a later generated-Fairy execution;
  the raw probe continues to cover its byte-level behavior.
- This is test and evidence hardening only. Language remains `fairyc-language/8`, SDK remains
  `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`. No API or ABI changed.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim.
