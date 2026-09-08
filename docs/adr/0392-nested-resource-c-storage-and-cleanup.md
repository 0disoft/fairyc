# ADR 0392: Nested resource C storage and cleanup

- Status: Accepted
- Date: 2026-09-09
- Contracts: `docs/language/0001-values-and-resources.md`, `docs/mir/drop-elaboration-v0.md`

## Decision

Extend the private C transport to acyclic directly nested resource fields. Resolve
dependencies before their containing types, retain declaration field order, and
reuse the target layout engine's size/alignment/offset checks. Duplicate definitions
and by-value cycles fail closed. This does not admit array, borrowed, tagged-resource
or other previously unsupported field representations.

Destruction follows the existing resource semantics: invoke the owner's cleanup,
then destroy resource fields in reverse declaration order. An empty owner cleanup
does not suppress field cleanup. Entry reachability follows nested cleanup symbols
even when the application never calls the child cleanup directly. Option/Result
payload cleanup remains conditional on the active variant. Whole-root moves still
transfer one obligation; no partial field moves are added.

A nonempty cleanup with resource fields must preserve changes it makes to its own
storage before field destruction. The private C implementation returns that
post-cleanup receiver to its drop caller, which then visits the fields. Its source
and MIR signature remain infallible void cleanup; no new user-callable function or
foreign ABI is published. Flat cleanup transport remains unchanged. Traps still
terminate without automatic cleanup.

## Evidence

- Dependency-order tests permute definitions and reject cycles/duplicates; cleanup
  reachability tests require the nested drop body through direct and tagged roots.
- `TestPrivateVectorNestedResourceDrops` executes empty outer cleanup, nonempty
  parent cleanup, reverse sibling order, active/inactive Option/Result paths and
  parent-body mutation visible to the later child cleanup under both MSVC profiles.
- `TestPrivateVectorNestedResourceDropTrace` compares exact source-linked native
  trace IDs with interpreter events, including a temporary dropped inside the
  owner's cleanup before its fields. The named Windows drop-trace profile is a
  closed test-only profile; production compilation stays uninstrumented.
- `TestWindowsVectorTokenStorage` uses ordinary SDK/CLI loading to store kind,
  trivia, source-byte bounds and owned cooked Buffer bytes in 33 tokens. Growth,
  shared nested-field reads, pop, clear, capacity retention and reuse execute under
  low and optimized MSVC without changing the proposed token representation.
- `TestPrivateVectorNestedResourceTokenOwnership` adds two injected growth failures,
  intact returned donors, pop/clear/refill cleanup order and independent allocator
  accounting: ten token cleanups, two allocation failures, equal allocation/free
  counts and zero live allocations after scope exit.

The native profiles share frontend, C emitter, runtime and MSVC lineage. These
tests prove the named storage/cleanup paths, not an independently compiled ABI,
sanitizer coverage, non-Windows runtime conformance or a completed lexer. The
storage probe does not tokenize input, reconstruct source or produce diagnostics.

## Compatibility and remaining work

This implements existing resource semantics without changing grammar, language 14,
SDK stage0.58 or runtime ABI 0. No portable resource layout or stable C API is
introduced. ADR 0391's token storage blocker is closed, but the self-hosting start
gate still requires a complete workload/CI assessment. Mutable field reborrows,
mutable vector access, iteration APIs and nested generic containers remain separate
work rather than prerequisites inferred from this test.
