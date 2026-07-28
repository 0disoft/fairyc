# ADR 0212: Forward stdin through stable Windows workspace run v3

- Status: Accepted
- Date: 2026-07-28
- Supersedes: ADR 0207
- Depends on: ADR 0211
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation: `internal/cli/run.go`, `internal/cli/workspace_native.go`, `internal/cdriver/windows_host.go`

## Context

Stable run v2 forwards program stdout and stderr but gives the temporary program immediate EOF on
stdin. ADR 0211 made a safe bounded `std::io::read_stdin` available, so requiring users to build and
locate a retained executable merely to supply input recreates the same split that v2 removed for
output.

Adding a flag or a second execution command would make normal process behavior opt-in. Pre-reading
stdin inside the compiler would also be wrong: it would delay interactive programs, duplicate bytes
in memory, consume input on compilation failure, and invent an arbitrary whole-input limit.

## Decision

1. Keep the exact `fairyc run [--release] [--locked]` spelling, workspace, lock, profile, temporary
   inspected PE, output, status, trap, timeout, Windows/amd64, MSVC, no-WSL, and no-retained-artifact
   contracts from v2.
2. The process stdin supplied to `fairyc` is attached to the temporary program after compilation,
   linking, and PE/unwind inspection succeed. The compiler does not pre-read, decode, retain, log,
   hash, or replay input bytes. Compilation, lock, source, link, and inspection failures consume no
   program input.
3. Input is streamed on demand and has no separate byte ceiling. The program determines how much it
   reads, while the existing bounded execution timeout remains the liveness ceiling. EOF, short
   reads, terminal behavior, and pipe behavior are observed by the program through the selected
   target adapter.
4. `cli.RunWithInput` owns this forwarding path. The embedding helper `cli.Run` continues to supply a
   closed empty reader so existing in-process callers that do not provide stdin keep deterministic
   EOF behavior.
5. Input transport failure is an execution failure and cannot roll back program effects. Program
   stdout and stderr remain separately captured under the shared 64 KiB ceiling and are not merged
   with compiler diagnostics.
6. Program arguments, a `--` delimiter, environment or configuration injection, live output
   interleaving, terminal mode management, cancellation API, Linux stable execution, and input
   forwarding for experimental target/workspace commands remain outside v3.

## Compatibility

The command spelling and accepted arguments do not change. After successful launch, inherited stdin
becomes stable run v3 compatibility data instead of guaranteed EOF. Programs that never read stdin
are unchanged. Scripts that intentionally relied on immediate EOF must redirect an empty stream.
Pre-launch compiler failure codes and post-launch program/trap status rules remain unchanged.

## Evidence

- A CLI regression supplies a counting reader to an invalid workspace and requires zero reads before
  the missing-entry failure.
- The Windows host smoke runs `examples/std_io_quickstart` through in-process
  `cli.RunWithInput` with `stdin!\n` and requires the exact echo after the existing stdout bytes.
- A separately built `fairyc.exe` repeats `run --release --locked` with the same input and exact
  separated stdout/stderr observation.
- Existing output-free, reserved-status, trap, timeout, output-ceiling, retained-artifact, lock, and
  compiler-failure tests remain unchanged.

## Consequences

`fairyc run` now behaves like the ordinary native program launcher for all three standard byte
streams without adding syntax or flags. The compiler stays out of the data path: it owns launch and
bounded observation, while the Fairy C program owns how and when stdin is consumed.

## Rejected alternatives

- Add `--stdin` or `--forward-stdin`: makes normal process behavior an opt-in mode.
- Add another stable `exec` command: duplicates the ordinary run concept.
- Read all stdin before compilation: consumes input on failure, breaks interaction, and creates an
  arbitrary memory and size policy.
- Give every target command stdin: broadens unstable evidence commands without a separate contract.
