# ADR 0095: Explicit multi-source interpreter run

- Status: Accepted
- Date: 2026-07-20
- Command: `experimental run`

## Context

The bounded MIR interpreter already executes a semantically analyzed program, but its CLI admitted
only one source file. After the Windows target commands gained an explicit module closure, users
could test multi-module code only by invoking the native MSVC path. Automatic import discovery and
package manifests remain outside the current phase.

## Decision

1. `experimental run` accepts one positional entry file plus zero or more repeated
   `--source <file.fy>` values. The first file selects the module whose `main() -> i32` is executed.
2. The command reuses the same explicit source loader and lowering closure as the Windows target
   commands: at most 64 files, the ordinary per-file limit, the existing aggregate byte limit,
   normalized duplicate rejection, source-root and symlink containment, source-relative module
   identity, and no discovery.
3. Only `--source` is admitted. Backend, target, compiler, profile, output, I/O binding, and host
   process options remain unavailable because this command is the bounded semantic interpreter.
4. The interpreter subset and observable result stay unchanged: `result: <i32>`, a canonical trap,
   source diagnostics, or an explicit capability error.
5. `examples/run/modules/` provides a two-file safe example. It requires the support file to be
   listed explicitly and works without MSVC, WSL, a C toolchain, or a host subprocess.

## Consequences

- Multi-module source logic can be tested quickly before running the native Windows command.
- A missing imported file fails visibly instead of being discovered from the filesystem.
- This does not define public `fairyc run`, package manifests, module search paths, caching,
  command arguments, source-visible I/O, or self-hosting.

## References

- `docs/adr/0016-stage-0-interpreter-fixture-contract.md`
- `docs/adr/0094-explicit-multi-source-windows-target-commands.md`
- `docs/cli/command-contract.md`
