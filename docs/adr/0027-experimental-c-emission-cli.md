# ADR 0027: Experimental C emission CLI

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- CLI source: `docs/cli/command-contract.md`
- Backend source: `docs/adr/0025-verified-scalar-c-emission.md`
- Implementation source: `internal/cli/` and `internal/cbackend/`

## Context

The C emitter and native fixture runner were reachable only from Go packages and an internal CI
command. A developer could interpret a `.fy` program, but could not ask the canonical `fairyc`
prototype to show the generated C. Adding file output, compiler discovery, or linking at the same
time would mix an inspection command with artifact ownership and target-toolchain policy that remain
unfinished.

## Decision

1. `fairyc experimental emit-c <file.fy>` is the fifth unstable developer command.
2. It accepts exactly one source-root-relative regular `.fy` file under the same bounded loading,
   syntax, semantic, Flow, MIR-lowering, verifier, and `main() -> i32` gates as
   `experimental run`.
3. The interpreter and C emitter share one single-file verified-MIR lowering path. Neither command
   reconstructs frontend or entry semantics independently.
4. On success, stdout contains exactly one deterministic ISO C11 translation unit for the
   provisional scalar `x86_64-linux-gnu` profile. There is no success banner, JSON envelope, ANSI
   escape, source text, absolute path, or stable Fairy C symbol name.
5. Source diagnostics and invocation or capability failures remain on stderr with the existing
   provisional experimental exit codes. A partial C translation unit is never reported as success.
6. The command writes no file, cache, object, executable, or manifest and invokes no C compiler,
   runtime, linker, host subprocess, or network operation.
7. The spelling does not settle the planned stable `fairyc emit <form>` syntax, target selection,
   build artifact policy, or target support status.

## Consequences

- A developer can now inspect or redirect the real C produced from a supported scalar `.fy` program
  while remaining clear that Go still runs the compiler.
- Resource, tagged-value, Region, borrow, unsafe, FFI, runtime, and standard-library programs fail
  closed at the C-backend capability boundary.
- ADR 0028 now lets the separate experimental build command consume the same verified-MIR and C unit
  while owning toolchain resolution, output paths, receipts, cleanup, and process status outside
  this inspection-only command.

## Rejected alternatives

- Print metadata before the C source: breaks direct piping and makes stdout cease to be the requested
  artifact.
- Write a default `.c` file beside the source: introduces collision, cleanup, and source-tree
  mutation policy without an explicit output contract.
- Compile as a side effect: collapses inspection, toolchain availability, linking, and execution
  failures into one command.
- Expose complete Fairy C names in C identifiers: leaks source spelling into an unstabilized ABI and
  creates an identifier-injection surface.
