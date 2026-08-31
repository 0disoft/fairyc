# Stable package compilation artifact v1

- Status: Experimental compiler-private contract
- Decisions: `docs/adr/0352-stable-package-compilation-artifact.md`,
  `docs/adr/0353-stable-compilation-cache-records.md`,
  `docs/adr/0354-combined-cached-state-reassembly.md`,
  `docs/adr/0355-unified-verified-compiler-result.md`
- Implementation: `internal/packageartifact/stable_artifact.go`,
  `internal/packageartifact/stable_compilation.go`, `internal/packageartifact/stable_cache.go`,
  `internal/workspacecompiler/session.go`
- Schema: `fairyc.stable-package-compilation/v1`
- Version: `fairyc-stable-package-compilation/1`

## Contents

One artifact contains exactly:

- one stable package MIR v3 value with its package instance, coordinate, projected type context,
  stable function bodies, and exact imports;
- one contract-state package-body artifact v1 with the same coordinate, exact layout and body hash,
  stable function descriptors, canonical operations and values, and explicit exits.

The outer artifact does not rewrite either nested representation. Each nested validator remains the
authority for its own identity and semantics.

## Required invariants

- both nested artifacts validate completely before encoding or after decoding;
- the MIR coordinate and state-body package coordinate are byte-identical;
- the caller supplies a positive maximum encoded byte count;
- JSON has the exact schema/version, field set, field order, and canonical byte representation;
- build-local package/function numbers, source spans, and transaction capability tokens do not enter
  the state payload;
- local state package renumbering under the same coordinate preserves complete artifact bytes.

## Persistent cache admission

ADR 0353 publishes exactly one `compilation` payload per persistent record. Lookup requires the
cache-selected nonzero state-body hash and verifies the record interface, package instance, complete
outer artifact, shared coordinate, and exact body hash. Any record-derived contradiction quarantines
only the immutable generation read; a missing caller hash does not delete a valid generation. The
persistent compiler identity `/2` makes MIR-only `/1` records incompatible misses.

## Combined all-hit reassembly

ADR 0354 retains both verified members on a persistent all-hit. Stable MIR reassembly returns its
fresh coordinate-owned logical-function bindings; every cached state transaction must bind through
the matching MIR symbol after current layout/signature verification and fresh capability creation.
ADR 0355 exposes those transactions through the same nested verified `compiler.Result` used by cold
and process-local paths. ADR 0357 rechecks that each fresh current transaction owner is call-free;
ADR 0358 also rechecks that it contains no dynamic control flow. Cached operations cannot bypass
those current-HIR boundaries. ADR 0359 additionally rejects unsafe code and recursively reachable
raw-pointer types while preserving ordinary safe-borrow and pointer-free resource types. ADR 0360
also rechecks that fresh current owners have no receiver or drop-glue execution semantics while
preserving explicit static associated functions. Source
syntax, call or conditional-state admission, pointer provenance, backend,
runtime, production persistence, and target admission remain closed.
