# ADR 0260: Generated-Fairy String Copy Failure Execution

- Status: Accepted
- Date: 2026-08-05

## Context

ADR 0247 published move-only `std::text::String` whose copy is an explicit fallible `Result`
operation, and the raw C probe proved String copy allocation rollback at the byte level with a
probe-only deterministic allocator. Product linkage intentionally exposes no allocator-failure
controls, so the `.err` arm produced from Fairy C for `text::copy` initial allocation failure was
not executed from generated source. ADR 0255, ADR 0256, ADR 0257, and ADR 0259 extended
generated-Fairy allocation-failure execution to Region, Buffer, `String.append`, and CString;
`text::copy` initial allocation remained the last public owned-text construction path without a
generated-Fairy failure execution.

## Decision

1. The Windows host smoke compiles one bounded Fairy C workspace through package loading, semantic
   Flow, verified MIR, and the x64 MSVC constrained-C backend. The source calls
   `text::copy("Fairy")` and switches on the returned `Result<String, TextError>`.
2. The C harness renames only the generated C entry point and adds a C-only outer entry point. The
   generated function body, Result construction, and automatic String cleanup remain unchanged
   compiler output.
3. The outer entry arms the existing probe-only SystemHeap allocator with
   `fy_std_system_heap_probe_fail_after_allocations_v0(0)`, so the first String storage allocation
   fails. The generated `.err(TextError)` arm requires `error_status == 1` and
   `error_allocation_status == 1`, then returns 42. After the generated main returns, the harness
   requires `fy_std_system_heap_probe_live_allocations_v0() == 0`, proving the automatic String
   drop released the failed owner.
4. Both low and optimized MSVC profiles compile the probe as separate translation units, pass
   COFF, PE32+, and AMD64 unwind inspection, and execute the forced failure. This yields two forced
   generated-Fairy String copy `.err` executions across two native images.
5. Failure injection remains test-only. No probe symbol enters the public SDK, runtime symbol
   contract, ordinary product linkage, checked-in user example, or generated Fairy source. The
   existing `generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)` closure is
   reused without a new linker or closure abstraction.
6. ADR 0258 narrows emission to the entry-reachable closure, so this probe's
   `unit.ForeignSymbols` is the exact minimal set `fy_std_string_copy_storage_v0` and
   `fy_std_string_drop_storage_v0`, and the host-smoke unit test freezes that set.

## Consequences

- The native evidence chain now covers source-level `text::copy` initial-allocation failure
  transport with exact statuses 1/1 and automatic String drop back to zero live probe
  allocations. This is a distinct proof from ADR 0257, which keeps the successful copy allocation
  and fails only the append growth allocation.
- The raw C probe remains responsible for failed-output null/len/capacity/alignment and byte-level
  empty preservation, malformed and null storage, deterministic size overflow, and repeated drop.
  The new proof executes only the generated-Fairy `.err` surface and does not replace that matrix.
- With this proof, every public owned-text construction and append path now has a generated-Fairy
  allocation-failure execution: String copy (ADR 0260), String append (ADR 0257), and CString
  construction (ADR 0259).
- This is test and evidence hardening only. Language remains `fairyc-language/8`, SDK remains
  `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`. No API or ABI changed.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim.
