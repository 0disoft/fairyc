# ADR 0289: Stabilize package-selected workspace tests

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Loader source: `docs/package/workspace-loader-v1.md`
- Depends on: ADR 0181, ADR 0265, ADR 0267, ADR 0268, ADR 0273, ADR 0287

## Context

The stable test command can compile and execute only the root package's isolated `tests/` tree.
Local workspace members are already exact manifest-owned packages in the production graph, but a
member author must currently move tests into the root package or use an external harness. Automatic
recursion would change existing all-test output, make duplicate `<module>::<function>` identities
ambiguous, and multiply native compilation without an explicit user choice.

## Decision

1. Add optional `--package <package-uid>` to stable test execution and listing. Omitting it preserves
   the root package and every existing output byte. The value must equal one exact local manifest
   package UID already loaded from the current workspace.
2. The selected package must have exactly one root-reachable resolved instance. An unknown UID, an
   unreachable catalog member, or multiple feature-distinct instances is code 2 with no report.
   Compiler-bundled SDK packages cannot be selected.
3. After the ordinary complete production workspace is loaded and its lock is maintained or
   verified, only the selected package's exact-lowercase contained `tests/` tree is overlaid in
   memory. The overlay is disjoint from that package's production source root, shares aggregate
   workspace source budgets, and never enters the lock, build, run, emit, cache, or public interface.
4. Test paths, `@test` validation, complete-overlay compiler admission, canonical
   `<module>::<function>` identities, repeatable `--exact`, text and JSON reports, interpreter
   isolation, native low/optimized MSVC isolation, trap continuation, exit codes, and no retained
   artifact behavior remain unchanged. Because one invocation admits one package, identities do not
   need a second package-qualified spelling and `fairyc.test-list/v1` and
   `fairyc.test-report/v1` remain sufficient.
5. `--package` is valid with execution or `--list`. It is singular; a missing value or duplicate flag
   is an invocation error. Running every member recursively and selecting more than one package in
   one invocation remain separate future contracts.
6. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.27`, runtime ABI
   `fairyc-runtime/0`, manifest, lockfile, package identity, MIR, and generated native code versions
   unchanged.

## Compatibility

This is an additive CLI option. Every invocation without `--package` retains root-only discovery,
selection, report bytes, statuses, lock behavior, and artifact behavior. Scripts opt into member
tests with a manifest UID that is already a stable workspace identity.

## Evidence

- Loader tests prove that only the selected member receives test sources while root sources remain
  production-only and package identity remains resolvable.
- CLI tests prove unchanged default root behavior, member semantic execution, listing, JSON v1
  reuse, lock verification, and missing, duplicate, unknown, and unreachable package rejection.
- The Windows host smoke runs a checked-in member test through a separately built `fairyc.exe` in
  semantic JSON mode and isolated low and optimized MSVC native modes.

## Consequences

Workspace packages can own tests beside their own source and dependencies without changing the
default root suite. A workspace-wide aggregate scheduler, package-qualified aggregate identities,
parallelism, caching, and feature-instance selection remain deliberate later decisions.
