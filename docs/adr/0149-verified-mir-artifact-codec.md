# ADR 0149: Encode compiler-private verified MIR artifacts

- Status: Accepted for prototype
- Date: 2026-07-22
- MIR source: `internal/mir/`
- Codec source: `internal/mircodec/`
- Cache source: `docs/package/whole-package-cache-v1.md`
- Measurement source: `docs/engineering/03-performance-budget.md`

## Context

The package cache closes artifact kind names but deliberately treats their bytes as opaque. ADR
0146 therefore blocks persistent storage until at least one real compiler-owned payload can be
encoded, decoded, and verified. The package-aware Windows build now produces verified MIR, but the
MIR remains one whole-program value: Stage 0 does not yet lower or link independently reusable
per-package MIR or object units.

Serializing that whole-program value is still useful as a narrow prerequisite. Pretending it is
already a reusable package-cache record would be wrong because the build-local package numbers and
cross-package functions belong to one resolved graph.

## Decision

1. `internal/mircodec` owns exact-version `fairyc.mir-artifact/v0` envelopes containing
   `fairyc-mir/0` and one complete `mir.Program`.
2. Encode and decode both require a caller-supplied positive byte limit. The codec chooses no
   product cache budget.
3. Encode accepts only MIR that passes the existing verifier. Decode rejects empty, oversized,
   malformed, unknown-field, incompatible-schema, incompatible-version, trailing-value, and
   noncanonical bytes, then reruns the verifier before returning the program.
4. Canonical bytes are compact Go JSON for the exact compiler implementation. They are private,
   disposable, and not a stable interchange, debugging, plugin, or remote-cache format.
5. The codec is not wired into `packagecache.MemoryStore` or workspace builds yet. A later slice
   must define a genuinely reusable package boundary or explicitly key a whole-program artifact by
   every graph input before any cache hit can bypass compilation.

## Consequences

- ADR 0146's generic “no compiler-owned artifact bytes” blocker is narrowed: verified whole-program
  MIR now has real round-trippable bytes.
- Persistent cache storage remains deferred. There is no package-granular producer, cache hit in a
  build, configured cold/warm measurement, disk policy, corruption fallback, or cross-process
  coordination.
- Exact re-encoding makes whitespace, alternate field order, and other semantically equivalent but
  noncanonical JSON a miss instead of multiplying stored representations.

## Verification

- unit tests prove deterministic encoding and exact structural round-trip;
- limit, empty input, unknown field, noncanonical whitespace, schema, and MIR-version failures are
  fail closed;
- mutated entry control flow is rejected both before encoding and after decoding;
- the full Go test, build, and vet suites cover the new package without adding runtime dependencies.
