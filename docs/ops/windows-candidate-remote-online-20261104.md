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

The full authored CLI low/optimized replay is recorded separately below once
finished; focused module results alone do not establish full CLI acceptance.

Updated source-of-truth owners: `docs/cli/command-contract.md`,
`docs/package/remote-sources-v1.md`, ADR 0498,
`docs/development/source-of-truth-packages-cli-and-sdk.md` and
`docs/development/ordered-work-20261031.md`.
