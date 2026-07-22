# ADR 0180: Stabilize workspace C emission v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Transport source: ADR 0137
- Prototype prerequisite: ADR 0027
- Implementation: `internal/cli/workspace_emit.go`, `internal/cbackend/`

## Context

The compiler already emits deterministic ISO C11 for both declared TargetSpec profiles and the
stable workspace path already owns manifest loading, generated-lock maintenance, package-aware
verified MIR, root-entry selection, and source/package failure categories. The missing public
boundary is a way to request that exact target-dependent application translation unit without also
compiling, linking, executing, or retaining a build artifact.

The old planned `emit <ast|hir|mir|c> [target]` spelling mixes disposable compiler internals with the
one representation that has an accepted transport contract. It also leaves target selection
ambiguous even though C scalar layout differs between LP64 and LLP64.

## Decision

Stabilize `fairyc emit c --target <target-id> [--locked]` as v1.

- `c` is the only stable form. AST, HIR, MIR, JSON, source maps, headers, objects, and executables
  remain outside this command.
- `--target` is mandatory and accepts the currently declared `x86_64-linux-gnu` and
  `x86_64-windows-msvc` profiles. The command never infers a target from the host.
- The current `fairy.toml` workspace, complete package graph, compiler-bundled standard-library
  package, root `main::main fn() -> i32`, and `fairy.lock` policy are the same as stable check/build.
  Without `--locked`, a missing or stale canonical lock is atomically updated before emission;
  `--locked` verifies it without writing.
- Success writes exactly one complete, deterministic, ANSI-free ISO C11 application translation
  unit to stdout with no banner, receipt, JSON envelope, absolute path, or original FY source text.
  The unit is the same target-selected application unit consumed by the corresponding backend path.
- The unit deliberately excludes the separate runtime and dependency-closed standard-library
  support translation unit. It may therefore contain external references and is not advertised as a
  standalone build artifact.
- The command writes no `.c`, header, object, executable, cache, or build directory and performs no
  compiler, linker, SDK, sysroot, WSL, subprocess, or network probe. Shell redirection remains the
  caller's output policy.
- Stable codes are 0 for success, 2 for invalid invocation, missing/invalid entry, unknown target,
  or an unsupported C-emission capability, 3 for source diagnostics, 4 for internal or output
  failure, and 7 for manifest, workspace, graph, or lock failure. Compilation and emission failures
  occur before stdout is touched; an external output writer may still accept a prefix before it
  returns a code-4 transport error, because a byte stream cannot be rolled back.
- Generated private symbols and whole-unit bytes are deterministic for one exact compiler and
  input, but they are disposable transport rather than a cross-release C API. Public `@repr(C)` and
  `extern C` boundaries remain governed by their separate ABI contracts.
- An output failure can occur after ordinary lock maintenance and does not roll that lock update
  back. `--locked` is the no-write form.

Adding another accepted TargetSpec is additive. Removing or reidentifying a target, changing the
mandatory target-selection grammar, or emitting a dialect other than the independently versioned
C11 transport requires a new compatibility decision.

## Evidence

- CLI tests emit the same workspace repeatedly, prove byte equality under `--locked`, and prove
  target-selected four-byte Windows `long` versus eight-byte Linux `long` assertions;
- failure tests cover missing workspace, syntax diagnostics, stable unsupported-semantic wording,
  missing entry, invalid forms and flags, help, zero-byte and partial-write output failure, lock
  immutability, and absence of a build directory;
- backend tests and emission fixtures own verified-MIR admission, deterministic C11 contents,
  target layouts, private symbols, traps, resources, tags, foreign declarations, and safety checks;
- full Go test, build, vet, Windows host smoke, and strict scaffold checks cover the shared boundary.
