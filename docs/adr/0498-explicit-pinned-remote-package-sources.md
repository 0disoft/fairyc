# ADR 0498: Explicit pinned remote source preparation

- Status: Accepted bounded prototype
- Date: 2026-11-01
- Extends ADRs 0143/0144/0145 through a separate explicit preparation step.
- Owner: [remote source contract](../package/remote-sources-v1.md).

Admit `fairy.sources.json` and `fairyc experimental fetch-packages [--offline]` for HTTPS ZIP
sources pinned by exact package UID, manifest SemVer, URL and archive SHA-256. Source paths are
explicit `vendor/<portable-package-name>` members of the existing root manifest. Selection is
the exact declared version, not a range solver. Transitive packages must also be explicitly
catalogued and pinned; remote manifests do not trigger more networking or execute scripts.

The compiler still consumes ordinary local members. `fairy.lock` retains its existing local
source digest/graph representation. The source plan binds transport provenance separately;
fetch verifies it, while normal check/build/run/test remain network-free. This does not claim
that a successful ordinary check reauthenticates a remote origin or the vendored non-source files.

Fetch must enforce bounded TLS downloads without redirects, archive hashes, portable contained
extraction, exact manifest identity and conservative existing-tree verification. Publish one
complete new package directory at a time; never merge, update or delete an existing package.
Cache hits must be rehashed; offline misses and corruption fail without networking. Completed
packages/cache blobs may remain after a later package fails; no workspace-wide transaction or
power-loss claim is made. Verified source preparation never overwrites `fairy.lock`.

SHA-256 proves equality to maintainer-reviewed bytes, not publisher authenticity or source safety.
Credentials, signatures, registries, semver range resolution, Git fetching, automatic upgrades,
publication, implicit builds and Fairy-authored Linux parity are separate future slices.
Registry selection remains UNDECIDED. Linux native execution stays in the combined queue.

The 2026-11-04 bounded extension connects the Windows Fairy-authored CLI's explicit
`experimental fetch-packages --offline` to the authored cached-archive restoration path.
This preserves the preparation contract and admits no online fallback. Shared success and
invocation/help output and exit codes match Go; preparation-error detail is candidate-owned.
See the [CLI contract](../cli/command-contract.md#experimental-explicit-remote-package-preparation).

The subsequent Windows extension connects the invocation without `--offline` to
Fairy HTTPS orchestration and validated immutable cache publication. WinHTTP supplies
system TLS/I/O; Fairy owns URL, size, archive, identity and publication policy. Native
I/O waits obey the remaining request deadline, with final callback-aware cancellation.
Offline execution and stable compilation gain no network fallback. See the
[online receipt](../ops/windows-candidate-remote-online-20261104.md); Linux remains queued.
