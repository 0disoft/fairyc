# ADR 0501: Isolated Linux HTTPS producer and bounded pipe transport

- Status: Accepted bounded producer decision; parent adapter and Linux execution pending
- Date: 2026-11-06
- Extends ADR 0498 and `architecture/candidate-linux-https-v1.md`.

Use the system libcurl 8-or-newer compatible `libcurl.so.4` API for Linux package
HTTPS I/O, with fixed reference Ubuntu/Debian amd64 library and system CA paths.
Do not invoke a curl executable, download another TLS stack, embed Go HTTP or
silently select an insecure/fallback transport. Native compilation of this
optional producer requires libcurl development headers; the system runtime and
`ca-certificates` are runtime prerequisites for online fetch. Neither is added to
ordinary Go dependencies or the current baseline toolchain provisioning by this
decision. Other distribution paths and installation/bundling are not certified.

Run TLS/DNS/global libcurl setup and cleanup only in a dedicated request child.
The planned parent owns one monotonic deadline covering startup, metadata and
body reads, and must terminate and reap its exclusively owned child on early
failure, timeout or close. No borrowed parent pointers enter the child: metadata
and binary body frames cross a private pipe. The child must not inherit unrelated
workspace descriptors. Initial process creation, signal ownership, parent-death
behavior, partial frames, absolute timeout and kill/reap error cases require their
own implementation and evidence before runtime selection.

This boundary is required because libcurl's NOSIGNAL setting with a synchronous
resolver cannot itself time out a DNS lookup. Its ordinary timeout is useful
defense in depth, not proof of the caller's deadline. See the primary
[NOSIGNAL documentation](https://curl.se/libcurl/c/CURLOPT_NOSIGNAL.html) and
[thread/global initialization rules](https://curl.se/libcurl/c/threadsafe.html).
Process isolation is the project's design choice; it is not a guarantee supplied
by those libcurl options. The producer function is not an in-process threaded
general-purpose client and must not be called from an arbitrary multithreaded
host instead of its owning child.

The child uses fresh easy/global state, HTTPS-only protocols, no redirect/auth/
cookie state, an explicit empty proxy, system certificate and hostname checks,
TLS 1.2 minimum and no automatic content decompression. It sends only an explicit
Accept: application/zip request header. URL escapes are retained. Native and
library failures close all successfully acquired local library/easy/header/global
resources before emitting an end frame; cleanup callback storage remains live.
The parent still owns robust final process cleanup even if a TLS library blocks.

The child/parent wire is private to one amd64 image revision. Frames carry a
12-byte kind/length/code header and at most 65,536 payload bytes. A single
144-byte detached response header contains status, encoding and optional length.
Response metadata is bounded to 64 KiB / 1,024 lines across informational, final
and trailer headers. Conflicting lengths, length/transfer ambiguity, folded or
malformed fields and framing-relevant trailers reject. Identical decimal lengths,
informational responses and benign trailers preserve their separate meanings.
The body retains no complete archive; it forwards at most maximum+1 bytes so the
Fairy caller can distinguish an exact limit from overflow. OS errors are nonzero
u32 values; libcurl errors use 65,536 plus CURLcode. No native error text is logged.

Current implementation: source assets, bounded metadata/body producer, dynamic
system API setup and independent Windows loader/libcurl/callback controls. These
assets are deliberately not selected by LinuxSource until the owning parent
adapter exists. The authored native loopback test rejects a fresh untrusted TLS
certificate without modifying any trust store, but has not run. Positive trusted
TLS, DNS cancellation, actual child lifetime, online orchestration, CLI selection,
distribution prerequisites and all broader/final gates remain open.

No public command/SDK, support flag, Go dependency, DB, CI or runner is changed.
