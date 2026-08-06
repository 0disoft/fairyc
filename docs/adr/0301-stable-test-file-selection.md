# ADR 0301: Stable exact test-file selection

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0267, ADR 0268, ADR 0295, ADR 0300

## Context

Stable `fairyc test` can select canonical test identities and can report their portable source
paths, but it cannot select one admitted source file. Editors and focused CI jobs therefore must
first list identities, group them by path, and pass every identity back through `--exact`. That is
needlessly indirect and cannot select a known test file that currently contains zero tests.

The test overlay already owns a contained, portable workspace-relative path for every admitted
test source. Reusing that exact domain avoids host-path normalization, glob syntax, fuzzy matching,
and a second filesystem observation. Selection must not hide syntax or compiler failures elsewhere
in the selected package overlay.

## Decision

1. Advance stable workspace test behavior to v9 with repeatable
   `--file <portable-test-path>` in execution and `--list` modes.
2. Match values byte-for-byte against the portable workspace-relative test-source paths used by
   `--locations`. Root paths look like `tests/example.fy`; member paths include their workspace
   prefix, such as `libs/answer/tests/example.fy`. Matching is case-sensitive and `/` is the only
   separator; no absolute path, backslash normalization, glob, directory, or fuzzy form is admitted.
3. Repeated distinct values form a union of files. A missing value, duplicate value, or path absent
   from the complete selected test overlay is a code-2 invocation error with no report.
4. A known admitted test file containing zero `@test` declarations is a valid selection and
   produces the existing zero-test success report.
5. Keep `--file` mutually exclusive with `--exact`. Both are exact selectors, and defining their
   intersection would add ordering and empty-selection ambiguity without new selection power.
6. Select only after test declaration validation. Syntax, declaration, semantic, Flow, MIR, and
   verifier admission still cover the complete selected package or workspace test overlay; the
   selector limits only listing and execution.
7. Preserve canonical report ordering, engines, trap continuation, locking, exit codes, text bytes,
   and JSON schemas. The selector adds no report field and does not change v1, v2, or v3 bytes.
8. Prove root, member, workspace, listing, zero-test, and rejection behavior in Go tests. Prove low
   and optimized isolated Windows MSVC execution by selecting one file while another admitted file
   contains an excluded failing test.
9. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.36`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, interface, and generated-code versions unchanged.

## Consequences

- Editors and CI can run or list one exact test file without reconstructing function identities.
- Portable path spelling is intentionally strict; host-native paths and case folding fail closed.
- File selection never turns an invalid unselected test source into a successful run.
- Tags, fuzzy names, directory/glob selection, changed-file discovery, durations, caching, and
  parallel scheduling remain separate decisions.

## Evidence

- `internal/cli/workspace_tests.go` owns the admitted file catalog and exact selector.
- `internal/cli/workspace_tests_test.go` owns root, member, workspace, zero-test, and rejection cases.
- `cmd/fairyc-windows-host-smoke/standalone_compiler_proof.go` owns low/optimized isolated native
  execution evidence with an excluded failing file.
