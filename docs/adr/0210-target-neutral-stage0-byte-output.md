# ADR 0210: Add target-neutral Stage 0 byte output

- Status: Accepted for prototype
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Package source: `docs/adr/0147-compiler-bundled-stage0-standard-library.md`
- Depends on: ADR 0205, ADR 0209
- Implementation: `internal/sdkstdlib/`, `internal/cbackend/`

## Context

The safe console surface can write valid UTF-8 text to both process-owned output streams. Fairy C
already has immutable `ref u8[]` views with exact length and no text promise, but emitting those
bytes still requires unsafe pointer conversion outside the standard library. Treating binary output
as text would either reject useful non-UTF-8 data or lie about its encoding.

The target adapters already accept an address and byte count, preserve partial progress, and make no
text decision. Adding another runtime symbol or a general stream object would duplicate settled
mechanics and prematurely open ownership, terminal, and capability policy.

## Decision

1. The compiler-bundled SDK version becomes `0.0.0-stage0.4`. The exact SDK source digest, package
   instance, public interface hash, dependent identities, and generated lock bytes change. The SDK
   identity, language revision, runtime ABI identity, package UID, and embedded source paths remain
   unchanged.
2. `std::io` adds
   `write_bytes(ref u8[]) -> Result<usize, ConsoleError>` and
   `write_stderr_bytes(ref u8[]) -> Result<usize, ConsoleError>`. Both declare
   `@effects(console)`, append no newline, allocate nothing, retain no borrow, and write the exact
   view bytes without UTF-8 validation, transcoding, or terminator insertion.
3. The byte operations reuse the existing private result conversion and the existing
   `fy_std_stdout_write_all_v0` and `fy_std_stderr_write_all_v0` symbols. No target C source, support
   resolver entry, result layout, numeric status, or runtime ABI identity changes.
4. Generated C must declare private immutable-byte, mutable-byte, and UTF-8 view transports before
   any prototype that uses them, even when the type appears only in an unused internal function
   signature. Transport need is collected from every internal result and parameter type rather than
   inferred only from literals or borrow operations.
5. `examples/std_io_quickstart` writes one text line and one byte-view line to each output stream,
   verifies every completed byte count, and returns zero only when all four operations succeed. It
   remains free of raw pointers, unsafe blocks, foreign declarations, and platform names.
6. No formatting, implicit newline, stdin, arguments, terminal detection, terminal rendering or
   encoding promise, owned stream handle, asynchronous I/O, cancellation, logging framework, or
   symbolic error-kind API is added.

## Compatibility

The two functions are source-additive. Exact Stage 0 SDK selection is intentionally not
lock-compatible: ordinary `fairyc check` refreshes a lock that names the previous source digest,
package instance, interface hash, and version, while `--locked` rejects it without substitution.
The runtime ABI identity remains `fairyc-runtime/0`, and the target support-symbol set is unchanged.

## Evidence

- Compiler integration tests lower all four public calls through verified MIR, use NUL and non-UTF-8
  bytes for the byte-view calls, and require `E7102` when any call lacks the `console` effect.
- C-backend regressions require signature-only immutable-byte, mutable-byte, and UTF-8 view
  transports to be emitted before their first parameter use.
- The WSL-free Windows smoke compiles the SDK through MSVC and observes exact text and byte-view
  lines on separated stdout and stderr through stable locked run, retained-image execution, and the
  same two paths through a separately built `fairyc.exe`.
- The target C adapters and their configured Linux probe matrix are unchanged. Linux native
  execution remains unobserved on this Windows host.

## Consequences

Safe Fairy C code can emit arbitrary immutable bytes without pretending they are text or opening an
unsafe boundary. Text and binary intent stay visible in the function name and parameter type, while
richer I/O policy remains separately decidable.
