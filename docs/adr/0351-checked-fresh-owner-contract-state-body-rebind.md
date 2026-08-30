# ADR 0351: Rebind stable contract-state bodies to fresh HIR owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/stable_rebind.go`
- Payload source: `docs/adr/0350-bounded-stable-contract-state-body-payload.md`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- Depends on: ADR 0345, ADR 0346, ADR 0348, ADR 0350

## Context

ADR 0350 can decode, verify, and hash one stable transaction payload without importing serialized
capability authority. It still returns only stable data. A cache or package-artifact consumer needs a
separate checked transition from source-independent coordinates and logical function descriptors to
the current compilation's build-local `hir.PackageID` and `hir.SymbolID` values.

Binding only by function name would accept signature drift. Binding only by the self-declared hash
would permit a different valid body for the same package coordinate to replace the body selected by
a cache key. Reusing the decoder's temporary capability would also confuse validation work with the
fresh authority owned by the current compilation.

## Decision

1. Add `RebindStableStateBody`. The caller must supply the exact expected ADR 0348 body hash, the
   current typed HIR package, one fresh build-local owner package, and the current map from local
   package IDs to stable coordinates.
2. Reject a zero expected hash and require it to equal the artifact hash before rebind. Independently
   validate the complete payload and recompute its hash through ADR 0350.
3. Require the coordinate map to contain only nonzero coordinates and to be one-to-one. The selected
   local package coordinate must equal the artifact owner coordinate; a valid body cannot be rebound
   into another package or through an ambiguous coordinate mapping.
4. Re-lower the selected package's current source-independent HIR layout and require exact UID, hash,
   ordered fields, codecs, prefixes, and presence to match the payload layout.
5. Resolve each stable transaction by exact logical module/function name in the selected package.
   Require exactly one current function, a fresh non-sentinel symbol, and byte-identical stable
   parameter, result, effect, and drop-glue descriptor bytes. Extra unrelated functions do not enter
   this boundary.
6. Reject two rebound functions sharing one fresh symbol. The artifact's descriptor order remains the
   deterministic transaction order, independent of current numeric symbols.
7. Reconstruct every program again from payload data, create a new private transaction capability,
   attach it to every operation, and re-enter the transaction-MIR verifier. Validation-time
   capabilities are never reused, and repeated rebinds create different authority identities.
8. Return the fresh package ID, expected hash, exact current layout, and verified
   `LoweredTransaction` records. The current HIR state-transaction side table is not an input, so a
   later cache hit need not retain or relower source operation bodies.
9. Keep package-artifact schema changes, memory/disk storage, all-hit reassembly, quarantine, and
   cached execution outside this slice.

## Validation

Tests describe a body under one package/function numbering, remove the fresh HIR transaction side
table, rebind it under different package/function symbols, and preserve deterministic transaction
order, exact layout/hash identity, verifier success, commit/read execution, and per-transaction
capability ownership. Repeating rebind proves capabilities are fresh. An unrelated function with an
unsupported local type proves the binder examines only named transaction owners.

Negative tests reject a missing, wrong, or self-inconsistent expected hash; missing, wrong, empty,
or duplicate coordinates; missing or changed current layouts; missing, duplicate, or signature-
drifted functions; sentinel symbols; and shared fresh symbols. Empty package bodies remain nonnil,
hash-bound, and rebind successfully without a layout or transactions.

## Compatibility

This extends only the compiler-private experimental transaction model. The language remains
`fairyc-language/12`; public-interface v5, package-cache key v4, stable whole-program MIR and package
artifact versions, runtime ABI, SDK, target, lockfile, Atlas, and CLI schemas remain unchanged.

## Consequences

Stable transaction payloads can now enter a fresh compilation without trusting old local numbers,
old capability tokens, current source operation bodies, or names alone. The next slice may attach
these payloads to stable package artifacts and cache records, but must still prove exact package
closure, hit/miss behavior, malformed-entry quarantine, and execution equivalence through the
workspace reassembly path.
