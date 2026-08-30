# ADR 0349: Add exact contract-state body identity to cache key v4

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Cache contract: `docs/package/whole-package-cache-v4.md`
- Fixture source: `spec/fixtures/package-cache/manifest-v8.json`
- Depends on: ADR 0174, ADR 0345, ADR 0348

## Context

ADR 0348 gives every verified source-independent HIR transaction set an exact stable package-body
hash, but whole-package cache key v3 does not consume it. Two compiler inputs can therefore have
equal acquired source, public interface, and dependency facts while carrying different generated
state operations. A future compiler path that executed those bodies under v3 could reuse stale MIR.

The body is private implementation. Invalidating every dependent through public-interface identity
would be correct but unnecessarily broad; stable function binding already permits a caller artifact
to remain reusable when a dependency body changes without changing its public signature.

## Decision

1. Replace cache-key encoding v3 with `fairyc.package-cache-key/v4` and the only text form
   `fycache4-sha256-<64 lowercase hexadecimal digits>`. V1 through v3 are incompatible misses.
2. Add the nonzero `fairyc.contract-state-package-body/v1` hash immediately after the current
   package's public-interface v5 hash. The encoding also writes the body-hash encoding and algorithm
   identities before compatibility inputs.
3. Compute all function-owning package body hashes from the completed typed HIR once, after semantic
   readiness and before any memory or persistent cache lookup. Malformed generated transactions fail
   before a cache decision.
4. Require every function-owning package to have both a current public interface and a current body
   hash. Missing or zero body identity fails closed.
5. Put only the owning package's body hash in its key. Dependency entries remain sorted
   `PackageCoordinateId + public-interface v5`; a private dependency-body change misses its own key
   without invalidating callers whose consumed interface is unchanged.
6. Keep persistent record and stable MIR artifact schemas unchanged. Cache records are disposable,
   and the v4 key namespace prevents a v3 record from being selected.
7. Keep state-operation payload transport closed. This change proves exact invalidation, not that a
   stable artifact can serialize, rebind, or execute contract-state transaction bodies.

## Validation

Unit tests freeze one full v4 key, reject a missing body hash, and prove body changes invalidate.
Workspace compiler tests compute real empty-body identities for a two-package workspace, alter one
owner's identity, require only that owner's key to change, and reject a missing owner identity.
Fixture manifest v8 freezes a second full key and independently changes compiler, language, source,
current interface, current body, dependency interface, target, feature, and profile inputs.

## Compatibility

Package-cache v4 intentionally invalidates every v3 memory or disk entry. No migration is required.
The language remains `fairyc-language/12`; public-interface v5, stable MIR/package artifacts, runtime
ABI, SDK, target, lockfile, Atlas, and CLI output schemas remain unchanged.

## Consequences

The compiler now has an exact body identity and an executable cache invalidation path before source
syntax is admitted. ADR 0350 later adds stable transaction payload serialization, bounded decode,
fresh capability reconstruction, and verifier re-entry. Stable owner rebind and cache reassembly
evidence remain required before cached state execution can be enabled.
