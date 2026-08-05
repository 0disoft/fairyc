# ADR 0257: Generated-Fairy String Append Failure Execution

- Status: Accepted
- Date: 2026-08-05

## Context

ADR 0247 published move-only `std::text::String` and `CString` whose copy, append, and
from-text operations are explicit fallible `Result` operations, and the raw C probe proved
allocation rollback at the byte level with a probe-only deterministic allocator. Product linkage
intentionally exposes no allocator-failure controls, so the `.err` arms produced from Fairy C for
`text::append` growth failure were not executed from generated source. ADR 0255 and ADR 0256
extended generated-Fairy allocation-failure execution to Region and Buffer; owned text remained the
remaining public resource family without a generated-Fairy failure execution.

## Decision

1. The Windows host smoke compiles one bounded Fairy C workspace through package loading, semantic
   Flow, verified MIR, and the x64 MSVC constrained-C backend. The source calls
   `text::copy("Fairy")` to construct a successful five-byte `String`, then calls
   `text::append(ref mut value, "Fairy")`.
2. The C harness renames only the generated C entry point and adds a C-only outer entry point. The
   generated function body, Result construction, borrowed views, and automatic String cleanup
   remain unchanged compiler output.
3. The outer entry arms the existing probe-only SystemHeap allocator with
   `fy_std_system_heap_probe_fail_after_allocations_v0(1)`, so the first copy allocation succeeds
   and the second append-growth allocation fails. The generated `.err(TextError)` arm requires
   `error_status == 1`, `error_allocation_status == 1`, `string_len == 5`,
   `!string_is_empty`, and a five-byte `string_as_str` view, then returns 42. After the generated
   main returns, the harness requires `fy_std_system_heap_probe_live_allocations_v0() == 0`,
   proving the automatic String drop released the surviving owner.
4. Both low and optimized MSVC profiles compile the probe as separate translation units, pass
   COFF, PE32+, and AMD64 unwind inspection, and execute the forced failure. This yields two forced
   generated-Fairy String append `.err` executions across two native images.
5. Failure injection remains test-only. No probe symbol enters the public SDK, runtime symbol
   contract, ordinary product linkage, checked-in user example, or generated Fairy source. The
   existing `generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)` closure is
   reused without a new linker or closure abstraction.
6. ADR 0258 narrows emission to the entry-reachable closure, so this probe's `unit.ForeignSymbols`
   is the exact minimal set of copy, append, and drop storage symbols; the host-smoke unit test
   freezes that set. Unnecessary SDK symbol emission is closed by that decision, not by this probe.

## Consequences

- The native evidence chain now covers source-level `String.append` growth-allocation failure
  transport, exact text and allocation statuses 1/1, preserved owner `len`, emptiness, and
  borrowed-view length, and automatic String drop back to zero live probe allocations.
- The raw C owned-text probe remains responsible for pointer, capacity, alignment, and byte
  `memcmp` preservation, NUL and UTF-8 rejection, malformed and null storage, deterministic size
  overflow, and repeated drop. The new proof does not replace that matrix.
- `text::copy` initial-allocation failure and `CString` allocation failure remain deferred to
  later generated-Fairy executions; the raw probe continues to cover their byte-level behavior.
- This is test and evidence hardening only. Language remains `fairyc-language/8`, SDK remains
  `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim.
