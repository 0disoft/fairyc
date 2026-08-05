# ADR 0273: Stabilize JSON test identity listing

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Schema source: `spec/cli/stable-test-list-v1.schema.json`
- Depends on: ADR 0181, ADR 0267, ADR 0268

## Context

ADR 0268 gives people an execution-free way to discover exact test identities, but automation must
still parse a human report and discard its summary. Editors, CI orchestration, and LLM tooling need
the same compiler-admitted identity boundary without depending on prose or inferring lock state.
Adding source locations or metadata would expand discovery semantics before those values have a
stable owner, so the first machine contract should expose only already-stable facts.

## Decision

1. Stabilize `fairyc test --list --json [--locked]`. `--json` requires `--list`; duplicate
   `--json` is an invocation error. Existing listing incompatibilities with `--native`, `--release`,
   and `--exact` remain unchanged.
2. JSON listing retains ADR 0268's complete workspace load, lock handling, root test overlay,
   compiler admission, canonical ordering, execution-free behavior, exit codes, and artifact rules.
3. Success emits one compact newline-terminated `fairyc.test-list/v1` object. Required fields are
   `schemaVersion`, `tests`, `count`, and `lock`. `tests` contains canonical identity strings in
   order, `count` equals its length, and `lock` is `updated`, `unchanged`, or `verified`.
4. The report contains no paths, source text, source locations, test metadata, host inventory,
   toolchain data, or ANSI escapes. The checked-in JSON Schema is the field/type authority.
5. Plain `fairyc test --list [--locked]` keeps its exact existing bytes. Language revision, SDK,
   runtime ABI, manifest, lockfile format, package identity, public interface, and generated code do
   not change.

## Evidence

- CLI tests cover canonical and empty JSON reports, flag-order independence, invalid and duplicate
  `--json`, writer failure, schema/struct field agreement, and preservation of text output.
- The Windows host smoke invokes both text and JSON discovery through a separately built
  `fairyc.exe`, requires exact canonical bytes from a trap-bearing test overlay, and then continues
  through isolated native execution.
- Go format, test, build, and vet plus documentation validation cover implementation and contract.

## Consequences

Automation can consume stable identities and lock state without parsing human text or executing a
test. This deliberately does not add workspace-member recursion, locations, metadata, tags, fuzzy
matching, file selection, caching, or machine-readable execution reports.
