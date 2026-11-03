# Windows Fairy HTTPS package transport, 2026-11-04

## Scope

The private Fairy `project_remote_https` module drives Windows WinHTTP through
`internal/candidatehost/windows_https.c`. URL admission, archive byte budgets and
response acceptance live in Fairy. The OS adapter handles TLS and asynchronous
request/read completion. WinHTTP is loaded only from System32, with system trust,
no proxy discovery, redirects, cookies, automatic credentials or decompression.
The deadline covers send, response headers and all body reads, including a slow
stream. Cancellation retains callback context/read storage through the final
`HANDLE_CLOSING` notification. TLS certificate bypass flags are never set.

The supplied timeout is bounded to 30 seconds; archive bytes are bounded to 16 MiB.
Content-Length and actual streaming bytes are checked separately. Only HTTP 200
and absent/identity content encoding are accepted. Go and authored runtime selectors
and the pinned distribution source pack include HTTPS only when referenced.

## Executed Windows checks

- `FAIRYC_RUN_HTTPS_TRUST_TESTS=1 go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateRemoteHTTPS$'`: MSVC low/optimized pass. Real loopback TLS exercises fixed/chunked/exact-limit responses, redirects, authentication challenges, encoded/oversized/truncated responses, timeout, untrusted TLS and non-HTTPS URL rejection. Redirect targets receive no request; request headers carry no ambient credentials, cookies or compression negotiation.
- The opt-in test installs one freshly generated one-hour loopback certificate in CurrentUser ROOT, then deletes exactly its fingerprint during cleanup. Production transport uses unchanged Windows trust.
- `go test -mod=readonly -count=1 ./internal/candidatehost -run '^TestPrivateWindows'`: pass, including ordinary/HTTPS/combined/unknown-symbol source selection.
- `go vet -mod=readonly ./internal/candidatehost`, runtime asset pin regeneration and `git diff --check`: pass.

## Boundaries

Transport was committed as `461e8e44`. The following slice connects online
command dispatch and immutable archive caching as described below. Linux adapters/native
execution, clean-machine distribution, full `go test ./...` and hosted CI remain
unexecuted. No push, dispatch, public API/SDK ABI, database, dependency version or
runner change. WinHTTP is an OS DLL dependency only for the online Windows path.

Native lifecycle references: [WinHttpCloseHandle](https://learn.microsoft.com/en-us/windows/win32/api/winhttp/nf-winhttp-winhttpclosehandle)
and [WinHTTP option flags](https://learn.microsoft.com/en-us/windows/win32/winhttp/option-flags).

Authored runtime packaging: `TestCandidateHTTPSRuntimePackaging` passes low/optimized, comparing complete Go/Fairy runtime bytes for five process/rooted/HTTPS combinations per profile.

## Online orchestration and immutable cache

`project_remote_online` binds the same manifest/plan, acquires the same exclusive
fetch lock and delegates existing-tree verification/publication to the offline
owner. Only an absent cache blob triggers HTTPS. Corruption, aliases and non-file
cache entries never trigger repair/download. Each request receives the lesser of
30 seconds and the remaining two-minute operation budget. `project_remote_cache`
stores bytes only after SHA-256, complete ZIP and package identity validation.
A synced private file is hard-linked relative to the retained cache directory;
closing removes only its original staging name. Link failures do not fall back
to replacement. An old digest-derived stage is preserved and rejected.

The authored CLI now admits `experimental fetch-packages` without `--offline`.
Download/hit/install/reuse counts and statuses follow Go. Preparation failure
reasons remain candidate-owned. Both the command leaf and actual source entry
explicitly declare the network effect. Stable commands and the isolated offline
API do not download, and fetch never writes `fairy.lock`.

Focused verification on the final engine/command source:

```powershell
$env:FAIRYC_RUN_HTTPS_TRUST_TESTS='1'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidate(FetchOnlineCommand|RemoteHTTPS|HTTPSRuntimePackaging|FetchCommand|RemoteOffline)$'
```

Passed in 103.750 seconds, low/optimized throughout. The online command compares
Go Fetcher results and complete file trees for fixed/chunked/escaped-path success,
missing cache parents, response/auth/redirect/encoding/size/hash/ZIP/identity
failures, corrupt cache, a held lock and cache type conflict. A separate stale
native-stage case proves preservation. A two-package failure preserves the first
complete package/cache. Warm online and offline calls make zero further requests.
Output failure returns 4 while completed cache/package publications survive.
All 26 prior offline restoration cases also pass, including junctions, stage
cleanup and partial progress. The transport probe additionally covers a body
stream that continues delivering chunks across the total request deadline.

`go vet -mod=readonly ./internal/cli ./internal/candidatehost ./internal/candidatebuild`,
CLI metadata `--check`, runtime source-pack pin check and `git diff --check` pass.
The expanded runtime packaging replay also includes the new hard-link symbol.

The full authored CLI low/optimized replay also passed; its independent
acceptance results follow.

Updated source-of-truth owners: `docs/cli/command-contract.md`,
`docs/package/remote-sources-v1.md`, ADR 0498,
`docs/development/source-of-truth-packages-cli-and-sdk.md` and
`docs/development/ordered-work-20261031.md`.

## Full authored CLI acceptance

Implementation source: `6845a16e` (transport predecessor `461e8e44`). Environment:
Windows amd64, Go 1.27.1, MSVC 14.51.36231. No frontend or policy implementation
was changed after this replay started.

```powershell
$env:FAIRYC_RUN_HTTPS_TRUST_TESTS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/online-fetch-cli-20261104'
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestCandidateFetchOnlineCLI$'
```

| Execution | Result | Elapsed |
| --- | --- | --- |
| low, 172 generated C parts | Pass | 121.04 seconds |
| optimized, 172 generated C parts | Pass | 286.24 seconds |
| complete Go test command | Pass | 407.432 seconds |

Each profile uses the complete authored dispatcher and removes its separately
built Go companion before command replay. The fixture performs shared syntax/help
and offline regression checks, the same online success/failure/tree comparisons
as the focused command test, and an actual online installation followed by authored
`check` and `check --locked`. The online output-failure fixture is also the
workspace used for those check calls, proving completed work survives stdout
failure. No candidate command invokes a Go frontend, external download command
or fixture transport substitute. The Go Fetcher is an independent test oracle
with the local server's verified trust configuration.

The source/image digests below come from the harness's exact generated-source
and binary identity receipt. They identify retained local artifacts, not a release
or clean-machine distribution result.

| Profile | Generated source identity | Executable SHA-256 |
| --- | --- | --- |
| low | `b8f6cb9eff88770a4695007abd459195973bb05eebc9ebfef7e6b13da1d4e3aa` | `6c04828331b6de5d83bd59e52c607775a2c07b66bc52d0870181526b30463e5b` |
| optimized | `54fc7efd0ad7ac94ef3e8292cea82e1183735d523d4df178da1699c29bcda8a1` | `ba5e3cd322387096c1a9da2be2858a2911b159f8b2fb3bf5d8fb642d07a819f2` |

The complete output is retained locally under
`.cache/online-fetch-cli-20261104/validation.log`. The expanded six-combination
runtime source comparison and unchanged Go offline CLI regression also pass
(`TestCandidateHTTPSRuntimePackaging` and
`TestRemotePackageCLIOfflineLockAndFailures`, 9.559 seconds total).

All temporary loopback trust certificates were removed. Linux transport/facade
selection and native execution remain on the combined deferred queue. Full
`go test ./...`, clean-machine/distribution acceptance and hosted CI were not run.
The public command grammar is unchanged; its existing online form is now
implemented by Fairy on Windows. No public SDK ABI, database, dependency,
repository hygiene or CI/runner configuration was changed. No push or release.
