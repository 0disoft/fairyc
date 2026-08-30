# ADR 0352: Bind stable MIR and contract-state payloads in one package artifact

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/packageartifact/stable_compilation.go`
- Package contract: `docs/package/stable-package-compilation-v1.md`
- Depends on: ADR 0171, ADR 0350, ADR 0351

## Context

Stable package MIR v3 and the ADR 0350 contract-state body payload are independently canonical and
verified, but no artifact says they belong to the same package coordinate. Putting state fields
directly into the mature MIR v3 envelope would change every existing MIR artifact and conflate two
independently versioned semantic models. Storing unrelated bytes next to each other without an outer
validator would instead allow a cache producer to pair valid payloads from different packages.

## Decision

1. Define outer schema `fairyc.stable-package-compilation/v1` and exact version
   `fairyc-stable-package-compilation/1`.
2. Store one existing `StableArtifact` MIR value and one existing
   `contractstatemir.StableStateBodyArtifact` value. Keep their internal schemas, hashes, canonical
   encodings, and validation ownership independent.
3. Validate the complete MIR artifact and complete state body through their owning validators, then
   require `MIR.Coordinate == StateBody.Package`. A pair of individually valid payloads from
   different package coordinates fails closed.
4. Project the MIR from one verified process-local package artifact while accepting only an already
   verified state body. This layer does not relower HIR state operations or manufacture a body hash.
5. Encode one strict canonical JSON envelope under a mandatory positive caller byte limit. Decode
   rejects empty or oversized bytes before semantic publication, unknown fields, trailing values,
   incompatible versions, noncanonical JSON, nested MIR corruption, nested state corruption, and
   coordinate mismatch.
6. Keep capability tokens and build-local state package numbers absent. Replacing an empty state
   body with an independently described body under a fresh local package ID and the same coordinate
   preserves every outer byte.
7. Do not change stable MIR v3, its body/context schemas, cache records, cache adapter APIs, MIR
   reassembly, state-symbol alignment, or workspace all-hit behavior in this slice.

## Validation

Tests compose a real extracted stable package MIR artifact with an independently described empty
state body, freeze the complete outer SHA-256, decode and byte-identically re-encode it, prove local
state package renumbering is inert, and assert capability authority text is absent. Negative tests
cover encode/decode bounds, unknown fields, noncanonical whitespace, wrong versions, valid
cross-package payload substitution, and state-body hash corruption. The complete Go suite includes
the existing Windows host smoke.

## Compatibility

This adds a new compiler-private outer artifact without replacing an existing format. Stable
package MIR remains `fairyc-stable-package-mir/3`; function-body and context versions, language
`fairyc-language/12`, public-interface v5, package-cache key v4, runtime ABI, SDK, target, lockfile,
Atlas, and CLI schemas remain unchanged.

## Consequences

One immutable byte sequence can now prove that package MIR and state operations share the same
stable package owner without collapsing their semantic models. ADR 0353 later changes the
persistent cache adapter to publish and retrieve this outer artifact with an expected body hash.
ADR 0354 later aligns cached MIR fresh symbols with rebound state transaction owners and proves
explicit cached oracle execution.
