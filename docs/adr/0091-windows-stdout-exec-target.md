# ADR 0091: Windows stdout and exec-target

- Status: Accepted
- Date: 2026-07-20
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental exec-target`

## Context

`run-target` intentionally rejects program output and prints a compiler-owned native-result receipt.
Forwarding bytes through that command would mix program output with its receipt and make an absent
trailing newline change framing. The runtime also needs to expose partial output progress instead of
pretending that one operating-system write is all-or-nothing.

## Decision

1. Add `experimental exec-target <file.fy> --target x86_64-windows-msvc --profile
   <low|optimized>`. It accepts no output path, compiler override, companion input, linker option,
   stdin, program arguments, or ambient target.
2. Compilation, exact standard-library resolution, MSVC discovery, COFF inspection, PE inspection,
   unwind inspection, timeout, temporary-file cleanup, and the shared 64 KiB output ceiling are the
   same as `run-target`.
3. The command forwards captured stdout and stderr to their matching CLI streams without a banner or
   result receipt. It preserves the program's full 32-bit Windows exit status as the CLI status.
   Cross-stream order is unspecified.
4. Admit exactly `fy_std_windows_stdout_write_all_v0(const uint8_t *, size_t)` for the Windows target.
   It returns a 16-byte `@repr(C)` result with `uint32_t status`, four bytes of target padding, and
   `size_t written`. The Linux C profile rejects this target-only symbol before emission.
5. Empty input succeeds without resolving stdout. Nonempty null input returns status 1. Missing
   stdout returns 2, `WriteFile` failure returns 3, zero progress returns 4, and complete success is
   status 0 with exact progress. The implementation loops in `DWORD`-bounded chunks and never hides
   a completed prefix.
6. `examples/windows_output/main.fy` observes empty and invalid-input results, writes the exact 16
   ASCII bytes `Fairy C says hi\n`, and exits 42. The configured Windows smoke requires those exact
   streams and status through `exec-target`.
7. No WSL or Linux compatibility layer participates. The raw function remains unsafe and does not
   establish portable `std::io`, Unicode console behavior, stdin, stderr writes, formatting, or
   target support.

## Consequences

- A Windows developer can now observe real FY program output instead of only a compiler receipt.
- Inspection-oriented `run-target` remains zero-output and keeps its existing result/trap contract.
- A future safe wrapper must supply a source-visible length, borrow/effect contract, typed error,
  and portable per-target implementation rather than exposing this private record directly.

## References

- `docs/adr/0090-bounded-windows-process-stream-capture.md`
- `docs/cli/command-contract.md`
- `docs/cli/output-and-exit-codes.md`
- `docs/runtime/runtime-stdlib-v0.md`
