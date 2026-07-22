# ADR 0148: Build and run local-exact workspaces on Windows

- Status: Accepted for prototype
- Date: 2026-07-22
- Product source: `docs/product/02-spec.md`
- CLI source: `docs/cli/command-contract.md`
- Package source: `docs/package/workspace-loader-v1.md`
- Implementation: `internal/cli/workspace_native.go`
- Native evidence: `cmd/fairyc-windows-host-smoke/`

## Context

`experimental check-workspace` proves that strict manifests, local members, the compiler-bundled
SDK, canonical locks, package-aware semantic analysis, and verified MIR fit together. It stops
before C emission. The existing Windows `build-target` and `run-target` commands can produce native
programs, but their positional entry plus repeated `--source` closure bypasses `fairy.toml`, package
identity, dependency visibility, and lock enforcement.

Stage 0 needs one honest project path before cache work is meaningful. Replacing the existing
source-closure commands or stabilizing `fairyc build` now would mix compatibility work with the
experiment. A separate package-aware command keeps the boundary explicit.

## Decision

1. Add unstable `experimental build-workspace` and `experimental run-workspace`. Both operate on
   the current directory and accept only the exact `x86_64-windows-msvc` target, explicit `low` or
   `optimized` profile, and optional `--locked`; build additionally requires one canonical
   no-overwrite `.exe` path below `build/`.
2. Both commands use the same workspace loader and lock policy as `check-workspace`. Ordinary mode
   creates or replaces a stale canonical lock after graph validation. `--locked` requires an exact
   existing lock and performs no lock write.
3. Source discovery, SDK injection, direct dependency visibility, build-local package IDs,
   diagnostics, semantic analysis, Flow, MIR lowering, and verification remain owned by their
   existing package-aware boundaries. The commands accept no positional source or ambient linker
   input.
4. The executable entry is exactly the root package's `main::main` with type `fn() -> i32`.
   Build-local qualification selects the root instance even when another package owns the same
   module or function spelling. That qualification remains private and is never serialized.
5. Verified package MIR enters the existing closed Windows C emitter, exact used-symbol target
   support resolver, x64 MSVC driver, PE inspector, and unwind inspector. This adds no new backend,
   runtime, ABI, foreign symbol, or standard-library capability.
6. Build retains one atomically promoted no-overwrite PE. Run executes the inspected in-memory
   artifact, rejects program output, reports canonical traps, and retains no executable.
7. `examples/workspace_quickstart` is the checked-in two-package program. The root imports one
   public function from a direct local dependency and returns 42.
8. The configured Windows smoke builds both profiles, runs the retained artifacts, performs one
   direct locked run, and performs another locked run through a separately built `fairyc.exe`.

## Consequences

- A user can now compile and run a real `.fy` project from `fairy.toml` without WSL.
- The native result is produced from package-aware verified MIR and MSVC, not by Go interpreting FY.
- The commands deliberately fix only `main::main`; configurable binaries, multiple targets,
  program I/O forwarding, tests, install, and stable command spelling remain separate decisions.
- Cold/warm build measurement now has a real package-aware Windows workload, but there is still no
  compiler-owned persistent artifact codec. ADR 0146's cache gate therefore remains closed.

## Verification

- unit tests cover closed option parsing, root-entry ownership/signature, package-qualified C
  emission, missing entry, non-Windows failure, and existing lock behavior;
- the full Go test, build, and vet suites cover the refactored shared Windows compilation path;
- the configured Windows smoke observes four package builds and two package runs returning 42,
  including a separately built compiler process, and inspects all resulting PE, COFF, and unwind
  structures.
