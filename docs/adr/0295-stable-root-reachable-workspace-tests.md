# ADR 0295: Stabilize root-reachable workspace test aggregation

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Loader source: `docs/package/workspace-loader-v1.md`
- Schema sources: `spec/cli/stable-test-list-v2.schema.json`, `spec/cli/stable-test-report-v2.schema.json`
- Depends on: ADR 0181, ADR 0265, ADR 0267, ADR 0268, ADR 0273, ADR 0287, ADR 0289

## Context

Singular `--package` lets a member own tests but requires one compiler invocation per package. A
workspace verification loop needs one opt-in command that admits and executes the complete
production graph's local tests. Reusing unqualified `<module>::<function>` identities would become
ambiguous when packages use the same test module and function names, while silently adding member
tests to existing root-only invocations would break stable output and native cost expectations.

## Decision

1. Add `--workspace` to stable test execution and listing. It is mutually exclusive with
   `--package`; omission preserves every root-only and singular-package behavior and output byte.
2. Select every local package that contributes exactly one resolved instance to the ordinary
   root-reachable production graph. Compiler-bundled SDK packages and catalog-only unreachable
   members are excluded. A feature-instance-ambiguous selected package rejects before a report.
3. Overlay each selected package's exact contained `tests/` tree in one ephemeral workspace. All
   overlays share the existing aggregate source file and byte budgets, never enter the lock, and do
   not mutate the production workspace.
4. Workspace identities are `<package-uid>::<module>::<function>`. Package UID is an existing exact
   manifest identity; the complete string is sorted in unsigned UTF-8 order and is accepted
   byte-for-byte by repeatable `--exact`. Default and singular-package identities remain
   `<module>::<function>`.
5. Text listing and execution retain their existing grammar with the qualified identity substituted.
   Workspace JSON listing uses `fairyc.test-list/v2`; workspace JSON execution uses
   `fairyc.test-report/v2`. Both retain the v1 field sets and deterministic field order while
   changing the identity domain. Non-workspace JSON remains exactly v1.
6. Listing compiler-admits the complete aggregate without execution. Semantic mode runs every
   selected entry in a fresh interpreter state. Native mode retains one inspected temporary Windows
   MSVC image and process per test, trap continuation, development/release profiles, no retained
   artifacts, output rejection, and existing exit-code meanings.
7. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.32`, runtime ABI `fairyc-runtime/0`,
   manifest, lockfile, package identity, public interface, MIR, target, and generated-code versions
   unchanged.

## Compatibility

The option and v2 schemas are additive. Commands without `--workspace`, including `--package`,
retain exact discovery, identity, report, lock, artifact, and process behavior. Automation opts into
the new qualified identity domain explicitly.

## Evidence

- Loader tests prove root and reachable-member overlays, UID ordering, aggregate budget reuse, and
  exclusion of an unreachable catalog member.
- CLI tests prove v1 preservation, v2 semantic and list JSON, package-qualified ordering, exact
  selection, missing unqualified selection, and option incompatibility.
- Schema guards match both v2 report structures to their checked-in JSON schemas.
- The Windows host smoke invokes a separately built `fairyc.exe` for semantic and list v2 JSON and
  low/optimized native aggregation across the checked-in root and member tests.

## Consequences

One command can now validate the complete root-reachable local production graph without identity
collisions. Catalog-unreachable member aggregation, multiple feature instances, parallel native
scheduling, caching, retries, durations, output attachments, and portable native targets remain
separate contracts.
