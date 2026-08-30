# Whole-package cache key and bounded memory store v1

- Status: Historical; superseded by `whole-package-cache-v2.md`
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
  `fyif2-sha256` public-interface hash;
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

Artifact kinds are closed to `compilation`, `typed-hir`, `mir`, and `object`. ADR 0353 adds
`compilation` for the exact-version stable package bundle without changing the generic record
envelope; ADR 0354 consumes both bundle members during verified all-hit reassembly. Kinds must be
nonempty, sorted, and unique. Their byte encoding is intentionally
unspecified outside the exact compiler build that put the record. The output public-interface hash
is metadata for dependent invalidation, not proof that the opaque payload is compatible.

`MemoryStore` remains process-local. ADR 0163 separately adds a caller-bounded immutable-generation
`DiskStore`: canonical records live under key and payload digests, concurrent identical writers
converge without a lock file, conflicting valid generations are safe misses, corrupt generations
are rejected, and collection enforces entry/byte/temp-age limits. No compiler command or Session
uses it by default. ADR 0164 adds an internal opt-in Session path with mandatory caller policy and
exact compiler-build identity. ADR 0165 derives that identity from executable bytes, and ADR 0166
adds one explicit experimental workspace command. Default limits, an ordinary-command policy, a
user-level shared path, cold/warm disk measurement, inspection/cleanup commands, and remote
transport remain unimplemented.

ADR 0146 records the current evidence result: persistence is deliberately deferred. ADR 0149 adds a
real exact-version codec for one verified whole-program MIR value, but that payload is not yet a
package-granular cache artifact and no package-aware object is produced. ADR 0152 establishes a
package-stable function identity so a later artifact can name explicit imports without serializing
build-local HIR symbols or MIR order. ADR 0153 now adds independently verified process-local package
function slices and fresh-symbol reassembly. ADR 0154 adds a bounded canonical codec whose random
identity deliberately rejects bytes outside the creating compiler process, allowing the existing
byte-bounded LRU to own the full payload without claiming persistence. Every slice still carries the
same workspace-wide type context and no compiler path reads it from `MemoryStore`. No configured
cold/warm measurement intent owns comparable cache-hit evidence. Process-codec payload sizes alone
cannot justify a disk format or default budget.

ADR 0162 supplies the stable package MIR payload required by a truthful disk record. ADR 0163 adds
the disposable bounded storage primitive. ADR 0164 connects both through explicit configuration,
ADR 0166 exposes only that explicit policy as an experimental command, ADR 0168 defines the
verified-hit-or-safe-miss crash outcomes, and ADR 0169 binds all workspace-cache I/O to opened roots.
None claims that a disk hit is faster than ordinary lowering.

ADR 0155 connects those bytes to the bounded LRU and distinguishes verified hits from raw store
hits, misses, and rejected entries. Rejection uses complete-record compare-and-delete so a stale
reader cannot remove a concurrent replacement. ADR 0156 adds a semantic-first workspace compiler
session that computes v1 keys and skips MIR lowering only after complete verified hits plus exact
reassembly. Partial hits, rejected payloads, and incompatible mixed-generation contexts all fall
back to ordinary lowering and refresh current artifacts. Memory mode remains process-local; stable
disk mode remains opt-in and has no elapsed-time speedup, default path, or default-limit claim.
