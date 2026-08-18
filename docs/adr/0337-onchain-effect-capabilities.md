# ADR 0337: Target-neutral on-chain effect capabilities

- Status: Accepted prototype slice
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Language contract: `docs/language/0008-effects-and-capabilities-v0.md`
- Depends on: ADR 0003, ADR 0008, ADR 0116, ADR 0121, ADR 0218, ADR 0328

## Context

Fairy C already has a closed source-visible effect vocabulary, canonical typed-HIR effect sets, a
direct-call caller-superset rule, and Resource Flow checks for effects caused by compiler-scheduled
cleanup. That machinery can represent blockchain authority without adding Solidity-style modifiers,
ambient transaction globals, a second call checker, or a chain-specific ownership model.

The missing contract is the authority vocabulary. A future contract body needs to distinguish
persistent-state reads from writes, event emission from storage mutation, external contract calls
from ordinary internal calls, and deterministic chain context from host clock or environment access.
Treating all of those as `network` or one broad `chain` effect would erase the distinctions required
for review, view classification, capability reduction, and target policy.

A separate `@view` attribute initially looks convenient, but it would create two declarations of the
same fact. The compiler would then need conflict rules for combinations such as
`@view @effects(storage_write)`, every public-interface and cache format would need another field,
and client generators could disagree with the effect checker. View eligibility can instead be
derived exactly from the canonical effect set.

## Decision

### 1. Add five exact target-neutral effects

The closed effect vocabulary gains these spellings:

| Effect | Meaning |
| --- | --- |
| `block_context` | Observe target-supplied block, transaction, caller, value, height, timestamp, or chain context through an admitted explicit API |
| `emit` | Append consensus-visible event or log output |
| `external_call` | Invoke another contract or chain program through a target-defined call boundary |
| `storage_read` | Observe persistent chain state |
| `storage_write` | Create, replace, or remove persistent chain state |

The spellings are chain-neutral. They do not name EVM, WASM, RISC-V, Move, Solana, Fuel, CosmWasm,
or any one storage or ABI model.

### 2. Reuse the existing effect checker unchanged

The existing direct-call rule remains authoritative. A caller must declare every effect declared by
the callee. A function with only `storage_read` cannot call a function requiring `storage_write`,
`emit`, or `external_call`; a dispatcher may declare their union and call narrower helpers.

The existing `E7101` diagnostic owns malformed, duplicated, empty, and unknown effect declarations.
The existing `E7102` diagnostic owns a missing caller capability. No blockchain-only diagnostic or
parallel authority graph is introduced.

Compiler-scheduled cleanup retains the same rule. If a resource cleanup body declares an on-chain
effect, every edge that actually performs that cleanup must carry the effect. This records the
behavior accurately but does not decide whether a later contract profile permits mutation-capable
cleanup.

### 3. Classify consensus-visible mutation conservatively

The on-chain subset is exactly:

```text
block_context
emit
external_call
storage_read
storage_write
```

The mutation-capable subset is exactly:

```text
emit
external_call
storage_write
```

`emit` is mutation-capable because it changes consensus-visible output even without changing a
storage key. `external_call` is mutation-capable because an unproven callee may mutate state, emit
output, transfer value, or re-enter the caller. A later static-call facility must have a separately
proven operation and contract rather than weakening `external_call` globally.

`block_context` and `storage_read` are read-only with respect to consensus state. They remain
observable authority and therefore are not treated as purity.

`storage_write` does not imply `storage_read`, and neither implies `block_context`. Every granted
authority remains explicit.

### 4. Derive view eligibility instead of adding `@view`

HIR exposes deterministic helpers for the on-chain vocabulary, mutation vocabulary, membership,
canonical mutation filtering, and read-only classification. A contract target may expose a function
as a view only when its canonical effect set has no `emit`, `external_call`, or `storage_write`.
The target must additionally reject host-only authority that it cannot implement deterministically.

This slice adds no source `@view` attribute. Effects remain the single source of truth, so source,
HIR, public-interface hashing, cache invalidation, client generation, and target policy cannot drift
between a role marker and the actual authority set.

### 5. Keep transaction and block values explicit

An effect declaration grants no ambient value. There is no implicit sender, attached value, block
timestamp, storage handle, or contract object. Future APIs must receive compiler-known capability
values such as an owned or borrowed `TxContext` or narrower storage and block-context handles.

The future type contract must state who constructs the value, whether it is copyable or storable,
which borrows may escape, whether mutation is exclusive, and how the target ABI transports it. The
effect checker answers what authority may execute; ownership and borrowing answer which concrete
capability value is used and for how long.

### 6. Keep target and runtime claims closed

Recognizing an effect does not implement it. This decision adds no storage layout, event encoding,
external-call ABI, reentrancy guard, rollback protocol, payable behavior, gas accounting, runtime
symbol, interpreter hook, backend operation, deployment artifact, or supported chain target.

A future contract profile must independently define its admitted effect subset. A deterministic
profile will normally reject host `console`, `filesystem`, `environment`, `process`, `network`,
`clock`, and `random` effects unless it supplies a consensus-safe operation with an exact separate
contract.

## Implementation

`internal/hir/effect.go` owns the new exact effect constants and keeps the complete vocabulary in
lexical order. It also exposes defensive-copy accessors for the on-chain and mutation subsets,
constant-time vocabulary membership over the bounded closed lists, canonical mutation filtering,
and one derived read-only predicate.

The semantic analyzer already routes every `@effects(...)` argument through `hir.ParseEffect`,
canonicalizes accepted values, stores them in typed HIR and semantic EffectFacts, checks direct calls
with `hir.MissingEffects`, and exports public effects through the existing interface descriptor.
Extending the HIR vocabulary therefore activates the established checker without a second semantic
path.

Unit tests prove vocabulary order, exact parsing, classification, defensive read-only derivation,
and canonical mutation filtering. Semantic integration tests prove typed-HIR preservation,
caller-superset acceptance, `E7102` rejection for missing storage-write authority, and `E7101`
rejection for an unknown storage effect.

## Compatibility

The public-interface v3 encoding already contains canonical effect strings. No descriptor shape or
hash algorithm changes. A public callable that adopts one of the new effects changes its existing
interface digest exactly as any other effect change does.

The accepted spellings expand pre-1 source semantics. Release integration must therefore perform an
atomic exact language-revision cut before presenting this slice as part of a released compiler
contract. This prototype PR deliberately does not perform the repository-wide manifest, SDK,
lockfile, previous-revision, cache-vector, and migration cut. The omission is explicit review debt,
not permission to ship the new vocabulary under an unchanged release pin.

MIR shape, MIR version, runtime ABI, standard-library version, target schema, C ABI, C transport,
and diagnostic catalog version remain unchanged because effect metadata already exists and no
runtime operation is added.

## Rejected alternatives

### One broad `chain` effect

Rejected because it cannot distinguish view-compatible reads from storage mutation, event emission,
or reentrant external calls.

### Treat chain access as `network`

Rejected because deterministic contract storage and block context are not host socket authority,
and the distinction is required for target admission.

### Add Solidity-style modifiers

Rejected because modifier expansion creates hidden control flow and a second policy mechanism.
Ordinary functions, explicit parameters, effects, ownership, and later target metadata remain the
preferred composition model.

### Add `@view` immediately

Rejected because it duplicates a fact already derivable from effects and forces conflict,
interface, cache, and tooling rules before a contract ABI exists.

### Treat `external_call` as read-only until proven otherwise

Rejected because that default is unsafe under mutable callees, value transfer, event emission, and
reentrancy. Static calls require an affirmative target proof.

## Deferred work

Contract-module or manifest profiles, exported entry metadata, payable entry semantics, a
compiler-owned `TxContext`, typed storage handles, event declarations, static calls, value transfer,
reentrancy analysis, transaction rollback, gas or fuel budgets, target-specific effect availability,
contract ABI and client generation, and deployable WASM or RISC-V backends remain separate slices.

## Validation and rollback

The focused validation set is the HIR effect unit package and semantic analyzer package. Repository
CI remains responsible for the complete Go suite and compatibility checks after the PR opens.

Rollback removes the five constants, classification helpers, integration tests, and this contract.
No runtime state, database schema, deployed artifact, SDK source, MIR artifact, or migration output
requires recovery.
