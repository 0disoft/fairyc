# ADR 0342: Release resource abilities and on-chain effects in language 11

- Status: Accepted
- Date: 2026-08-30
- Product source: `docs/product/02-spec.md`
- Language contracts: `docs/language/0008-effects-and-capabilities-v0.md`, `docs/language/0009-resource-abilities-v0.md`
- Interface prerequisite: `docs/adr/0341-public-interface-resource-abilities-v4.md`

## Context

ADRs 0336 and 0337 proved resource abilities and target-neutral on-chain effect classification only
through direct experimental semantic analysis. Public-interface v4 now transports effective
resource abilities without erasure. Keeping the same syntax rejected by resolved-package analysis
would leave the compiler's primary check, build, run, test, cache, and native paths unable to use the
proven ownership and authority facts.

## Decision

1. `fairyc-language/11` admits `@abilities(drop, store, key)` and the five target-neutral effects
   `block_context`, `emit`, `external_call`, `storage_read`, and `storage_write` in resolved packages.
2. The complete twelve-effect vocabulary is canonical for language 11. Missing caller authority
   remains `E7102`; malformed effect declarations remain `E7101`.
3. Existing resource Flow rules, including `E1108` for compiler-scheduled destruction of a no-drop
   resource, apply unchanged to stable package compilation.
4. Public-interface v4 transports abilities and effect strings. Package cache fixture v6 uses v4
   dependency hashes and the exact `/11` language identity.
5. The compiler-bundled SDK advances to `0.0.0-stage0.53`; new scaffolds and checked-in examples use
   language 11. Runtime ABI remains `fairyc-runtime/0` because neither capability adds runtime data.
6. Every workspace root must declare language 11, including programs that do not import `std`.
   Language 10 inputs fail with `language-revision-mismatch`; no source rewrite is required because
   the admitted attributes already use the current grammar.

## Runtime boundary

An effect declaration exposes required authority but does not invent a host operation. Storage,
event, block-context, and external-call APIs still require explicit parameters, SDK contracts, MIR
operations where necessary, backend adapters, and target evidence. `store` and `key` remain static
eligibility facts; they do not serialize a value or grant ambient storage access.

## Evidence

Resolved-package semantic tests carry canonical abilities and on-chain effects into HIR. Public
interface and cache fixtures prove invalidation, and the language-11 native quickstart carries both
attributes through stable build and run while returning 42 without new ABI symbols.

## Consequences

Workspace manifests, generated locks, SDK identity, cache vectors, documentation, and release
fixtures move together. Reverting this ADR requires reverting the complete language-11 identity cut
rather than silently accepting the new spellings under language 10.
