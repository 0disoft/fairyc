# ADR 0267: Stabilize exact test selection

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Depends on: ADR 0181, ADR 0265

## Context

The stable test command discovers and runs every root-package test. That is useful for complete
verification, but the native lane compiles one isolated image per test, so repeatedly investigating
one failure pays the cost of every unrelated test. A substring or glob filter would be convenient
but would freeze matching, escaping, case, and zero-match behavior without evidence that those
semantics are right for Fairy C.

The harness already owns canonical source-visible identities such as
`tests::math::addition::adds_two`. Reusing those identities gives scripts and humans an unambiguous
selection boundary without adding another naming language.

## Decision

1. Extend the spelling to
   `fairyc test [--native [--release]] [--exact <test-identity>]... [--locked]`.
2. `--exact` is repeatable. Each value must equal one discovered canonical
   `<module>::<function>` test identity byte-for-byte. Flag order never changes execution or report
   order; selected tests retain the complete suite's unsigned UTF-8 canonical ordering.
3. Duplicate exact identities are invocation errors. A missing value, unknown identity, or any
   unknown flag is also code 2 with no report. Multiple unknown identities are reported in sorted
   order. An absent `--exact` continues to select every discovered test; an empty suite still
   succeeds only when no unknown selector was requested.
4. Test-tree loading, syntax, `E9001` validation, dependency reconstruction, semantic analysis,
   Flow, MIR lowering, and initial verification still cover the complete overlay. Exact selection
   limits execution, and in native mode therefore limits private harness creation, reachable C
   emission, MSVC compilation, PE/unwind inspection, and process execution. It does not hide a
   broken unselected test source or change production identities.
5. Interpreter/native outcome grammar, trap continuation, lock handling, engine selection,
   profiles, output rejection, exit codes, and no-artifact behavior remain unchanged.

## Compatibility

The flag is additive. Every existing invocation without `--exact` remains byte-for-byte unchanged.
Canonical test identities were already stable report data under ADR 0181; this decision makes them
accepted CLI input without changing language, SDK, runtime ABI, manifest, lock, package identity,
public interface, or generated test code. Substring, prefix, glob, regex, tag, file, line, skip,
ignore, and random-order selection remain outside the contract.

## Evidence

- CLI tests select multiple identities supplied in reverse order, require canonical report order,
  prove an unselected trapping test is not executed, and reject missing, duplicate, and unknown
  selectors.
- Native dependency-injected tests continue to prove trap isolation and output/receipt failures.
- The Windows smoke passes two exact selectors in reverse order to a separately built `fairyc.exe`,
  requires canonical failing-then-passing native results, and runs the checked-in selected test in
  both low and optimized profiles.

## Consequences

Users can rerun one or several known tests without paying unrelated native compilation and process
costs. Exact selection intentionally optimizes attribution and repeatability rather than discovery;
listing, fuzzy matching, tags, workspace-member recursion, parallel scheduling, persistent test
caches, and machine-readable reports remain separate work.
