# Private Linux HTTPS facade v1

This is the Fairy response/lifetime policy for the pending Linux source-fetch
transport, within ADR 0498 and `package/remote-sources-v1.md`. It does not admit
Linux fetch commands or establish a native TLS provider or executed Linux I/O.

## Ownership and private ABI

`linux_remote_https::download(address, maximum, timeout)` returns an owned binary
buffer or `{code, native_code, cleanup_code}`. Classifications retain the Windows
facade's meanings: input/budget 1, local allocation 2, native/ABI/timeout 3,
status/encoding/length mismatch 4 and byte limit 5. Native error values are opaque
nonzero u32 values, preserved separately from a consuming-close failure. No
native diagnostic string or credentials are returned or logged.

The native implementation must implement all four C functions below before this
facade can be selected in production Linux runtime assembly. They return zero
on success and a nonzero native error on failure:

- `fairyc_private_linux_https_open_v1(address, size, maximum, timeout, output)`:
  takes exact UTF-8 URL bytes including one final NUL; owns no caller buffer after
  return. It receives both the maximum body bytes and a 1..30,000 ms total budget.
  Initialize output to zero. Any nonzero partial-error output must still be a
  valid exclusively owned request which can be consumed by close.
- `fairyc_private_linux_https_headers_v1(raw, status, encoding, capacity, length,
  declared)`: detached status, ASCII/UTF-8 encoding bytes (not Windows UTF-16),
  length at most capacity and Content-Length as usize. SIZE_MAX alone means
  absent. Malformed/ambiguous native headers must fail, not become absent.
- `fairyc_private_linux_https_read_v1(raw, data, capacity, copied)`: one bounded
  binary read, at most capacity bytes. A successful zero copied count means EOF.
  Failure must not retain the borrowed output pointer.
- `fairyc_private_linux_https_close_v1(raw)`: consumes one valid owned request
  even on failure, ends pending I/O/callback ownership before return and never
  leaves a borrowed output pointer live. The caller never retries a close.

Fairy closes every admitted handle exactly once on success or failure, including
partial factory output. A nominally successful factory returning zero is an ABI
failure. Scratch storage outlives request close. Explicit close clears the owned
handle first; automatic resource cleanup cannot issue a second close. A primary
read/header/status failure retains its classification/native cause when close
also fails. A close failure after successful download prevents successful return.

## Admission and bounds

The shared URL owner rejects plaintext, credentials, query/fragment and malformed
source URLs. Linux byte transport also validates UTF-8, NUL and the 4,095-byte URL
bound. It preserves percent escapes and Unicode bytes exactly. Invalid input,
zero/over-16 MiB body bounds and zero/over-30,000 ms budgets fail before native open.
The online operation must supply min(30 seconds, remaining two-minute deadline).

Only status 200 with absent or exact `identity` encoding passes. Declared length
must fit the maximum. Reads use at most 65,536 bytes and one extra-byte probe at
the remaining limit, including an EOF probe after exactly filling the maximum.
Reported copied counts beyond the provided buffer fail before indexing/appending.
Both shorter and longer bodies than a declared length fail. Embedded NUL/0xff
and chunk boundaries preserve exact bytes. No retry, filesystem or publication
operation occurs in this module.

The native provider must independently bound retained response/metadata storage
and one monotonic deadline across setup, headers and reads. It must verify system
TLS trust and host identity; disallow redirects, ambient proxy/auth/cookie config
and compression; retain safe cancellation lifetimes; and work without finding an
external executable through PATH. A facade/model pass does not prove any of those
native requirements. The TLS provider, native assembly and real-socket acceptance
remain separate implementation work.

## Current evidence

`internal/cli/linux_remote_https_test.go` executes the actual Fairy facade with
independent native boundary/fault outputs. The source fixture is
`examples/lexer_candidate/tests/linux_remote_https.fy`. Both-target emission and
Windows MSVC low/optimized executions pass with empty PATH. This is private policy
evidence only. The Linux fetch receipt records exact commands, failures and scope.
