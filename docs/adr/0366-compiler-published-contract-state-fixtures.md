# ADR 0366: Execute contract-state fixtures from one verified compiler result

- Status: Accepted
- Date: 2026-08-31

## Context

ADR 0364 and ADR 0365 created versioned cell and map execution corpora, but their runner called
`contractstatemir.LowerHIRTransactions` directly. That proves the state-MIR verifier and semantic
oracle, yet bypasses the compiler coordinator that must publish ordinary MIR and contract-state MIR
atomically. A regression could therefore leave the fixtures green while the real compiler returned
partial executable IR or rejected the same HIR at its ordinary MIR boundary.

The fixture package remains source-independent. It needs compiler publication evidence without
claiming that contract-state declarations or operations have been admitted into Fairy source.

## Decision

1. Wrap each fixture HIR package in a completed, diagnostic-free `semantic.PhaseFlow` result and
   submit it to `compiler.LowerAnalyzed`.
2. Require exactly one verified ordinary MIR function and one published contract-state transaction
   before constructing the semantic-oracle store. Execute only the transaction returned by the
   compiler result; the runner no longer calls state lowering directly.
3. Keep both historical v0 and current v1 manifests unchanged. This is a runner-strengthening
   decision, not a new fixture data format.
4. Add a negative publication regression that changes one canonical `u64/be/v1` fixture value into
   a one-byte noncanonical encoding after HIR construction. The compiler must return
   `contract-state-mir-lower`, `Verified == false`, zero ordinary MIR functions, and zero published
   state transactions.
5. Do not infer source admission from a manually completed semantic result. Source parsing,
   semantic construction of state HIR, stable CLI behavior, native lowering, runtime persistence,
   and deployable targets remain closed.

## Verification

Every v0 and v1 case now proves the chain
`HIR -> compiler.LowerAnalyzed -> verified compiler.Result -> contract-state oracle`.
The negative case proves that successfully lowered ordinary MIR is still withheld when later state
MIR verification fails.

## Consequences

The versioned transaction corpus now exercises the same atomic publication boundary used by cold
compiler requests. Future cached-result and source admission work can depend on this runner without
mistaking direct subsystem success for a valid complete compiler result.
