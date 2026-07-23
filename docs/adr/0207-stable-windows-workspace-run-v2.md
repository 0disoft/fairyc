# ADR 0207: Promote program-facing stable Windows workspace run v2

- Status: Accepted
- Date: 2026-07-23
- Supersedes: ADR 0177
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Implementation: `internal/cli/workspace_native.go`, `internal/cli/program_output.go`

## Context

Stable run v1 treated `main`'s `i32` as compiler result data, printed a compiler receipt, and
rejected every program byte. That made the first safe `std::io::write` API unusable through the
ordinary execution command and forced users to retain and find an executable merely to see output.
Adding a second stable `exec` command would preserve the wrong default and teach two commands for
one job.

The native runtime already appends one compiler-private trap receipt after program stderr. The
Windows execution driver already captures both streams separately under one shared bound and
preserves the complete 32-bit process status.

## Decision

1. Keep `fairyc run [--release] [--locked]`, its workspace, lock, profile, inspected temporary PE,
   no-retained-artifact, Windows/amd64, MSVC, timeout, and no-WSL boundaries.
2. After successful launch, forward captured program stdout and stderr to their matching compiler
   streams with no compiler success banner. Cross-stream ordering is unspecified because capture
   uses independent OS pipes. The shared 64 KiB output ceiling remains fail-closed.
3. For an ordinary return, preserve the exact 32-bit `main` status as the `fairyc` process status.
   Once execution begins, any status may be program-owned and may numerically equal a pre-execution
   compiler failure code.
4. Continue compiling the temporary run image with the private trap receipt. When the exit status
   and exact stderr suffix identify a trap, remove only that suffix, forward all preceding program
   stderr, append `trap: <category>\n`, and return stable trap status 5. An equal normal status without
   the exact suffix remains a normal program status.
5. Keep stdin closed and continue rejecting program arguments, a `--` delimiter, positional source,
   output, target, toolchain, profile-value, cache, JSON, configuration, and network surfaces.
6. If forwarding stdout or stderr fails after execution, return internal/output status 4. Earlier
   bytes and program effects cannot be rolled back. Output-limit, timeout, compile, link, inspection,
   package, lock, and source failures remain pre-execution compiler failures.
7. The private receipt is process-control metadata, not a public runtime ABI, authentication
   mechanism, or hostile-code boundary. Unsafe or foreign code is not sandboxed from imitating it.

## Evidence

- Unit tests distinguish an equal reserved normal status from an exact receipt-bearing trap,
  preserve program stderr before the receipt, retain malformed or mismatched receipt-like text as
  program output, preserve stream identity, and cover both compiler-output writer failures.
- The Windows smoke covers output-free status 42 in development and release, equal normal status
  131, a division-by-zero trap, exact safe stdout with status zero, lock restoration, and retained
  build-artifact immutability.
- A separately built `fairyc.exe` repeats output-free status propagation, generated-starter success,
  and safe `std::io` output through stable locked run.

## Consequences

`fairyc run` is now the ordinary program-facing execution command. Scripts must no longer parse a
`result:` line and must treat the post-launch process status as the program's status. Compiler
failure codes remain meaningful only when execution did not begin, except trap status 5 which is
identified by its canonical stderr line. Program arguments, stdin, unbounded streaming, live
cross-stream ordering, Linux stable execution, and target support remain later decisions.

## Rejected alternatives

- Add stable `fairyc exec`: duplicates the ordinary execution concept and leaves the wrong default.
- Add `--forward-output`: makes correct program behavior an opt-in mode and retains two exit models.
- Mix a result receipt after program stdout: corrupts byte-exact output and fails when output has no
  trailing newline.
- Interpret every reserved status as a trap: makes ordinary `i32` statuses in that range impossible.
