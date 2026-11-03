# Windows preview preparation, 2026-11-04

## Frontend failure triage

Inspected [Frontend run 41190196880](https://github.com/0disoft/fairyc/actions/runs/41190196880)
on remote main `eb274643bfcfe746617121ddc6f130b018978b53`. Its failures have two causes:

- PowerShell renders revision-mismatch diagnostics with continuation gutters. The old
  comparison left a gutter between `differs` and `from requested candidate`.
  Existing local fix `817d43dd58f2c49999492525380585cd46daad24` strips line-leading
  gutters before normalizing whitespace and retains a regression using the wrapped error.
- Runtime support source begins with the embed-only `//go:build ignore` comment.
  Three source-order cases expected the POSIX feature macro at byte zero.
  Existing local fix `66fae50dfb075774e5d2d9e550d564907396460b` excludes that exact
  comment before checking the first C directive, preserving the macro-order assertion.

Both fixes were already present in clean local source `04e01e5d`. No duplicate code
change or workflow relaxation was needed. Rechecked on Windows AMD64:

```text
python -B -m unittest discover -s .github/scripts -p 'test_*.py'
  PASS: 25 tests, no skips; includes actual PowerShell workflow blocks.

go test -mod=readonly -count=1 -timeout 3m ./internal/cstdlib ./internal/cli -run '^(TestSourceForSymbolsSelectsMinimalDependencyClosure|TestProductSupportSourceLinksOnlyCalledStandardLibraryClosure)$'
  PASS: both packages, including Linux-target source assertions on Windows.

node --test tools/windows-preview/verify.test.mjs
  PASS: 21 tests, no skips.
```

Logs are retained locally under `.cache/release-readiness-20261104/`. The source
assertions above do not execute Linux programs. Native Linux remains in the existing
batched queue. This check does not establish a passing hosted run for local HEAD;
the fixes and subsequent work have not been pushed. No hosted workflow was dispatched.

## Scope

This record changes no product API, schema, dependency, repository hygiene rule,
runner, workflow, source-of-truth contract or release version. Publication and
clean-machine evidence remain separate from local Windows ZIP verification.
