# Package fetch HTTP session — 2026-11-14

One performance refactor: retain the default HTTPS connection pool for one
`Fetcher.Fetch` operation instead of constructing and closing one per package.
Previously the default CLI path recreated `http.Transport` in every
`downloadWithin`; supplied caller clients already shared their transport.

`Fetch` now owns one default client in its value-receiver copy for online work,
and closes idle connections on every return. `downloadWithin` copies request
policy from that shared client while reusing its transport. Standalone download
calls without a supplied client retain their private-client cleanup. Caller
clients are never closed/mutated. Offline fetch allocates no default pool.

## Reproduction and measured effect

A real local HTTPS server serves 32 distinct, hash-pinned valid package ZIPs
to an isolated default-client fetch process. HTTP/1.1 keep-alive is enabled.
The test checks the server's actual TCP connection states and GET count.

```text
go test -mod=readonly ./internal/packagesource \
  -run '^TestFetchDefaultHTTPSConnectionReuse$' -count=1 -v
before: packages=32 requests=32 TCP/TLS connections=32 (expected-one assertion fails)
after:  packages=32 requests=32 TCP/TLS connections=1  (passes)
```

This removes 31 of 32 connection/handshake setups (96.875%). For N sequential
successful downloads from one keep-alive origin, setup count changes from N to
one; GET count/body bytes are unchanged. Servers that close connections, failed
bodies, and separate origins can require further connections. No fixed latency
or whole-operation speedup is inferred from loopback test duration.

The same default-client path is replayed with a freshly built Go CLI via
`FAIRYC_HTTP_SESSION_CLI`: 32 GETs use one TLS connection. Its subsequent
`fetch-packages --offline` invocation adds zero requests/connections. Test-only
CA trust is supplied solely to isolated subprocesses; system trust stores and
production TLS verification are not changed.

## Validation

- Full `internal/packagesource`: pass, 10.450 s. Includes existing TLS refusal,
  redirect/encoding/size bounds, retry/cancellation, archive and cache tests.
- Explicit caller-owned transport: fetch does not call CloseIdleConnections;
  the caller can still close it. The default Fetcher value is not mutated.
- Fresh CLI build plus actual online/offline CLI replay: pass, 2.399 s package
  run, 32 GETs / one TLS connection / no offline network I/O.
- `gofmt` and `git diff --check`: pass.
- Logs: `.cache/http-session-before.log`, `.cache/http-session-after.log`,
  `.cache/http-session-full.log`, `.cache/http-session-cli.log`.

The 30-second download, two-minute operation, redirect refusal, TLS verification,
body bounds and retry policy are unchanged. No public API/schema, dependency,
database, hygiene or CI/runner changes. Owning remote-source v1 contract unchanged.
No desktop/input tests; measurements were made on Windows/amd64, not Linux.
