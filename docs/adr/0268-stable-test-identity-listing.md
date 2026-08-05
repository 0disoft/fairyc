# ADR 0268: Stabilize test identity listing

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Depends on: ADR 0181, ADR 0267

## Context

ADR 0267 accepts canonical test identities as repeatable exact selectors, but a user who has not
already run the suite or inspected its source has no stable command that discovers those values.
Guessing identities or adding fuzzy matching would weaken exact selection and introduce a second
matching language. Running every test solely to discover its name is especially wasteful when a
native test traps, emits output, or needs an unavailable toolchain.

The test harness already discovers, validates, and sorts canonical identities before choosing an
execution engine. Exposing that existing boundary gives exact selection a complete user workflow
without treating source search or a prior failure report as CLI input discovery.

## Decision

1. Stabilize `fairyc test --list [--locked]` as a separate discovery mode. It does not combine with
   `--native`, `--release`, or `--exact`; duplicate `--list` and unknown flags remain invocation
   errors.
2. Listing retains the ordinary production workspace load and lock behavior, root-only `tests/`
   overlay, syntax and `E9001` validation, dependency reconstruction, semantic analysis, Flow, MIR
   lowering, and verifier admission. It therefore lists tests from one complete compilable overlay
   rather than hiding a broken source behind discovery output.
3. Listing never invokes the interpreter, C backend, MSVC, PE inspector, or a test process. A test
   that would trap or emit native output is still listed successfully after compiler admission.
4. Each discovered identity is emitted in existing unsigned UTF-8 canonical order as
   `test <module>::<function>`. The final line is
   `test list: <count> discovered; lock <updated|unchanged|verified>`. An empty suite emits only the
   final line. Output is buffered and a writer failure uses code 4.
5. Success uses code 0; invocation and compiler-capability rejection use 2; source and test-shape
   diagnostics use 3; internal and output failures use 4; workspace, test-source, graph, and lock
   failures use 7. Listing cannot produce a language-trap code 5 or native-build code 6 because it
   executes no test or target toolchain.

## Compatibility

This is an additive stable CLI mode. Existing semantic and native invocations, exact selection,
execution reports, exit codes, and artifacts remain unchanged. The listed values reuse ADR 0181's
existing canonical report identity and ADR 0267's exact-selector input without creating another
identity format. Language revision, SDK, runtime ABI, manifest, lockfile, package identity, public
interface, and generated test code remain unchanged. Machine-readable listing, workspace-member
recursion, and fuzzy, tag, or file discovery remain separate contracts.

## Evidence

- CLI tests require canonical listing order, prove that a trapping test is not executed, cover an
  empty suite, reject duplicate or incompatible options, surface a broken test overlay before any
  list report, and classify writer failure.
- The Windows host smoke invokes `test --list --locked` through a separately built `fairyc.exe`,
  requires both canonical identities from a trap-bearing starter test file plus the exact summary,
  and then continues to exercise the same tests through the isolated native path.
- Go format, test, build, and vet plus documentation validation cover the implementation and
  synchronized stable contract.

## Consequences

Users can now discover a canonical identity and feed it directly to `--exact` without executing the
suite or learning a matching syntax. Listing deliberately proves compiler admission, not runtime or
target capability. Structured machine output, test metadata, source locations, workspace-wide
listing, fuzzy search, tags, and caching remain later decisions.
