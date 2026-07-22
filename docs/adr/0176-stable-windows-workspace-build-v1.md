# ADR 0176: Stabilize the native Windows workspace build

- Status: Accepted
- Date: 2026-07-22
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Prerequisites: ADR 0148 and ADR 0175
- Implementation: `internal/cli/workspace_native.go`, `internal/cli/build.go`

## Context

The package-aware experimental build already proves the complete local workspace, lock, verified
MIR, constrained C11, closed standard-library linkage, MSVC, PE, and unwind path. Its required
target, internal profile name, and output path make ordinary development needlessly ceremonial.
Retaining a new path for every build would also recreate the unbounded artifact growth Fairy C is
intended to avoid.

Promoting the experimental spelling unchanged would freeze backend-facing choices as user-facing
policy. Conversely, hiding an unbounded cache or claiming target support would exceed the current
evidence.

## Decision

Implement `fairyc build [--release] [--locked]` as stable workspace build v1.

- The current directory must contain the strict root `fairy.toml`; positional sources, output,
  target, compiler, linker, profile-value, cache, JSON, network, and stdin inputs are rejected.
- The root package's validated kebab-case name selects the only retained product path:
  `build/<package-name>.exe`.
- Default mode uses the existing low-optimization development profile. `--release` selects the
  existing ordinary optimized profile without LTO. Repeating either flag is invalid.
- Ordinary mode maintains canonical `fairy.lock`; `--locked` requires an exact existing lock and
  writes nothing.
- The root entry remains exactly `main::main` with type `fn() -> i32`. The command reuses the
  package-aware verified-MIR and closed `x86_64-windows-msvc` C11/MSVC pipeline.
- Compilation and PE/unwind inspection finish in temporary storage. A nonempty synced staging file
  then replaces the prior default artifact through a source-root-contained rename. A failed compile,
  inspection, staging write, or replacement preserves the previous successful artifact.
- Success writes exactly
  `built: <path> (x86_64-windows-msvc, <development|release>, lock <state>)`.
  Stable codes are 0, 2, 3, 4, 6, and 7.

This is a stable command contract over one local host path, not a declaration that the Windows
target is supported, distributable, ABI-stable, or self-hosted. Persistent cache defaults remain
closed by their separate evidence gate.

## Evidence

- unit tests cover closed flags, default naming, lock/source failure categories, missing entry, and
  contained replace-on-success publication while retaining experimental no-overwrite behavior;
- the configured Windows smoke performs development and release replacement builds, inspects and
  executes both products, and repeats a locked release build through a separately built `fairyc.exe`;
- full Go test, build, vet, and strict scaffold checks cover the shared pipeline and documentation.
