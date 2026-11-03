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

## Retained Windows ZIP

Built once from clean main commit `9bc77723bd0a02714173f50883ac348c8caa0a52`:

```text
node tools/starter-smoke.mjs --package-local
node tools/windows-preview/verify.mjs --receipt .cache/starter-smoke-2q2uI0/receipt.json --revision 9bc77723bd0a02714173f50883ac348c8caa0a52
```

Both passed. The packaging receipt records all sixteen steps, including compiler/LSP
build identity, ZIP round trip, LSP version/protocol, starter creation, check/locked
check, semantic tests, native tests and run in both profiles. Build metadata records
Go 1.27.1, Windows AMD64, CGO disabled, `vcs.modified=false`, `-trimpath` and `-s -w`.
The build used the existing Go license and toolchain notice without modification.

| Artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| ZIP | 9,532,934 | `b027d0e133bed35f2432f0e9abd0670ed8a7012f3882d9eb9e74f95b652de0a5` |
| `fairyc.exe` | 17,520,640 | `508a652b41b1b018efe6a63870848cbdd204590336eceb072bc1bd5a72dccd27` |
| `fairyc-lsp.exe` | 8,389,120 | `beb5acb9d618562496c1e60f83368d772eb6e6c284d0849d5e79de49275007eb` |

The exact ZIP, `.zip.sha256`, internal notices/checksums, extracted files and build
receipt remain in `.cache/starter-smoke-2q2uI0/`. The retained-artifact verifier
passed all identities; its output is
`.cache/release-readiness-20261104/preview-recheck.json`.

## Execution outside the checkout

A separate local verification script rechecked the retained receipt before expanding
the exact ZIP under `%TEMP%/fairyc ZIP 검증-PsDAf3/preview`. It created a fresh starter
under the adjacent `projects` directory. No source checkout or compiler cache was
copied. Child PATH contained only `C:\Windows\System32;C:\Windows`; inherited Go,
Visual Studio, compiler override and Fairy-specific environment variables were removed.
Installed MSVC/SDK remained available through normal compiler discovery.

```text
node .cache/release-readiness-20261104/external-smoke.mjs .cache/starter-smoke-2q2uI0/receipt.json 9bc77723bd0a02714173f50883ac348c8caa0a52
```

All thirteen steps passed: extraction, `new`, `check`, locked check, semantic test,
and each profile's explicit `build`, direct generated EXE execution, `run` and native
test. Both native tests report one pass, zero failures; generated EXEs and `run`
return zero with no output. Source-file hashes and `fairy.lock` were preserved after
locked commands; compiler, LSP and ZIP hashes matched the verified candidate.

Raw commands, results and source/artifact identities are retained in
`.cache/release-readiness-20261104/external-zip-receipt.json` and
`external-smoke.log`. The first extraction attempt failed because this one-off
verification script used incorrect PowerShell environment-variable syntax. That
setup error was corrected before any compiler execution; the unsuccessful attempt
is separately retained as `external-zip-setup-failure.json` and
`external-smoke-setup-failure.log`. It is not counted as a successful run or a product
defect. The passing run used a fresh temporary directory.

This is same-host Windows evidence, not a clean-machine or second-PC execution.
The ZIP contains the Go Stage 0 compiler and LSP, not the internal Fairy-written CLI.
No Linux native execution, complete Go suite, new hosted CI result, push or release
was performed. Documentation recorded after packaging does not change the artifact's
source revision; retain and publish these exact bytes only after the separate release
gates are met. Do not rebuild and reuse this receipt.

## Scope

This record changes no product API, schema, dependency, repository hygiene rule,
runner, workflow, source-of-truth contract or release version. Publication and
clean-machine evidence remain separate from local Windows ZIP verification.
