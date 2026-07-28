# ADR 0211: Add target-neutral Stage 0 stdin byte reads

- Status: Accepted for prototype
- Date: 2026-07-28
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Package source: `docs/adr/0147-compiler-bundled-stage0-standard-library.md`
- Depends on: ADR 0125, ADR 0126, ADR 0205, ADR 0209, ADR 0210
- Implementation: `internal/sdkstdlib/`, `internal/cstdlib/`, `internal/cstdlibcontract/`

## Context

Fairy C can already construct a safe mutable byte view over a mutable fixed `u8[N]` array, and the
target standard-library boundary already owns process console handles. Reading stdin still required a
user-visible raw pointer crossing or a platform-specific wrapper. That contradicts the purpose of the
compiler-bundled `std::io` package and leaves even a bounded command-line byte reader outside safe
source.

This decision must not smuggle in text decoding, buffering, line handling, an owned stream object, or
stable CLI stdin forwarding. A read is allowed to return fewer bytes than the buffer can hold, and an
ordinary end-of-file is not an error.

## Decision

1. The compiler-bundled SDK version becomes `0.0.0-stage0.5`. The exact SDK source digest, package
   instance, public interface hash, dependent identities, and generated lock bytes change. The SDK
   identity, language revision, runtime ABI identity, package UID, and embedded source paths remain
   unchanged.
2. `std::io` adds
   `read_stdin(ref mut u8[] buffer) -> Result<usize, ConsoleReadError>`. It declares
   `@effects(console)`, allocates nothing, retains no borrow, and performs at most one bounded
   operating-system read. Success is the exact initialized prefix length; success with zero bytes is
   end-of-file. Callers that need to fill a buffer or consume a stream must loop explicitly.
3. `ConsoleReadError` is distinct from output's `ConsoleError`. It contains numeric `status` and
   `read` fields so input does not misuse the output-specific `written` name. Raw status 1 is invalid
   nonempty input, status 2 is an unavailable stdin stream, and status 3 is an operating-system read
   failure. The safe wrapper cannot construct invalid nonempty storage.
4. Empty input succeeds with zero without consulting stdin. The private raw symbol is
   `fy_std_stdin_read_v0`; its C transport is `{ uint32_t status; size_t read; }`. It joins the existing
   target-owned console component and does not enter the thin runtime or public C ABI.
5. The POSIX adapter bounds one request by `SSIZE_MAX`, retries only `EINTR`, maps `EBADF` to status 2,
   and treats `read` returning zero as EOF. The Win32 adapter bounds one request by `UINT32_MAX`, uses
   `GetStdHandle(STD_INPUT_HANDLE)` and `ReadFile`, maps invalid handles to status 2, and treats
   `ERROR_BROKEN_PIPE` and `ERROR_HANDLE_EOF` as EOF.
6. `examples/std_io_quickstart` reads into one eight-byte fixed array, validates the returned count,
   forms an immutable initialized-prefix view, and echoes exactly that prefix through `write_bytes`.
   It remains free of raw pointers, unsafe blocks, foreign declarations, allocation, and platform
   names.
7. Stable `fairyc run` remains a closed-stdin command. This decision proves input through direct
   execution of a built artifact; forwarding caller stdin through the compiler driver requires a
   separate CLI compatibility decision.
8. No UTF-8 validation, text or line reader, delimiter scanning, buffering, implicit retry-to-fill,
   seek, terminal mode, cancellation, asynchronous I/O, program arguments, or owned stream handle is
   added.

## Compatibility

The function and error type are source-additive. Exact Stage 0 SDK selection is intentionally not
lock-compatible: ordinary `fairyc check` refreshes a lock that names the prior source digest, package
instance, interface hash, and version, while `--locked` rejects it without substitution. The runtime
ABI identity remains `fairyc-runtime/0`. The private target support-symbol set gains
`fy_std_stdin_read_v0`, but that symbol is pre-1 implementation compatibility data rather than a
public Fairy or C ABI.

## Evidence

- Compiler integration tests lower the public call through verified MIR and require `E7102` when the
  caller omits the `console` effect.
- Component-contract tests require the new symbol to select only the target's console component on
  Linux and Windows.
- The Linux probe covers empty input, invalid raw input, an exact five-byte pipe payload without
  overwriting the unused suffix, EOF, and an unavailable descriptor. Four configured Clang/GCC
  low/optimized executions remain unobserved on this Windows host.
- The WSL-free Windows smoke builds the checked-in workspace with MSVC, directly executes the PE with
  EOF, a short `stdin!\n` payload, and a nine-byte payload, and observes respectively no echo, the
  exact short echo, and only the first eight bytes. A separately built `fairyc.exe` repeats the
  artifact and short-input path.

## Consequences

Safe Fairy C programs can now perform small allocation-free binary reads without confusing EOF with
failure or pretending bytes are text. The deliberately single-read contract stays predictable for
pipes and terminals and leaves buffering and stable CLI forwarding as separate, testable decisions.
