# ADR 0345: Carry contract-state layout identity through HIR, interfaces, and caches

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- State source: `docs/contract-state/transaction-kernel-v0.md`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- Interface source: `docs/package/public-interface-hash-v5.md`
- Cache source: `docs/package/whole-package-cache-v3.md`
- Depends on: ADR 0174, ADR 0328, ADR 0338, ADR 0344

## Context

ADR 0344 gave the compiler an exact transaction-MIR layout, but no typed package representation could
own it. Adding source grammar next would still leave package interfaces and cache reuse unaware of a
layout change.

Dependency-only invalidation is insufficient. A source-independent HIR layout may be injected by a
future profile or generator without changing the current package's source digest, so the owning
package cache key must consume its own interface identity as well.

## Decision

1. Add source-independent `hir.StateLayout` and ordered `hir.StateField` values to `hir.Package`.
   Each layout names its build-local owner package, contract UID, cell-or-map kind, and canonical key
   and value codec identities. No source form constructs these values yet.
2. Add `contractstatemir.LowerHIRLayouts`. It sorts by build-local package ID, permits at most one
   layout per package, and lowers through the canonical ADR 0344 layout builder.
3. Cut public-interface hash v5. A package interface carries its contract UID and full 32-byte layout
   hash before ordinary module records. Packages without state encode an empty layout list.
4. Preserve build-local package renumbering invariance: package numbers select ownership but do not
   enter the state layout hash or public-interface bytes.
5. Cut whole-package cache key v3. The key adds the current package's complete public-interface hash
   after its source digest, while retaining sorted coordinate-plus-interface dependency entries.
6. Workspace cache assembly must supply the exact interface computed for every function-owning
   package. Missing current-interface metadata fails closed before cache lookup.
7. Keep Stage 0 grammar, semantic source analysis, storage accesses, stable whole-program MIR,
   package MIR artifacts, interpreter, C backend, target runtime, SDK, and runtime ABI unchanged.

## Validation

Unit and fixture tests prove:

- HIR layouts lower in package order to the same canonical hash as transaction MIR;
- duplicate package layouts and unknown field kinds fail closed;
- codec or field changes alter public-interface v5;
- build-local owner renumbering preserves the interface;
- the v5 fixture injects one source-independent HIR layout, freezes its exact interface bytes, and
  retains all prior source same/different relations;
- current-package and dependency-interface changes independently alter cache key v3;
- the v7 cache fixture freezes one complete v3 key and all compatibility inputs remain mandatory.

## Compatibility

Public-interface v5 replaces v4 for current consumers; `fyif4` remains historical and fails parsing
under the v5 consumer. Package-cache key v3 replaces v2; old `fycache2` keys are misses and are
collected lazily. Cache record storage remains disposable exact-version compiler state.

The language remains `fairyc-language/12`; stable MIR, runtime ABI, SDK identity, target identities,
lockfile, Atlas formats, and CLI bytes are unchanged.

## Consequences

The owning package and every dependent package now invalidate on an exact state layout change even
when the layout entered after source acquisition. Grammar can remain closed while the compiler
proves package identity.

State access expressions, `storage_read` and `storage_write` checking, transaction capability flow,
stable MIR integration, interpreter execution, deterministic backend transitions, and source
diagnostics remain separate gates.
