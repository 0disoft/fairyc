# ADR 0337: Experimental target-neutral on-chain effect capabilities

- Status: Accepted experimental slice
- Date: 2026-08-20
- Product source: `docs/product/02-spec.md`
- Language contract: `docs/language/0008-effects-and-capabilities-v0.md`
- Depends on: ADR 0003, ADR 0116, ADR 0121, ADR 0218, ADR 0328, ADR 0336

## Context

Fairy C already has source-visible effects, canonical HIR effect sets, caller-superset checking, and
Resource Flow checks for compiler-scheduled cleanup. A future contract profile needs to distinguish
state reads, state writes, event emission, external calls, and deterministic chain context without
adding Solidity-style modifiers, ambient globals, or a second authority checker.

One broad `chain` or reused `network` effect would erase distinctions needed for review, view
classification, capability reduction, reentrancy policy, and target admission. A separate `@view`
marker would duplicate facts already derivable from effects and allow contradictory declarations.

## Decision

The experimental compiler vocabulary adds five exact target-neutral effects:

| Effect | Authority represented |
| --- | --- |
| `block_context` | Observe target-supplied block, transaction, caller, value, height, timestamp, or chain context through an admitted explicit API |
| `emit` | Append consensus-visible event or log output |
| `external_call` | Invoke another contract or chain program through a target-defined call boundary |
| `storage_read` | Observe persistent chain state |
| `storage_write` | Create, replace, or remove persistent chain state |

The existing caller-superset and cleanup-effect rules apply unchanged. `E7101` owns malformed or
unknown declarations and `E7102` owns missing caller authority. Effects grant no ambient sender,
value, block data, storage handle, or contract object; future APIs must transport explicit
compiler-known capability values.

`emit`, `external_call`, and `storage_write` are mutation-capable. Event emission changes
consensus-visible output. An unproven external call may mutate, emit, transfer value, or re-enter,
so only a separately proven future static-call operation may be classified more narrowly.
`block_context` and `storage_read` are read-only with respect to consensus state but remain
observable authority. No effect implies another.

View eligibility is derived from the canonical effect set. This slice adds no `@view`, `@entry`,
payable rule, implicit context, storage API, event encoding, external-call ABI, rollback protocol,
gas schedule, runtime symbol, backend operation, deployment artifact, or supported chain target.

## Compatibility and admission boundary

`Stage0Effects()` remains the exact released `fairyc-language/10` vocabulary. The complete new set is
available through compiler-internal experimental helpers and direct single-package semantic analysis
only. Stable resolved-package analysis reports an on-chain effect as unsupported, so workspace check,
build, run, test, cache, and native emission cannot silently acquire the new semantics.

Public-interface v3 can already encode canonical effect strings, but stable package admission stays
closed. A public callable that later adopts a new effect will change its existing interface digest.
Release requires one exact language-revision cut with manifests, SDK pins, locks, previous-revision
handling, cache vectors, fixtures, and migration behavior. This ADR changes no current SDK, runtime
ABI, MIR version, target schema, C ABI, or cache identity.

## Consequences

Future contract work gains one authority substrate shared by semantic analysis, HIR, public-interface
hashing, and target policy. Host effects such as filesystem, process, network, environment, clock,
and random remain distinct and are normally rejected by deterministic contract profiles.

The remaining work is deliberately larger than vocabulary admission: explicit context ownership,
typed persistent state, event and call transports, reentrancy analysis, transaction rollback,
metering, target-specific effect availability, client metadata, and deployable targets each require
separate evidence-backed slices.
