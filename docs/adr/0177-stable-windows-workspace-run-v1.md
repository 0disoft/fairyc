# ADR 0177: Stabilize the native Windows workspace run

- Status: Accepted
- Date: 2026-07-22
- Superseded by: ADR 0207
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Prerequisites: ADR 0148, ADR 0175, and ADR 0176
- Implementation: `internal/cli/workspace_native.go`

## Context

Stable check and build now provide honest current-workspace verification and one retained native
artifact. Running a program still requires the backend-facing target and profile vocabulary of
`experimental run-workspace`, even though ordinary users have no useful choice among those values.

Treating the Fairy program's `i32` result as the compiler process status would make a successful
example returning 42 look like a failed compiler invocation. Retaining another executable would
also duplicate build output, while forwarding program streams or arguments would claim runtime
capability that Stage 0 does not yet provide.

## Decision

Implement `fairyc run [--release] [--locked]` as stable workspace run v1.

- It uses the same current-directory manifest, generated lock, root `main::main`, development and
  release profile meanings, and closed `x86_64-windows-msvc` pipeline as stable build.
- Ordinary mode may maintain the canonical lock before later compilation or execution fails;
  `--locked` performs no write.
- Positional sources, output, target, compiler, linker, profile-value, cache, JSON, configuration,
  network, stdin, program arguments, and a program-stream forwarding delimiter are rejected.
- The inspected executable is created in compiler-owned temporary storage, executed under the
  existing bounded capture and timeout policy, and removed. Stable run never creates, replaces, or
  consumes `build/<package-name>.exe`.
- Stage 0 admits no native program stream contract. Any captured stdout or stderr is therefore a
  build/execution failure rather than silently mixed with the compiler receipt.
- A normal `i32` return writes exactly
  `result: <i32> (x86_64-windows-msvc, <development|release>, lock <state>)` and the compiler exits
  with 0. The value is program data, not the compiler status. The run-only scalar runtime flushes a
  compiler-private trap receipt before its reserved trap exit; absence of that exact receipt keeps
  every normal `i32`, including values that share the same OS status, distinguishable from a trap.
  Ordinary retained build artifacts use the receipt-free runtime.
- A recognized language trap writes exactly `trap: <category>` to stderr and exits with 5. Stable
  codes are 0, 2, 3, 4, 5, 6, and 7.
- A compiler-output write failure after execution returns 4 and cannot roll back program effects.

This stabilizes one local command contract, not Windows target support, native program I/O,
argument transport, distribution, self-hosting, or persistent-cache defaults.

## Evidence

- unit tests cover the shared closed flags, help routing, package/source failure categories, root
  entry rejection, and absence of retained output before toolchain resolution;
- the configured Windows smoke executes development and release success, normal result 131, and a
  division-by-zero trap with the same OS status, restores the canonical lock after source mutation,
  and proves the retained stable-build artifact is byte-identical before and after run;
- a separately Go-built `fairyc.exe` performs locked release execution with exact result 42 while
  leaving the retained artifact unchanged;
- full Go test, build, vet, and strict scaffold checks cover the shared pipeline and contracts.
