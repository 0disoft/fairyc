# Windows Fairy offline fetch command, 2026-11-04

This connects the [offline restoration API](windows-candidate-remote-offline-20261103.md)
to `file_cli` through the authored `project_fetch_cli` command module. Host: Windows amd64,
Go 1.27.1, MSVC 14.51.36231. Linux remains batched; no WSL, remote runner or CI dispatch is used.

## Command behavior

`fairyc experimental fetch-packages --offline` validates all syntax before workspace I/O and
restores only hash-pinned cached archives beneath the working directory. No Go fetch frontend,
online transport or subprocess is called. Existing archive/identity/containment/lock/staging
policies remain owned by the previous offline slice.

The success line matches Go exactly, including zero downloads and installed/reused counts.
Invocation/help output matches the shared Go contract. The embedded help had drifted and was
regenerated from its owners with `tools/candidate/fairyc-candidate-cli-embed`; the same tool's
`--check` verifies exact source equality. No target support or dependency version is promoted.

Preparation failures print `package fetch failed:` plus bounded candidate reason text, rather
than reproducing Go/OS error strings. Nonzero cleanup status adds an inspection hint. Exit codes
remain 0 success, 2 invocation, 7 preparation and 4 output failure. Success-output failure may
follow completed publication, and the command preserves that completed package.

Bare fetch without `--offline` is deliberately rejected with code 2 before workspace reads;
the Go CLI's online mode is not claimed for the candidate. Help aliases, unknown experimental
commands and excess/duplicate/invalid flags do not consult the cache or modify the workspace.

## Acceptance

`0e48a217` adds the command leaf and shared metadata refresh. The narrow native test builds
the real authored import closure and passes low/optimized MSVC in 18.363 seconds:

```powershell
go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateFetchCommand$'
```

The leaf compares exact Go invocation/help and cold/warm success output, whole resulting trees,
unchanged graph-lock data, corrupt warm-cache rejection and no online fallback. Read-only NUL
handles exercise failed stdout and stderr writes: output failure returns 4, completed restoration
survives, and failure diagnostics do not mask a stderr write error.

The full CLI replay uses the authored `main` and complete public dispatcher. Each profile emits
172 C parts through the existing builder. The C-only companion is removed before every command
replay, so it cannot provide a hidden Go package fetch or compiler frontend. It repeats the leaf
assertions through the real dispatcher and then restores a fresh workspace and successfully runs
ordinary `check` and `check --locked` on the installed member.

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/fetch-cli-20261104'
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestCandidateFetchCLI$'
```

Existing exact-C/image digest records allow safe local reuse. The integration remains opt-in;
this work does not add a workflow or spend remote runner time. Both profiles passed: low
98.98 seconds, optimized 232.58 seconds, total 331.695 seconds including full compilation.
Each profile runs 22 complete CLI processes (44 total); the separate leaf replay runs 19
processes per profile. The existing per-build allowance and 16 MiB private compiler stack
reserve are unchanged.

Also executed and passed: the existing Go offline CLI regression, scoped CLI vet, Go formatting,
CLI metadata regeneration/check and Git diff checks. There is no need to repeat the unchanged 30/23/26-case
codec/private API matrix from the prior receipt.

## Remaining scope

Online candidate transport, Linux retained-directory adapter/command selection and native
execution, clean-machine packaging and release/full-repository acceptance remain unexecuted.
The public CLI gains this bounded candidate command; public SDK API, private native ABI,
manifest/lock formats, DB, dependency versions, repository hygiene and runner configuration are
unchanged in this slice. The existing cooperative deadlines, bounds, partial-completion and
manual stale-lock/stage recovery limits still apply. No push or deployment is performed.
