# ADR 0165: Identify persistent-cache compiler builds by executable digest

- Status: Accepted primitive
- Date: 2026-07-22
- Implementation: `internal/buildidentity/identity.go`
- Consumer prerequisite: `docs/adr/0164-verified-stable-disk-mir-reuse.md`

## Context

Persistent package-cache keys require the full compiler build identity. Fairy C has no released
compiler version source yet, and a SemVer value would still be too broad: two binaries from the same
release can contain different lowering fixes, build tags, or local modifications. Branch names,
moving tags, source timestamps, and the executable filename are not content identities.

## Decision

Define `fairyc.compiler-executable-sha256/v1:<64 lowercase hex>` as the exact digest of the running
compiler executable bytes. Hashing streams through a fixed 128 KiB buffer. Empty images and every
open, read, or close failure reject; there is no version-string fallback that could accidentally
widen cache compatibility.

Linux prefers `/proc/self/exe`, which continues to name the running image after an on-disk rename or
unlink, and falls back to `os.Executable`. Other hosts use `os.Executable`; Windows prevents ordinary
replacement of the running image. Signing, rebuilding, or otherwise changing executable bytes
intentionally creates another build identity and therefore cache misses.

## Evidence

Unit tests prove byte determinism, byte sensitivity, exact domain-labeled text, and empty-input
rejection. A separate test process launched from the same executable must produce the exact parent
identity. Hosted Go tests provide the corresponding Linux observation; local tests provide Windows.

## Consequences and limits

- A separately invoked unchanged `fairyc` executable can safely share persistent cache keys.
- Hashing the executable adds one sequential read to the experimental persistent-cache command. It
  is not used by ordinary commands and has no performance claim yet.
- Reproducible source builds may still miss if their executable metadata differs. Safe misses are
  preferred to claiming compatibility across byte-distinct compiler builds.
- The identity says which executable bytes ran; it does not sign or authenticate the executable.
