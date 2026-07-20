# ADR 0127: Add an experimental verified-MIR check command

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- MIR source: `docs/mir/mir-v0.md`

## Context

`check-syntax` stops after AST lowering, `check-semantics` stops after typed HIR and Flow, and `run`
continues through verified MIR but then requires an executable `main() -> i32` and
interpreter-supported behavior. A library module or FFI wrapper can therefore be valid through MIR
while `run` rejects execution. Using `emit-c` as a check would add backend capability requirements
and dump a translation unit.

The future release-stable `fairyc check` still needs target, configuration, package, output, and
compatibility decisions. An experimental command can expose the implemented verified-MIR boundary
without pretending those decisions are settled.

## Decision

1. Add `fairyc experimental check <primary.fy> [--source <file.fy>]...`.
2. The primary file and repeated sources form one explicit closure of at most 64 relative regular FY
   files under the current source root. Existing per-file and aggregate limits, duplicate rejection,
   containment, and no-discovery rules apply. The first file is not a required executable entry.
3. The command runs syntax analysis, name resolution, typing, effect checks, every implemented Flow
   group, MIR lowering, and the MIR verifier for the complete explicit closure.
4. It selects no target, invokes no interpreter or backend, resolves no toolchain, executes no
   process, accesses no network, and writes no artifact. Verified foreign calls are therefore
   checkable even though the MIR interpreter cannot execute them.
5. Success writes exactly one ANSI-free line:
   `check ok: <files> file(s) (verified MIR: <functions> function(s), <foreign> foreign declaration(s))`.
   Counts describe provisional compiler IR and may include compiler-private functions. Existing
   source-diagnostic, subset, invocation, internal, and output-failure codes remain unchanged.
6. Stable `fairyc check` remains reserved. This command does not decide package discovery,
   target-aware validation, configuration, JSON, stable counts, or release exit-code compatibility.

## Consequences

- Non-executable libraries, FFI wrappers, and programs beyond interpreter capabilities can be
  validated without generating C.
- The command reuses the source loader, semantic pipeline, MIR lowerer, and verifier used by run and
  build paths; it creates no parallel compiler pipeline.
- `check-semantics` remains useful when typed HIR exists before MIR support.

## Verification

- CLI tests cover a library without `main`, a verified foreign call, a multi-file closure, source
  diagnostics, malformed flags, output failure, and help discovery;
- complete Go test, build, and vet intents cover shared-pipeline regressions;
- the Windows standalone compiler smoke checks the synchronized help surface without WSL;
- no Linux, network, artifact, database, or release action is required.
