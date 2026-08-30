# ADR 0341: Encode resource abilities in public-interface v4

- Status: Accepted
- Date: 2026-08-30
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0009-resource-abilities-v0.md`
- Package contract: `docs/package/public-interface-hash-v4.md`

## Context

ADR 0336 added canonical `drop`, `store`, and `key` metadata to typed HIR but deliberately kept a
public resource with non-legacy abilities out of public-interface v3. Stable package admission
cannot open while a dependency hash can erase whether a resource may be implicitly destroyed or
stored as persistent state.

## Decision

1. Public-interface v4 adds one canonical ability list to every public item record.
2. Public resources encode their effective abilities. An omitted source attribute encodes `drop`.
3. Non-resource items encode an empty list so the record layout stays uniform and deterministic.
4. Unknown, duplicate, noncanonical ability lists and `key` without `store` fail closed.
5. The binary version byte becomes `04`; the text prefix becomes `fyif4-sha256-`.
6. Public ability changes invalidate dependent package cache keys through the existing interface
   hash field.
7. V1 through v3 remain immutable historical contracts.

## Evidence

The v4 manifest preserves all v3 stability and invalidation cases and adds a source fixture proving
that a public resource ability change changes the hash. Unit tests additionally reject malformed
HIR ability sets before encoding.

## Boundary

This decision satisfies the interface-transport prerequisite for stable resource abilities. It
does not change the current language revision, stable package semantic admission, MIR, C emission,
runtime ABI, SDK, or target claims.

## Consequences

Existing packages receive new interface hashes even when they use only implicit `drop`, because the
encoding version changes. Cache and fixture contracts must consume v4 explicitly rather than
accepting old prefixes accidentally.
