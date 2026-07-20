# ADR 0089: Standalone Windows compiler smoke

- Status: Accepted
- Date: 2026-07-20
- Host: `windows/amd64`
- Target: `x86_64-windows-msvc`

## Context

Calling `internal/cli.Run` from a Go smoke proves compiler behavior but does not prove that the
`cmd/fairyc` entry point can be built and used as a separate Windows process. A user-facing local
workflow should not need `go run` for every invocation. At the same time, committing a generated
compiler binary or calling it self-hosted would confuse a disposable Go bootstrap product with a
released Fairy C toolchain.

## Decision

1. Add `examples/windows_quickstart/main.fy`, a minimal scalar FY program whose `main() -> i32`
   returns 42 without FFI or provisional standard-library internals.
2. The configured Windows smoke builds `./cmd/fairyc` with the declared Go toolchain into a bounded
   temporary directory. Build subprocess output is captured with independent 64 KiB stdout and
   stderr ceilings and a 90-second deadline.
3. The smoke executes the resulting `fairyc.exe --help`, requires the `experimental run-target`
   surface, then invokes that executable in a second process to compile and run the quickstart for
   `x86_64-windows-msvc` at the optimized profile.
4. The standalone invocation must emit the target-labelled native result 42 with no stderr. Its
   generated target PE traverses the same COFF, PE, and Microsoft x64 unwind checks as every other
   `run-target` call.
5. Compiler and target temporary directories are removed. A manual local developer may build the
   ignored `build/fairyc.exe`; that artifact is never source truth, test evidence by existence, or a
   distribution promise.
6. The compiler executable is still produced by Go. The FY quickstart is compiled by that executable
   into a native MSVC target. This is bootstrap-process evidence, not Fairy C self-hosting. No WSL or
   Linux compatibility layer participates.

## Consequences

- Windows validation now crosses the real command entry point and OS process boundary before
  compiling a user-editable FY program.
- A user can build one local compiler executable and reuse it without paying `go run` startup on
  every command.
- Release packaging, installation, signing, PATH mutation, update delivery, stable CLI syntax, and
  self-hosting remain separate decisions.

## References

- `DEVELOPMENT.md`
- `docs/adr/0001-initial-architecture-boundaries.md`
- `docs/adr/0088-experimental-windows-target-run-command.md`
- `docs/ops/ci.md`
