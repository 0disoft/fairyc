# ADR 0348: Give contract-state HIR bodies an exact package identity

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/body_hash.go`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- Depends on: ADR 0345, ADR 0346, ADR 0347

## Context

ADR 0347 can lower and execute source-independent HIR state accesses, but those access bodies were
identified only by build-local package and function symbols. Public-interface v5 intentionally
excludes private bodies, and package-cache key v3 therefore cannot distinguish two generated HIR
packages with equal source and interface bytes but different state transactions.

Adding source syntax or cached execution before closing that identity gap would permit stale reuse.
The identity also cannot contain the fresh opaque capability created during lowering: capability
identity is process authority, not reproducible artifact data.

## Decision

1. Define `fairyc.contract-state-package-body/v1`, SHA-256, with the only text form
   `fycsbody1-sha256-<64 lowercase hexadecimal digits>`.
2. Verify the complete HIR transaction set through ADR 0347 lowering before hashing. Malformed
   layouts, owners, effects, operations, values, or exits therefore cannot acquire an identity.
3. Hash the stable package coordinate, optional exact layout hash, and transaction count. Sort
   transactions by a canonical function descriptor rather than build-local symbol order.
4. Encode each function's source-logical module/name, stable parameter/result/drop-glue types, and
   canonical effects. Nominal types use `PackageCoordinateId`, never `hir.PackageID`.
5. Encode exact operation order, operation kind, field index, key/value codec identity and bytes,
   plus the static exit and trap category. Operation reordering is a body change.
6. Exclude source spans, mutable HIR slice identity, build-local package/symbol numbers, and opaque
   transaction capability tokens. Renumbering and input transaction order therefore remain inert.
7. Compute a nonzero identity for an empty transaction set as well as a populated one. Duplicate
   requested packages and duplicate stable function descriptors fail closed.
8. Keep this an identity-only compiler boundary. No package artifact stores or rebinds transaction
   payloads yet, and cache key v3 remains unchanged in this commit.

## Validation

Tests freeze one exact full hash, round-trip its canonical text, preserve it across package/function
renumbering and transaction input order, and change it for layout, signature, encoded value,
operation order, and exit changes. Empty bodies remain nonzero and distinct. Missing effects,
duplicate package requests, missing coordinates, and malformed text fail closed.

## Compatibility

This adds a compiler-private experimental identity. The language remains `fairyc-language/12`;
public-interface v5, package-cache key v3, stable MIR and package artifacts, runtime ABI, SDK,
target, lockfile, Atlas, and CLI bytes remain unchanged.

## Consequences

Generated state transaction bodies now have a deterministic identity suitable for cache and future
artifact binding without confusing process authority with persistent bytes. The next slice must add
this hash to a new whole-package cache-key version and prove body-only invalidation. Stable payload
transport and reassembly remain required before cached state execution can be admitted.

ADR 0349 later completes the cache-key portion with v4. Stable payload transport and reassembly
remain open.
