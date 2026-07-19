# ADR 0070: Provisional target maturity list

- Status: Accepted
- Date: 2026-07-19

## Context

`internal/target` already owns validated, host-independent TargetSpec values and canonical target
ordering. The experimental `target-spec` command exposes one exact JSON document, but a developer
should not have to know a target ID or decode the complete layout contract just to learn which
profiles exist and why none is currently supported.

The planned CLI reserves `fairyc targets` to report known profiles, maturity, evidence, and
unavailable requirements. Target selection, filtering, stable JSON, local toolchain inventory, and
the formal Windows target remain undecided or explicitly deferred.

## Decision

1. `fairyc targets` is a provisional, no-argument human inspection command. `-h` and `--help` show
   global help; every other argument is an invocation failure with prototype code `2`.
2. The command consumes only isolated values from `target.All()` and validates each value before
   rendering. For every target in canonical ID order it writes the target ID, TargetSpec schema,
   admitted emission subset, independent `emittable`, `conformant`, and `supported` yes/no states,
   and every sorted missing-evidence marker.
3. Output is deterministic, ANSI-free, and source-independent. The command performs no source or
   configuration read, directory scan, executable lookup, SDK or sysroot probe, process execution,
   network request, or write. It does not use WSL.
4. The target list reports declared target contracts, not host capabilities. The target-neutral
   Windows MSVC smoke therefore does not add `x86_64-windows-msvc`, and local compiler versions or
   paths never appear.
5. `fairyc.target-spec/v0` remains the machine-readable source for the current target details. This
   human list has no JSON mode, filter, target override, status promotion, or compatibility promise.
   Phase 3 still owns release stabilization.

## Consequences

- Developers can see immediately that `x86_64-linux-gnu` is known while `emittable`, `conformant`,
  and `supported` all remain false and can see the exact evidence still missing.
- Windows host-smoke success cannot accidentally masquerade as a declared Windows target.
- Adding or promoting a target still requires updating and validating its TargetSpec evidence;
  the CLI does not maintain a second target registry.
