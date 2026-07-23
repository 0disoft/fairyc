# ADR 0205: Add target-neutral Stage 0 stdout write

- Status: Accepted for prototype
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Package source: `docs/adr/0147-compiler-bundled-stage0-standard-library.md`
- Implementation: `internal/sdkstdlib/`, `internal/cstdlib/`, `internal/cstdlibcontract/`

## Context

The checked-in Windows output example proved effect propagation, unsafe containment, exact UTF-8
bytes, partial-progress transport, and real MSVC execution. Users still had to copy an example-local
unsafe declaration tied to `fy_std_windows_stdout_write_all_v0`. That is the wrong public boundary:
ordinary output should not expose raw pointers or force source code to name one operating system.

Adding formatting, implicit newlines, stdin, stderr, terminal detection, or a symbolic error enum in
the same step would freeze unrelated policy. The current backend also cannot place a user enum
inside `Result` without inventing an unsupported nested-enum layout.

## Decision

1. The compiler-bundled SDK version becomes `0.0.0-stage0.2` and embeds two deterministic sources:
   `std/alloc.fy` and `std/io.fy`. Its source digest and every dependent package and lock identity
   change with those exact bytes; the SDK identity and runtime ABI remain unchanged.
2. `std::io` publicly exports exactly
   `write(ref str) -> Result<usize, ConsoleError>`. It writes the text's exact UTF-8 bytes to stdout,
   appends no newline, allocates nothing, retains no borrow, performs no formatting or transcoding,
   and declares `@effects(console)`.
3. Success contains the exact number of bytes written. Failure contains a copy record with numeric
   `status` and `written`; `written` is the completed prefix. Status 1 is invalid input, 2 is
   unavailable stdout, 3 is an operating-system write failure, and 4 is zero progress. Safe `str`
   input cannot produce status 1 through the wrapper, but the raw boundary keeps the invariant.
4. Raw pointer conversion and the private `@repr(C)` result live only in the SDK module. They call
   target-neutral internal symbol `fy_std_stdout_write_all_v0`. This symbol and record layout are
   implementation ABI, not a public C header or stable native Fairy ABI.
5. The closed support resolver maps the same symbol to one target-owned component: POSIX `write` on
   `x86_64-linux-gnu` and Win32 `WriteFile` on `x86_64-windows-msvc`. Both return empty success before
   consulting stdout, reject nonempty null input, loop until completion or failure, preserve partial
   progress, and treat zero progress as failure. Linux retries `EINTR` and bounds each request by
   `SSIZE_MAX`; Windows bounds each request by `DWORD`.
6. The old `fy_std_windows_stdout_write_all_v0` symbol remains admitted for the existing experimental
   example. Both Windows symbols delegate to one implementation so their status and progress
   behavior cannot drift.
7. `examples/std_io_quickstart` is the safe user example. Its entry declares `@effects(console)`,
   calls only `std::io::write`, checks the returned byte count, prints `Fairy C says hi\n`, and
   returns 42. It contains no raw pointer, `unsafe`, foreign declaration, or platform name.
8. `println`, formatting, stderr, stdin, console encoding conversion, terminal capabilities,
   asynchronous I/O, cancellation, retry policy after a reported failure, symbolic error kinds, and
   a stable portable file package remain outside this decision.

## Consequences

- Ordinary Fairy C code can perform real stdout output through a C-friendly safe API while keeping
  capability authority visible at the function header.
- A target-neutral source name does not imply equal host internals or complete target support.
- Numeric errors are deliberately small and progress-preserving; a future symbolic taxonomy must
  preserve compatibility or introduce a separately versioned API.
- Stable `fairyc run` still rejects program streams because it owns a compiler result receipt.
  Users who need program output build and execute the retained artifact; the experimental
  `exec-target` command remains the direct-output execution surface.

## Verification

- SDK and workspace tests require exact two-source ordering, `std::io` provider resolution, effect
  checking, verified MIR, and automatic SDK identity/version propagation.
- Component tests require the target-neutral symbol to select Linux console on Linux and Windows
  console on Windows, with no unrelated component closure.
- The Linux native runner report v32 adds four configured `linuxConsoleExecutions` over Clang/GCC
  and low/optimized profiles. Its probe covers empty, invalid, successful redirected output, and
  closed-stdout behavior. No Linux execution receipt has been observed on this Windows host.
- The WSL-free Windows smoke builds the checked-in workspace through the stable command, executes
  the inspected PE with exact stdout and status 42, then repeats the locked build and execution
  through a separately built `fairyc.exe`.
