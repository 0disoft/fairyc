# ADR 0209: Add target-neutral Stage 0 stderr write

- Status: Accepted for prototype
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Package source: `docs/adr/0147-compiler-bundled-stage0-standard-library.md`
- Depends on: ADR 0205, ADR 0207
- Implementation: `internal/sdkstdlib/`, `internal/cstdlib/`, `internal/cstdlibcontract/`

## Context

Stable run v2 already preserves bounded stderr as a distinct program stream, but safe Fairy source
could write only stdout. Logging an error therefore still required a target-specific unsafe foreign
declaration even though the target adapters, effect, progress result, and failure policy were already
settled for stdout.

Formatting, stdin, program arguments, terminal detection, and a symbolic error taxonomy remain
separate policy decisions. A general stream object would add ownership and capability questions that
are unnecessary for the two process-owned output streams.

## Decision

1. The compiler-bundled SDK version becomes `0.0.0-stage0.3`. Its two embedded source paths, SDK
   identity, language revision, runtime ABI identity, and package UID remain unchanged; the exact
   source digest, package instance, public interface hash, dependent identities, and generated lock
   bytes change.
2. `std::io` adds
   `write_stderr(ref str) -> Result<usize, ConsoleError>`. Like `write`, it declares
   `@effects(console)`, writes exact UTF-8 bytes, appends no newline, allocates nothing, retains no
   borrow, and reports the completed prefix on failure.
3. Both public functions reuse one private scalar result conversion. The stderr wrapper alone calls
   target-neutral internal symbol `fy_std_stderr_write_all_v0`; raw pointers, the `@repr(C)` record,
   and target linkage remain private implementation details.
4. Linux routes stdout and stderr through one bounded POSIX write loop selected by file descriptor.
   Windows routes both through one bounded `WriteFile` loop selected by `STD_OUTPUT_HANDLE` or
   `STD_ERROR_HANDLE`. Empty input succeeds before handle lookup, nonempty null input is invalid,
   `EINTR` is retried on Linux, requests retain their existing platform bounds, and the first
   non-retryable failure preserves progress.
5. Status 2 now means that the selected output stream is unavailable. Statuses 0, 1, 3, and 4 and
   the result layout retain their ADR 0205 meanings. The closed support resolver maps both
   target-neutral symbols to the same target-owned console component. The historical Windows-only
   stdout symbol remains available only for its existing experimental example.
6. `examples/std_io_quickstart` writes `Fairy C says hi\n` to stdout and
   `Fairy C warns too\n` to stderr, verifies both completed byte counts, and returns zero. It remains
   free of raw pointers, unsafe blocks, foreign declarations, and platform names.
7. No formatting, implicit newline, stdin, program argument, stream handle, terminal encoding,
   asynchronous I/O, cancellation, logging framework, or symbolic error-kind API is added.

## Compatibility

The new Fairy function is source-additive for programs that do not use it. Exact Stage 0 SDK
selection is intentionally not lock-compatible: an existing generated lock names the previous SDK
version, source digest, package instance, and interface hash and must be refreshed by ordinary
`fairyc check`. `--locked` continues to reject that stale lock rather than silently substituting
bytes. The runtime ABI identity string stays `fairyc-runtime/0`; the new C symbol is pre-1 internal
implementation compatibility data, not a stable public C or native Fairy ABI.

## Evidence

- Compiler integration tests compile both public calls through verified MIR and require `E7102`
  when either call lacks the `console` effect.
- Component tests require both symbols to select exactly one Linux or Windows console component and
  reject unrelated or unknown standard-library symbols as before.
- The Linux separate-translation-unit probe exercises empty, invalid, redirected success, and
  unavailable output for both descriptors under its configured Clang/GCC low/optimized matrix.
  Those executions remain unobserved on this Windows host.
- The WSL-free Windows smoke observes exact separated stdout and stderr through stable locked run,
  a retained inspected PE, and the same two paths through a separately built `fairyc.exe`.

## Consequences

Safe Stage 0 programs can now distinguish ordinary output from diagnostics without crossing unsafe
or naming an operating system. The two public functions deliberately remain byte-oriented and
explicit; richer console ergonomics require their own contract and evidence.
