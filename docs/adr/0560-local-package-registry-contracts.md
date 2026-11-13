# ADR 0560: Local registry preparation contracts

- Status: Accepted for the Go Stage 0 experimental preparation surface
- Date: 2026-11-14

Extend package preparation without adding implicit build networking or changing
the exact manifest/source-plan/lock schemas. Resolve a bounded reviewed catalog
to existing exact HTTPS/SHA-256 pins. The first range family is exact stable
versions, caret and tilde, with canonical uint32 components; prerelease/build,
wildcards, disjunctions and dependency solving are not inferred.

Require detached Ed25519 signatures over domain-separated exact catalog bytes,
explicit local public keys, a caller-reviewed sequence floor and an unexpired
catalog. No automatic key discovery/TOFU or external registry endpoint is added.
Updates are reviewable proposals preserving package identities and refusing
downgrades or changed content for an existing exact version. No manifest, source
plan, trust floor or lock is silently changed.

Local Git preparation reads a caller-selected repository and full immutable
commit. Read tree/blob objects without hooks, attributes, checkout, filters,
submodules, replacement objects or lazy network retrieval. Reuse the bounded
portable archive/package validators, return an archive digest and package/commit
receipt, and create output exclusively. It does not authenticate a publisher.

Each implementation lands separately. Remote Git/registry hosting, credentials,
key distribution/revocation services, signed transparency and automatic update
application remain separate work; a local verifier is not a deployed ecosystem.
