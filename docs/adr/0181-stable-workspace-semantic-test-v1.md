# ADR 0181: Stabilize isolated workspace semantic tests v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Loader source: `docs/package/workspace-loader-v1.md`
- Implementation: `internal/workspaceloader/test_sources.go`, `internal/cli/workspace_tests.go`,
  `internal/interpreter/`

## Context

The compiler can already load an exact production workspace, check it through verified MIR, and
execute the admitted semantic subset with the interpreter. A stable test command still needs two
boundaries that ordinary compilation must not guess: which files are tests, and which functions are
automatic test entries.

Putting `@test` functions in the manifest-owned production source root would make test-only code
participate in build, run, emit, public-interface hashing, and cache invalidation. Running one native
image would also stop at the first trap without a stable way to identify the failing test. Stage 0
therefore needs a smaller semantic harness before target-native testing.

## Decision

Stabilize `fairyc test [--locked]` as a root-package semantic test command.

- Ordinary workspace loading remains unchanged and never reads `tests/`. The test command first
  loads the production workspace and maintains or verifies its canonical `fairy.lock`, then overlays
  an optional exact-lowercase root-package `tests/` tree in memory. Test bytes never enter the
  production lock, build, run, emit, package cache, or public interface.
- The test tree must be a real contained directory, may not be a symlink, and may not overlap the
  root production source or an explicit workspace member. It shares the existing workspace file and
  byte budgets. An absent or empty tree is valid.
- A test file path is package-relative with its literal `tests/` prefix: `tests/math/add.fy` must
  declare `module tests::math::add;`. Test modules join only the root package's ephemeral source set,
  may contain private helpers, and may import production modules or manifest-declared dependencies.
  Workspace-member test trees are not discovered by this v1 command.
- `@test` is harness-owned and valid exactly once, without arguments, on a private safe
  zero-argument free function returning `void`. The harness validates that shape as `E9001`, removes
  only the accepted marker from the in-memory AST, and then runs the ordinary package-aware semantic,
  Flow, MIR-lowering, and MIR-verifier pipeline. `@test` in production source remains outside ordinary
  language compilation rather than silently becoming build metadata.
- Test identities are source-visible `<module>::<function>` strings sorted by unsigned UTF-8 bytes.
  Each selected function is rechecked as a zero-argument void MIR entry and runs in a fresh
  deterministic interpreter runtime. Mutable runtime state, step accounting, drop traces, and region
  traces therefore do not leak between tests.
- A language trap fails that test but does not stop later tests. Interpreter capability or resource
  limit rejection aborts the command as an unsupported semantic-test subset instead of pretending
  the test failed. Engine invariants remain internal failures.
- The command invokes no C backend, native compiler, linker, SDK, WSL, subprocess, network, or host
  I/O adapter and creates no source, object, executable, cache, or build artifact. Target-native,
  program-I/O, benchmark, fuzz, fixture-discovery, filtering, ignored, setup/teardown, and parallel
  tests remain separate future contracts.
- Each completed test line is exactly `test <identity> ... ok` or
  `test <identity> ... FAILED (<trap-category>)`. The final line is exactly
  `test result: <ok|FAILED>. <passed> passed; <failed> failed; lock
  <updated|unchanged|verified>`. Zero tests succeed with only the final line.
- Stable codes are 0 for all tests passing, 2 for invocation or unsupported semantic-test
  capability, 3 for source or test-declaration diagnostics, 4 for internal or output failure, 5 for
  one or more trapped tests, and 7 for manifest, workspace, graph, test-source, or lock failure.
  Compilation/capability failures produce no report. An external writer may accept a report prefix
  before returning a code-4 error, and ordinary lock maintenance may already have completed;
  `--locked` is the no-write form.

Adding target-native execution, member recursion, filters, test arguments, richer outcomes, or a
machine-readable protocol requires a separate compatibility decision. Adding another trap category
does not change the report grammar but follows the independently versioned language trap contract.

## Evidence

- Loader tests prove production isolation, optional/empty trees, exact case, overlap rejection,
  path-to-module identity, dependency lookup reconstruction, digest separation, and syntax failures.
- CLI tests prove canonical ordering, successful and failing reports, continuation after a trap,
  all `E9001` declaration shapes, zero-test success, ordinary-check isolation, canonical lock
  behavior, output failure, no build artifact, and stable invocation rejection.
- Interpreter tests prove zero-argument void entry admission, trap propagation, wrong-result
  rejection, and boolean comparison values retaining their semantic `bool` type.
- The Windows host smoke builds a standalone Go `fairyc.exe`, runs the checked-in isolated FY test
  under `--locked`, and proves the previously retained native build artifact is byte-identical.
