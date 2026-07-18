# Provisional SystemHeap C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation source: `internal/cstdlib/`

## Context

The hosted executable phase needs an allocation base before `Arena`, `Buffer`, owned text, or Linux
`File` can have executable storage semantics. Putting allocation into `internal/cruntime` would
contradict the thin-runtime boundary, while exposing source-level library calls before modules,
effects, and the standard-library package instance are implemented would create a fake API.

The first slice therefore needs independent native evidence without pretending that a Fairy C
program can already construct `SystemHeap` or call its operations.

## Decision

1. `internal/cstdlib` owns target-standard-library C sources. `internal/cruntime` remains limited to
   compiler support and process adaptation.
2. The provisional `x86_64-linux-gnu` SystemHeap transport uses major-suffixed internal symbols and
   an explicit block record containing pointer, logical byte size, and requested alignment.
3. Stage 0 accepts only nonzero power-of-two alignments no greater than C
   `_Alignof(max_align_t)`. This closed fundamental-alignment subset covers every currently admitted
   language and C scalar layout. Over-aligned storage remains unsupported.
4. A zero-size request validates its alignment, does not call the platform allocator, and produces
   the normalized empty block. Deallocating an empty block is a no-op.
5. Nonzero allocation is fallible. Resize is implemented as allocate, bounded prefix copy, then
   release; public or internal use of raw `realloc` is excluded. Any validation or allocation
   failure leaves the original block unchanged.
6. A closed C probe exercises normalization, fundamental alignment, byte preservation across growth
   and shrink, unsupported-alignment rollback, zero-size resize, deallocation, and invalid output
   rejection plus malformed-block rejection without mutation. The Linux native lane compiles and
   executes it with Clang and GCC at low and optimized profiles.
7. `fairyc.native-c-report/v6` adds `systemHeapExecutions`. A successful configured run records four
   executions. This receipt remains unobserved until the hosted Linux job runs.
8. The helper is not linked into `experimental build`, exposed in generated C, or callable from
   Fairy C source yet. `complete-runtime` and `complete-standard-library` therefore remain missing
   target evidence.

## Rationale

This proves the allocation failure-atomicity that every later owned container depends on while
keeping the runtime/standard-library ownership line honest. Copy-and-release is slower than a
platform realloc fast path, but it makes old-storage preservation and alignment behavior explicit;
optimization can be revisited only after measurements and equivalent failure semantics exist.

## Consequences

- `Arena`, `Buffer`, text ownership, and `File` remain unimplemented.
- The current probe does not force a real out-of-memory event; deterministic OOM injection needs a
  later allocator test boundary rather than an enormous host allocation.
- Over-aligned allocations fail instead of delegating implementation-defined behavior to libc.
- No target maturity boolean changes until the complete standard-library surface and hosted receipts
  exist.
