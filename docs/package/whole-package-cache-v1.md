# Whole-package cache key and bounded memory store v1

- Status: Executable prototype contract
- Decision: `docs/adr/0141-whole-package-cache-key-and-bounds-v1.md`
- Fixture source: `spec/fixtures/package-cache/manifest-v1.json`
- Implementation source: `internal/packagecache/`
- Fixture runner: `internal/packagecachefixture/`

## Key input

One already-resolved package build supplies:

- full compiler build identity;
- exact language and MIR versions;
- target schema plus target identity and ABI profile;
- toolchain family/baseline/capability fingerprint, never a local executable path;
- backend identity and sorted unique language-owned profile flags;
- runtime ABI and exact standard-library `PackageInstanceId`;
- current package `PackageInstanceId` and exact `fysrc1-sha256` source digest;
- dependency pairs sorted uniquely by full instance ID, each paired with the consumed
  `fyif1-sha256` public-interface hash;
- sorted unique resolved feature set.

The key also writes the v1 identity/source/interface encoding and digest names. This is deliberate
redundancy: a future algorithm cannot look compatible merely because its raw digest width matches.

## Canonical encoding

The encoder writes ASCII `fairyc.package-cache-key`, a zero byte, version byte `01`, the three
component encoding/algorithm pairs, then every compatibility string in the order above. Strings and
string arrays use unsigned 64-bit network-order lengths and counts. Full package IDs, source digest,
and interface hashes write their 32 digest bytes. Dependency pairs remain sorted by package digest.

SHA-256 hashes the complete bytes. The canonical text is:

```text
fycache1-sha256-<64 lowercase hexadecimal digits>
```

Unknown versions, malformed text, omitted all-zero identities, noncanonical sets, and duplicate
dependency instances are misses or invalid producer input, never partial matches.

## Store behavior

`MemoryStore` requires both `MaxEntries > 0` and `MaxBytes > 0`. There is no default because Stage 0
has no measured universal storage budget. Logical record size counts the full key, output-interface
hash, artifact metadata, and artifact bytes. `Put` rejects an individually oversized record before
changing prior state, replaces the same key atomically under the process mutex, and evicts the
least-recently-used entries until both limits hold. `Get` promotes a hit and returns a deep copy.

Artifact kinds are closed to `typed-hir`, `mir`, and `object`. They must be nonempty, sorted, and
unique. Their byte encoding is intentionally unspecified outside the exact compiler build that put
the record. The output public-interface hash is metadata for dependent invalidation, not proof that
the opaque payload is compatible.

The store is process-local. Persistence, cross-process locking, crash-safe writes, collection across
compiler builds, default limits, command-line inspection, and remote transport remain unimplemented.
Those features must preserve the v1 miss/fallback behavior but require measured policy and a separate
storage-format decision.

ADR 0146 records the current evidence result: persistence is deliberately deferred. ADR 0149 adds a
real exact-version codec for one verified whole-program MIR value, but that payload is not yet a
package-granular cache artifact and no package-aware object is produced. No configured cold/warm
measurement intent owns comparable evidence. Fixture or codec payload sizes alone cannot justify a
disk format or default budget.
