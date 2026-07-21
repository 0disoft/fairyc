# ADR 0132: Add deterministic owned-text allocation-failure probes

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Related decisions: `docs/adr/0131-safe-owned-text-fy-lifecycle.md`,
  `docs/adr/0133-deterministic-arena-allocation-failure-probe.md`,
  `docs/adr/0134-deterministic-buffer-allocation-failure-probe.md`

## Context

The owned-text implementation maps allocator failure into an explicit status and promises that a
failed constructor leaves canonical empty output while a failed String append preserves the prior
allocation, fields, and bytes. Ordinary native execution cannot reliably force the host allocator to
return null, so the earlier probe exercised nearby rollback paths but did not prove this exact branch.

Replacing the product allocator, exporting a general fault-injection API, or depending on global CRT
interposition would widen the runtime boundary and make the test platform-dependent. The probe needs
one deterministic failure point without changing product linkage or the standard-library ABI.

## Decision

1. `SystemHeap` routes its internal allocation and release calls through two compile-time macros.
   Their defaults remain the C runtime `malloc` and `free`; product behavior and symbols are unchanged.
2. `stage0_system_heap_probe_allocator.c` is a shared probe-only prefix. It overrides those macros
   with a one-shot fail-after-N-successful-allocations adapter and exposes only probe control plus a
   live-allocation count to the separately compiled caller.
3. Product support resolution never includes the adapter or its control symbols. The adapter is
   concatenated only by closed probe implementation constructors; owned-text uses
   `OwnedTextProbeImplementationSource` and its Windows MSVC counterpart.
4. The raw probe forces initial String and CString allocation failure and requires canonical empty
   output plus zero live allocations. It then forces String append reallocation failure and requires
   the original pointer, fields, bytes, and one live allocation to remain unchanged before drop
   returns the count to zero.
5. The Linux native runner uses the portable instrumented callee for its existing four
   compiler/profile executions. The WSL-free Windows smoke uses the target-specific 16-byte-alignment
   callee for two separate-translation-unit MSVC executions. Source-call and product builds continue
   to use the uninstrumented implementation.
6. This is not a public allocator interface, thread-safe fault-injection service, source-level Fairy
   API, stable native ABI, or proof for every allocation site. It is a closed single-process probe.
7. `fairyc.native-c-report/v31` is not bumped: no field or count meaning is added. The existing
   `ownedTextExecutions` success condition is strengthened in place, as earlier probe hardening did.

## Consequences

- Allocation-failure preservation no longer depends on exhausting host memory or an unverified code
  reading.
- Product executables still call the ordinary target allocator and contain no probe control state.
- The live-allocation oracle detects leaked prior or replacement storage across the forced failures
  and final cleanup, but it is not a general heap-leak detector or concurrency test.

## Verification

- Go source-contract tests require macro defaults, adapter-before-fallback ordering, deterministic
  source construction, Windows alignment injection, and absence of probe controls from product source;
- the raw C probe checks exact failure statuses, storage and byte preservation, and live counts;
- the Windows host smoke compiles caller and instrumented callee separately, inspects both COFF
  objects plus linked PE/unwind data, and executes low and optimized profiles without WSL;
- the configured Linux native runner consumes the same probe and instrumented portable callee when
  its glibc, Clang, and GCC lane runs.
