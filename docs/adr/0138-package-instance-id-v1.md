# ADR 0138: Fix the first concrete PackageInstanceId encoding

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/package-instance-id-v1.md`
- Graph source: `docs/package/package-graph-v0.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Context

The resolved-package graph already treats an exact tuple of package UID, source identity, version
identity, and sorted features as one logical instance. It previously used pairwise tuple comparison
and deliberately exposed no serializable identity. Phase 3 lockfiles, whole-package cache keys, and
Atlas maps all need a deterministic package-instance key before their own formats can be decided.

Reusing build-local package numbers would leak graph traversal order. Concatenating strings would
lose component boundaries. A shortened digest would require a collision table in every consumer.
BLAKE3 or another new digest dependency may be faster on large byte streams, but these identity
tuples are tiny and the repository already uses SHA-256 for deterministic compiler-owned storage.
Digest throughput is not the current build bottleneck.

## Decision

1. `PackageInstanceId` v1 hashes a versioned, domain-separated, length-prefixed canonical encoding
   of the existing exact identity tuple.
2. The digest is the complete 256-bit SHA-256 result. Its canonical text is
   `fypi1-sha256-` followed by 64 lowercase hexadecimal digits.
3. The encoding uses exact valid UTF-8 bytes. It performs no Unicode, path, separator, case, URL,
   Git, or version normalization. The resolver producing each opaque component owns those rules.
4. Features must be nonempty, valid UTF-8, sorted, and unique before hashing. Input order cannot
   become a second identity for the same resolved feature set.
5. A repeated full digest denotes one class only after the complete identity tuple compares equal.
   An unequal tuple with the same digest is an internal collision error, never silent deduplication.
6. Package graph resolution records the full ID on every instance class and uses it for class lookup.
   Build-local HIR package numbers remain separate and must not be serialized as this ID.
7. The ID is not a package coordinate, dependency request, module path, filesystem name, native ABI
   symbol, cache format, manifest, lockfile, registry key, or proof that source bytes are trusted.
8. No 128-bit display alias is introduced. Atlas may add one only with an explicit full-digest
   collision table and compatibility contract.

## Consequences

- Identity class lookup becomes expected linear time rather than pairwise tuple scanning.
- Lockfiles, caches, and `.fymap` can share one package-instance reference without sharing schemas.
- Changing any component, feature, field order, tag, length encoding, domain, digest, or text prefix
  creates a different versioned identity contract.
- The current implementation materializes one small canonical byte slice per graph node. A streaming
  implementation may replace that allocation only if it produces identical fixture bytes.
- Public manifest limits, source acquisition, and dependency solving remain separate Phase 3
  decisions. ADR 0139 subsequently fixes the independent public-interface hash.

## Verification

- versioned fixtures freeze empty-feature, multi-feature, and ambiguous concatenation vectors;
- negative fixtures reject unsorted, duplicate, and empty features with stable reasons;
- unit tests reject invalid UTF-8 and noncanonical text encodings;
- graph tests prove identical tuple deduplication carries the expected full ID;
- the graph rechecks the complete tuple before accepting an existing digest class.
