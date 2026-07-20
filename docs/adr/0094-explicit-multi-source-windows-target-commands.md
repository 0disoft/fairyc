# ADR 0094: Explicit multi-source Windows target commands

- Status: Accepted
- Date: 2026-07-20
- Commands: `experimental build-target`, `run-target`, and `exec-target`

## Context

The Windows target commands accepted one FY file even though semantic analysis, imports, HIR, MIR,
the interpreter, and constrained C emission already understand explicit same-package module sets.
That forced a checked safe output helper to remain beside `main`. Public package manifests,
lockfiles, dependency solving, and standard-library package installation are not ready, so automatic
source discovery would pretend that Phase 3 already exists.

## Decision

1. The three Windows target commands accept one positional entry file plus zero or more repeated
   `--source <file.fy>` values. The first file selects the module whose `main() -> i32` is the entry;
   every listed file participates in one semantic and lowering closure.
2. The complete list is explicit. The driver performs no directory walk, wildcard expansion,
   manifest lookup, implicit sibling discovery, transitive source discovery, or ambient package
   search.
3. At most 64 files are accepted. Each file keeps the ordinary per-file source limit, and the set
   keeps the existing four-file-equivalent aggregate byte limit. The driver rejects an empty set,
   duplicate normalized paths, path or symlink escape, malformed source, module/path mismatch,
   unresolved or ambiguous imports, unsupported semantics, and a missing or mistyped entry.
4. Every path is relative to the current source root. Multi-source compilation enforces the
   normative source-root-relative file-to-module mapping for every listed file.
5. `--source` is repeatable; `--target`, `--profile`, and `--output` retain their existing uniqueness
   and required-value rules. The target remains exactly `x86_64-windows-msvc`.
6. The source set changes only frontend closure. It does not admit companion C, assembly, object,
   archive, library, linker option, compiler override, SDK path, environment-selected target, or
   arbitrary standard-library component.
7. `examples/windows_output/main.fy` now imports a separately listed
   `windows_console.fy`. The entry module contains only safe calls; the support module owns the
   narrow unsafe Windows declaration and checked wrapper. The configured Windows smoke requires the
   same two-file command both in-process and through a temporary standalone Go-built `fairyc.exe`.
8. No WSL or Linux compatibility layer participates. This is a bounded experimental source closure,
   not a public package build, manifest format, module search path, installation contract, or
   self-hosting step.

## Consequences

- Users can split FY programs into explicit modules on the real Windows native path without copying
  helpers into `main`.
- Missing source files fail visibly instead of being guessed from imports.
- Phase 3 can later replace the repeated list with a manifest-owned package closure without having
  to preserve implicit discovery behavior introduced here.

## References

- `docs/adr/0009-stage-0-grammar-and-parser-contract.md`
- `docs/adr/0014-resolved-package-graph-fixture-contract.md`
- `docs/adr/0082-windows-msvc-target-c-emission.md`
- `docs/adr/0088-experimental-windows-target-run-command.md`
- `docs/adr/0091-windows-stdout-exec-target.md`
- `docs/adr/0093-borrowed-view-internal-call-transport.md`
- `docs/cli/command-contract.md`
