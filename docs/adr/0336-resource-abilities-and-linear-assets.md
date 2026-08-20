# ADR 0336: Experimental resource abilities and linear assets

- Status: Accepted
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Language contract: `docs/language/0009-resource-abilities-v0.md`
- Depends on: ADR 0003, ADR 0011, ADR 0032, ADR 0120, ADR 0218

## Context

Fairy C resources currently guarantee unique ownership and exactly-once cleanup on normal control-flow exits. That model deliberately fits native resources such as files, buffers, arenas, sockets, and opaque operating-system handles. It does not distinguish cleanup from conservation. A blockchain token, authorization ticket, or other linear asset would therefore be silently destroyed when an owning scope ends.

Treating every resource as non-droppable would break existing native code and force ordinary handles through noisy explicit disposal calls. Adding a blockchain-only asset declaration would duplicate ownership, Flow, tagged payload, cleanup, and MIR rules. The narrower solution is to retain one resource model and make implicit destruction an explicit type-level ability.

Move demonstrates the practical value of closed resource abilities, but Fairy C cannot import Move's storage model wholesale. Fairy C already has compiler-owned cleanup bodies, effect-checked cleanup, C transport, and a verified MIR pipeline. The feature must fit those existing boundaries without introducing traits, reflection, hidden runtime state, or chain-specific semantics.

## Decision

1. A resource may declare one `@abilities(...)` attribute. The closed first vocabulary is `drop`, `store`, and `key`.
2. `drop` permits compiler-scheduled destruction. An explicit valid ability set that omits `drop` makes the complete owned value linear with respect to destruction.
3. A resource without `@abilities` preserves the existing behavior and implicitly permits cleanup. This compatibility rule prevents a source-wide migration and does not imply `store` or `key`.
4. `store` and `key` are validated forward-compatible metadata only. `key` requires `store`. Neither ability creates storage layout, serialization, persistence, ABI, runtime, or chain support in this revision.
5. `copy` is not admitted. Every resource remains non-copyable under the existing ownership model.
6. Flow rejects compiler-scheduled destruction of a no-drop resource with `E1108`. Covered cleanup edges include normal and early function exit, nested block exit, loop control exit, and live-resource replacement.
7. Whole-root move, by-value call transfer, and return remain valid because they transfer the single obligation rather than discharging it.
8. The prohibition follows compiler-owned cleanup recursively through resource fields and compiler-known `Option` and `Result` payloads. A droppable outer resource cannot hide and destroy an inner no-drop asset.
9. Invalid, empty, duplicated, unknown, or inconsistent ability declarations are `E1107`. Invalid declarations stop before Flow to avoid secondary cleanup diagnostics.
10. `discard`, unsafe source, raw pointers, and empty cleanup bodies do not bypass the no-drop rule.
11. Explicit destruction is deferred. A future `burn` contract must add a source-visible authorization rule, one explicit MIR obligation-discharge operation, verifier invariants, interpreter behavior, backend lowering, and negative fixtures. It may not be implemented as a generic forget primitive.

## Implementation boundary

The compiler parses abilities from the existing generic attribute AST, canonicalizes them onto resource type metadata, removes the recognized resource-owned attribute before generic unsupported-attribute checks, and consults the canonical metadata from Resource Flow.

This acceptance is limited to direct, single-package experimental semantic analysis. Stable resolved-package analysis keeps `@abilities` unsupported under `fairyc-language/10`; therefore workspace check, build, run, test, cache, and native emission cannot silently acquire the new semantics.

HIR retains the canonical effective ability set so later compiler stages cannot erase the ownership fact. Public-interface v3 fails closed when a public resource carries a non-legacy ability set instead of hashing an incomplete interface. `store` and `key` do not affect MIR, code generation, ABI, runtime behavior, or storage layout.

Stable admission requires one coordinated language-revision cut, a public-interface encoding that includes resource abilities, and the corresponding package, cache, lock, fixture, and release migration. This ADR does not perform that cut and changes no current SDK, runtime ABI, language identity, or cache identity.

## Consequences

Fairy C gains a usable foundation for conserved blockchain assets without becoming a Solidity or Move clone. Existing native resource code remains source-compatible. Asset transfer uses the same ownership machinery already exercised by resource Flow, `Option`, `Result`, MIR cleanup, and backend lowering.

The foundation is not yet a released Fairy C source feature. Keeping the stable package path closed avoids a compatibility promise before interface hashing and downstream lowering can transport the ability set exactly.

The compiler must retain recursive cleanup visibility. Future aggregate, generic, storage, or ABI work cannot erase the inner resource ability set or treat an outer `drop` declaration as permission to destroy every contained value.

The deliberate limitation is that a no-drop asset cannot yet be intentionally destroyed. That restriction is preferable to an unverifiable escape hatch. Mint, burn, split, merge, storage, and contract-profile semantics return as separate evidence-backed proposals.
