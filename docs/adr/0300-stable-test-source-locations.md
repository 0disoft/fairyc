# ADR 0300: Stable exact-current test source locations

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Schema sources: `spec/cli/stable-test-list-v3.schema.json`, `spec/cli/stable-test-report-v3.schema.json`
- Depends on: ADR 0184, ADR 0268, ADR 0287, ADR 0295

## Context

Stable `fairyc test` exposes canonical identities for discovery, exact selection, semantic results,
and isolated native results. The v1/v2 JSON contracts intentionally omit paths and source metadata.
That keeps them small, but an editor, CI annotator, or LLM tool must rediscover the declaration and
can silently attach an old result to changed source.

The validated test overlay already retains canonical portable paths, exact bytes, function name
spans, package UIDs, and the source-file and line-map identity algorithms from ADR 0184. Reopening a
filesystem path after compilation would weaken that authority and introduce a race. Adding fields to
v1/v2 would break their closed `additionalProperties: false` schemas and byte contracts.

## Decision

1. Advance stable workspace test behavior to v8 by adding repeatable-safe Boolean `--locations` to
   both execution and `--list`. Duplicate flags remain code-2 invocation errors.
2. Preserve discovery, canonical identities, `--exact`, ordering, compiler admission, engines,
   trap continuation, totals, locking, exit codes, and artifacts. Without `--locations`, every v1/v2
   JSON byte and human text byte remains unchanged.
3. In text mode, insert
   ` at <portable-workspace-relative-path>:<one-based-line>:<one-based-byte-column>` after each test
   identity. The position identifies the function name, not the `@test` attribute or body.
4. In JSON mode, use `fairyc.test-list/v3` or `fairyc.test-report/v3` for root, selected-package,
   and workspace requests. Located lists retain the canonical `tests` string array and add an equally
   ordered `locations` array. Located execution outcomes add one required `source` record.
5. Each source record contains manifest `packageUid`, portable workspace-relative `path`, 1-based
   `line` and `byteColumn`, half-open `startByte` and `endByte`, and exact `fileIdentity` plus
   `lineMapIdentity` using ADR 0184's `fyfile1` and `fylm1` encodings.
6. Derive the complete record from the already-contained validated test bytes and AST name span.
   Do not reopen a path, emit an absolute host root, serialize source text, normalize newlines,
   consult timestamps, or infer a location from generated C/native artifacts.
7. A retained location is exact-current only when a consumer verifies both identities against the
   current authorized file. The command does not grant filesystem authority or claim that a path has
   not changed after report emission.
8. Prove root JSON list/execution and text listing in Go tests, schema-to-struct closure for both v3
   documents, and low plus optimized isolated Windows MSVC execution with exact source locations and
   unchanged trap-continuation outcomes.
9. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.36`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, and generated-code versions unchanged.

## Consequences

- Editors and automation can navigate directly to an admitted test and reject stale source anchors
  without parsing human diagnostics or trusting host-local absolute paths.
- Existing stable consumers receive no new fields or bytes unless they explicitly request v3.
- Located reports pay one bounded source-file and line-map identity scan per admitted test file.
  Test source budgets already bound that cost; no cache or persistent index is introduced.
- The report intentionally omits source text, enclosing declaration/body spans, documentation,
  timestamps, durations, program streams, generated artifact identities, and host inventory.
- Native evidence remains Windows x64 MSVC evidence, not a portable native-target claim.

## Evidence

- `internal/cli/workspace_tests.go` owns discovery, source-anchor derivation, and text/JSON output.
- `internal/packagecontent/file_identity.go` owns exact file and line-map identity algorithms.
- `spec/cli/stable-test-list-v3.schema.json` and `stable-test-report-v3.schema.json` own machine shape.
- `internal/cli/workspace_tests_test.go` and `schema_test.go` own exact unit and schema closure.
- `cmd/fairyc-windows-host-smoke/standalone_compiler_proof.go` owns low/optimized isolated native
  execution evidence.
