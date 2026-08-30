# ADR 0350: Define a bounded stable contract-state body payload

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/stable_payload.go`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- Depends on: ADR 0346, ADR 0347, ADR 0348, ADR 0349

## Context

ADR 0348 gives each verified contract-state package body an exact stable hash, and ADR 0349 makes
whole-package cache key v4 consume that hash. Neither decision transports the body itself. A future
cache hit therefore cannot reconstruct the state layout and transaction operations without trusting
process-local HIR again.

The transport boundary must not serialize the opaque transaction capability created by ADR 0346.
That token is live process authority, not data. The decoder also needs an explicit byte budget and
an independent verifier path; accepting arbitrary JSON into executable compiler state would turn a
cache artifact into an authority-forging or resource-exhaustion boundary.

## Decision

1. Define the exact envelope schema `fairyc.contract-state-package-body-artifact/v1` and version
   `fairyc-contract-state-package-body-artifact/1`. Unknown schema or version values are
   incompatible misses, not migrations.
2. Encode one package coordinate, its exact ADR 0348 body hash, an optional canonical layout, and a
   function-sorted transaction array. Each transaction carries the existing stable binary function
   descriptor, canonical effects, exact ordered operations and bytes, and explicit exit/trap data.
3. Reuse the ADR 0348 function-descriptor encoder for both hashing and transport. Decode parses every
   length-delimited descriptor field, validates logical names, byte-exact canonical
   coordinate-stable types and effects, rejects trailing bytes, and requires byte-identical
   canonical re-encoding.
4. Exclude build-local package/function identities, spans, mutable slice identity, and capability
   tokens. Package/function renumbering with the same stable coordinate produces identical payload
   bytes.
5. Require every encode and decode call to supply a positive maximum byte count. Decode rejects an
   oversized or empty artifact before JSON allocation, disallows unknown fields and trailing values,
   and accepts only the byte-exact canonical JSON produced by the encoder.
6. Rebuild the canonical layout from declarations, create a fresh private capability for each
   transaction, attach it to every reconstructed operation, and run the existing MIR verifier.
   Serialized bytes can describe work but can never supply execution authority.
7. Require transactions to be sorted and unique by stable function descriptor, then recompute the
   ADR 0348 body hash from the reconstructed verified programs. Any identity, layout, signature,
   effect, operation, value, order, or exit drift fails closed.
8. Keep this slice inside `internal/contractstatemir`. It does not yet expose a public rebind API,
   extend stable package/MIR artifact schemas, store the payload in memory or disk caches, or execute
   a cached body.

## Validation

Unit tests freeze the complete canonical payload SHA-256, round-trip and byte-identically re-encode
it, match its body hash to ADR 0348, and preserve all bytes across build-local package/function
renumbering. They prove capability and local symbol text are absent, enforce encode/decode limits,
reject noncanonical whitespace, unknown fields, wrong versions and trailing values, and cover empty
package bodies. Independent mutations reject missing identity, nil arrays, malformed layouts,
duplicate or malformed function descriptors, effect mismatch, unknown operations, noncanonical
values, and invalid exit metadata. The complete Go suite includes the Windows host smoke.

## Compatibility

This adds a compiler-private experimental artifact. The language remains `fairyc-language/12`;
public-interface v5, package-cache key v4, stable whole-program MIR and package artifact versions,
runtime ABI, SDK, target, lockfile, Atlas, and CLI schemas remain unchanged.

## Consequences

Contract-state bodies now have a deterministic, bounded, capability-free transport description and
an independent verifier re-entry path. The next slice may expose a checked rebind boundary that maps
stable function identities into fresh HIR/MIR owners. Package-artifact and cache integration must
remain separate until reassembly proves exact provider matching, fresh identity allocation,
quarantine on malformed payloads, and execution equivalence.

ADR 0351 later completes the checked fresh-HIR-owner rebind, ADR 0352 adds the outer package
artifact, and ADR 0353 admits it to persistent cache records. Combined cached state-body reassembly
and explicit oracle execution arrive later in ADR 0354.
