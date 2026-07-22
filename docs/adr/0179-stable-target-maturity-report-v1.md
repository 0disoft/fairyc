# ADR 0179: Stabilize target maturity report v1

- Status: Accepted
- Date: 2026-07-22
- Target source: `internal/target/spec.go`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Prototype prerequisite: ADR 0070
- Implementation: `internal/cli/target.go`

## Context

`fairyc targets` already renders every validated target description without reading a workspace or
probing the host. Its value is the separation between a declared target profile and the machine on
which the compiler happens to run, and between the independent `emittable`, `conformant`, and
`supported` maturity states.

Freezing the current target rows would be wrong. New targets may be added, missing evidence must be
removed when it is supplied, and maturity may advance. The stable boundary therefore needs to own
the report grammar and failure behavior while leaving TargetSpec data and evidence under their own
versioned contracts.

## Decision

Stabilize `fairyc targets` as a source-independent human report v1.

- The command accepts no arguments. `-h` and `--help` use the shared top-level help surface.
- Success renders every validated TargetSpec in canonical target-ID order. Each block contains the
  target ID, schema identity, emission subset, `emittable`, `conformant`, and `supported` as
  `yes` or `no`, then `missing evidence:` followed by zero or more sorted `    - <marker>` lines.
- Blocks are separated by one blank line. The complete report is ANSI-free and ends with a newline.
- Rendering is atomic: invalid or empty compiler-owned target data fails before stdout is written.
- The command reads no source, manifest, lock, configuration, current directory, environment,
  target, toolchain, SDK, sysroot, WSL, process, or network state and writes no file.
- Stable codes are 0 for a successful report, 2 for an invalid invocation, and 4 for an internal
  target invariant or output failure. Failure emits no partial stdout.
- Additive target rows, evidence removal, and maturity changes are TargetSpec data evolution rather
  than changes to the CLI grammar. Removing or reidentifying a declared target still requires the
  applicable target-compatibility decision.
- JSON, filtering, target selection, and local toolchain availability remain outside v1. The
  experimental single-target JSON command is not promoted by this decision.

The report describes compiler-owned declarations, not whether this host can build or run a target.

## Evidence

- CLI tests freeze the current complete two-target report byte for byte, including canonical order,
  labels, indentation, blank-line separation, sorted missing evidence, and the final newline;
- CLI tests prove repeatability with a missing source root, argument rejection, shared help, output
  failure, and absence of host-local paths;
- target tests prove structural validation, immutable lookup copies, closed profile identities,
  sorted sets, and fail-closed maturity ordering;
- full Go test, build, vet, and strict scaffold checks cover the shared command boundary.
