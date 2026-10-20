# Public-interface hash fixtures

`manifest-v4.json` compiles thirteen complete Fairy C source variants through syntax, semantic Flow,
typed HIR, public-interface extraction, and the v4 hash. The baseline freezes the full digest.
Private implementation, ordinary public bodies, declaration order, private resource
representation, private must-use state, private module constants, and private source-location use must preserve it. Public signatures, public struct field
order, public foreign link names, public must-use state, and public resource abilities must change it. V1 through v3 remain frozen
historical inputs rather than current runner contracts.

Inline-body invalidation is not applicable to current Stage 0, which exposes no public inline function
contract. Before admitting an inline contract or cross-package inline-body reuse, [reopen Phase 3](../../../docs/development/phase-status.md#phase-3-scope-decision-2026-10-21)
and require a separate versioned body hash/invalidation class with body-only dependent-cache
invalidation on cold/warm and persistent paths, retaining private-implementation stability and
incompatible safe-miss checks. This fixture does not serialize HIR, layout, native ABI, object code,
cache records, manifests, or lockfiles.
