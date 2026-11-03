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

This first slice supplies transport, not online command dispatch or package caching.
Those are connected in the subsequent implementation slice. Linux adapters/native
execution, clean-machine distribution, full `go test ./...` and hosted CI remain
unexecuted. No push, dispatch, public API/SDK ABI, database, dependency version or
runner change. WinHTTP is an OS DLL dependency only for the online Windows path.

Native lifecycle references: [WinHttpCloseHandle](https://learn.microsoft.com/en-us/windows/win32/api/winhttp/nf-winhttp-winhttpclosehandle)
and [WinHTTP option flags](https://learn.microsoft.com/en-us/windows/win32/winhttp/option-flags).

Authored runtime packaging: `TestCandidateHTTPSRuntimePackaging` passes low/optimized, comparing complete Go/Fairy runtime bytes for five process/rooted/HTTPS combinations per profile.
