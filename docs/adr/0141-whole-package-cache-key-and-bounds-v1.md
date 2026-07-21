# ADR 0141: Key whole-package reuse by exact compatibility and require explicit bounds

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/whole-package-cache-v1.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`
- Measurement source: `docs/engineering/03-performance-budget.md`

## Context

The package instance, exact source digest, and dependency public-interface hashes now have concrete
identities, but none is sufficient alone. Reusing output across a compiler, language, MIR, target,
ABI, backend, toolchain, runtime, standard-library, feature, or profile change can silently execute
the wrong program. Using absolute paths or timestamps would instead destroy reproducibility.

Stage 0 has no measured numeric cache budget or stable serialized HIR/MIR format. An unbounded map
would reproduce the disk-growth failure the product is intended to avoid. Inventing a universal
MiB limit before a benchmark exists would turn one developer machine into fake product policy.

## Decision

1. Whole-package cache key v1 hashes a domain-separated canonical tuple containing the exact
   compiler build, language revision, MIR version, target schema and target, ABI profile, toolchain
   capability fingerprint, backend, canonical profile flags, runtime ABI, standard-library package
   instance, current package instance, exact source digest, sorted dependency instance/interface
   pairs, and canonical feature set.
2. The key encoding records the PackageInstanceId, package-source digest, and public-interface hash
   encoding and digest algorithms. Changing one cannot silently reuse a v1 entry.
3. Dependency pairs sort uniquely by full PackageInstanceId. Profile flags and features are
   nonempty valid UTF-8 sets supplied in sorted unique order. Required compatibility strings are
   nonempty valid UTF-8.
4. The encoding uses unsigned 64-bit network-order lengths and full 256-bit identities. SHA-256
   produces the only text form: `fycache1-sha256-` plus 64 lowercase hexadecimal digits.
5. The first store is process-local and concurrency-safe. It requires positive caller-supplied
   entry and byte limits, deep-copies records, rejects an entry larger than the byte limit without
   disturbing prior state, and evicts least-recently-used entries until both limits hold.
6. Records contain the produced public-interface hash and a sorted unique closed set of nonempty
   opaque whole-package typed-HIR, MIR, or object payloads. Those payloads are compiler-private,
   exact-version, disposable bytes, not public exchange formats.
7. Cache miss, incompatible key, corrupt future persistent entry, and collection all fall back to
   rebuilding from source. Cache state never becomes semantic or source authority.
8. Persistent storage, CLI policy, default numeric limits, function-level reuse, packed storage,
   compression, and remote caching remain deferred until measurements and their own contracts exist.

## Consequences

- A private edit invalidates the current package through its source digest while an unchanged public
  interface can preserve dependent reuse.
- Every named compatibility dimension fails closed as a key change instead of relying on one broad
  compiler version.
- The implementation cannot grow without a caller choosing explicit bounds.
- The initial memory store does not accelerate separate one-shot compiler processes. It establishes
  key, copy, accounting, and eviction behavior before filesystem concurrency and crash recovery are
  introduced.

## Verification

- one exact full-key vector freezes v1 canonical encoding;
- fixtures prove compiler, source, dependency-interface, target, feature, and profile changes
  invalidate independently;
- unit tests reject noncanonical sets and duplicate dependency instances;
- store tests prove entry and byte bounds, LRU order, hit/miss/eviction accounting, deep-copy
  isolation, invalid-record rejection, and nondestructive oversize rejection.
