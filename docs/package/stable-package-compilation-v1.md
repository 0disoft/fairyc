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
- Version: `fairyc-stable-package-compilation/3`

## Contents

[ADR 0423](../adr/0423-lossless-mir-static-bytes.md) advances stable MIR to `/5`
and function bodies to `/4` for lossless Base64 static bytes. Old payloads are
rejected; the outer schema stays v1 and the nested contract-state
representations are unchanged.

One artifact contains exactly:

- one stable package MIR v3 value with its package instance, coordinate, projected type context,
  stable function bodies, and exact imports;
- one contract-state package-body artifact v1 with the same coordinate, exact layout and body hash,
  stable function descriptors, canonical operations and values, and explicit exits.

The outer schema remains v1 while the nested state body selects its compatible v1 linear, v2 graph,
v3 helper, or v4 borrow envelope. ADR 0381 makes borrow identities and view bounds stable inputs
without changing stable package MIR v3 or prior nested bytes.

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
and process-local paths. ADR 0357 keeps unrepresented calls closed; ADR 0379 permits only a stable
v3 payload's exact same-package no-argument void helper graph and rechecks the current root/helper
call sequence during fresh rebind;
ADR 0358 also rechecks that it contains no dynamic control flow. Cached operations cannot bypass
those current-HIR boundaries. ADR 0359 additionally rejects unsafe code and recursively reachable
raw-pointer types while preserving ordinary safe-borrow and pointer-free resource types. ADR 0360
also rechecks that fresh current owners have no receiver or drop-glue execution semantics while
preserving explicit static associated functions. ADR 0361 additionally rechecks that fresh current
owners contain no retained checked arithmetic or bounds operation with an unrepresented trap edge.
ADR 0362 rechecks that fresh owners carry neither owned resource values nor lexical regions while
preserving safe resource borrows and static resource namespaces. ADR 0363 rejects every fresh-owner
statement, expression, operator, or expression-local effect outside the closed positive allowlist.
ADR 0381 additionally rechecks v4 block-confined state-borrow provenance and view bounds before
cached execution. Source syntax, general call or
conditional-state admission, raw-pointer exposure, backend, runtime, production persistence, and
target admission remain closed.
