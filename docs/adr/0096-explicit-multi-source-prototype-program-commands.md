# ADR 0096: Explicit multi-source prototype program commands

- Status: Accepted
- Date: 2026-07-20
- Commands: `experimental emit-c`, `build`, `build-host`, and `run-host`

## Context

The interpreter and declared Windows target commands accept one entry FY file plus repeated
`--source` files, but four older program commands still accepted exactly one file. That made the
same verified program invalid merely because a developer moved from interpretation to C inspection,
Linux product build, or target-neutral Windows host smoke. Package manifests and source discovery
remain unfinished.

## Decision

1. `emit-c`, `build`, `build-host`, and `run-host` accept one positional entry file plus zero or more
   repeated `--source <file.fy>` values.
2. All four commands reuse the shared explicit program loader and verified-MIR lowering closure:
   at most 64 files, ordinary per-file limits, the existing aggregate byte limit, normalized
   duplicate rejection, source-root and symlink containment, source-relative module identity, and
   no discovery.
3. The first file alone selects `main() -> i32`. Additional files participate in semantic analysis,
   HIR, MIR, verification, and the one generated C translation unit.
4. `--source` means only Fairy C source. It does not admit C or assembly sources, objects, archives,
   libraries, linker flags, compiler paths, SDK paths, package manifests, or ambient module search.
5. Existing target, toolchain, profile, output, overwrite, runtime, foreign-symbol, and
   target-neutral feature gates remain unchanged.
6. The checked-in two-module arithmetic example now crosses deterministic C emission and real local
   MSVC `build-host` and `run-host` execution. No WSL or Linux compatibility layer participates.

## Consequences

- One explicit module closure can move from fast interpreter execution through C inspection and
  native Windows host smoke without merging files.
- Linux `build` has the same source contract even though this Windows machine cannot execute its
  Clang/GCC host path.
- This does not define stable commands, automatic import discovery, package builds, caching,
  distribution, target support, or self-hosting.

## Superseded boundaries

This decision widens only the single-source input clauses in ADR 0027, ADR 0028, and ADR 0068. Their
remaining output, toolchain, target, and failure contracts stay in force.

## References

- `docs/adr/0027-experimental-c-emission-cli.md`
- `docs/adr/0028-experimental-linux-scalar-build-cli.md`
- `docs/adr/0068-windows-msvc-target-neutral-host-smoke.md`
- `docs/adr/0094-explicit-multi-source-windows-target-commands.md`
- `docs/adr/0095-explicit-multi-source-interpreter-run.md`
- `docs/cli/command-contract.md`
