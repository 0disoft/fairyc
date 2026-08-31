# ADR 0347: Lower source-independent HIR state access through verified transaction MIR

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- HIR source: `internal/hir/state.go`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- State source: `docs/contract-state/transaction-kernel-v0.md`
- Depends on: ADR 0338, ADR 0344, ADR 0345, ADR 0346

## Context

Package-owned state layouts, opaque transaction identity, and storage-effect verification existed,
but no typed HIR operation crossed all three boundaries. Testing transaction MIR directly could not
prove that a future HIR producer would resolve the owning function, preserve its declared effects,
or bind a field name to the exact package layout before execution.

Source syntax is still premature. Ordinary function bodies, stable package artifacts, and cache
records do not yet carry a state-access body identity, so admitting source operations now would make
incremental reuse and compatibility claims outrun the transport.

## Decision

1. Add source-independent `hir.StateTransaction`, `hir.StateAccess`, and `hir.StateValue` records to
   `hir.Package`. A transaction names its build-local package and one existing ordinary HIR function
   symbol. No Stage 0 source form constructs these records.
2. Keep the HIR function as the sole effect authority. The side table carries no duplicate effect
   set; lowering copies the function's canonical declared effects into transaction MIR.
3. Name persistent fields in HIR by declaration name. Lowering resolves each name through the exact
   package-owned `StateLayout` and emits the canonical field index. Missing layouts, functions,
   duplicate transaction identities, duplicate function symbols, and unknown fields fail closed.
4. Keep values codec-explicit and data-independent. Lowering clones their encoded bytes, maps the six
   closed cell/map access kinds plus the three static exits, creates one fresh opaque capability per
   transaction, and attaches it to every operation.
5. Run the ADR 0346 verifier before publishing any lowered program. Missing or noncanonical effects,
   host effects, field-kind drift, noncanonical encodings, and invalid exits therefore cannot reach
   the semantic oracle.
6. Sort output by package and function symbol. Separate transactions receive distinct capability
   identities; mutable HIR byte slices cannot alias the lowered program.
7. Keep this bridge process-local. Public-interface v5 continues to encode only layout identity, and
   package-cache key v3 does not claim to key source-independent transaction bodies. Stable MIR,
   package body artifacts, ordinary compiler execution, SDK, runtime, backend, and target identities
   remain unchanged.

## Validation

Unit tests lower out-of-order package/function transactions, prove all three exits, distinct and
operation-consistent capabilities, byte-slice isolation, exact field-name resolution, canonical
effect preservation, and HIR-to-MIR-to-oracle read/write execution. Focused negative cases reject
missing owners, layouts, functions and symbols; duplicate functions and transactions; unknown fields
and kinds; missing, host, or noncanonical effects; wrong field kinds; malformed values; and invalid
exits before publication. A public-interface regression proves transaction-body changes leave
public-interface v5 unchanged while the separately encoded layout identity remains authoritative.

## Compatibility

This is a compiler-private in-memory HIR and lowering cut. The language remains
`fairyc-language/12`; public-interface v5, package-cache key v3, stable MIR, runtime ABI, SDK,
target, lockfile, Atlas, and CLI bytes remain unchanged.

## Consequences

ADR 0357 later requires every current transaction owner body to be call-free during direct lowering
and stable fresh-owner rebind. That boundary prevents ordinary callee execution from diverging from
the straight-line state side model without claiming transitive call-graph or reentrancy support. ADR
0358 later rejects dynamic control flow in those same current-owner paths. ADR 0359 later excludes
unsafe code and reachable raw-pointer types without treating safe borrows as state permissions. ADR
0360 later excludes receiver-bound methods and automatic drop glue while retaining static associated
functions as explicit entries. ADR 0361 later excludes ordinary potentially trapping operations
without removing the state program's explicit static trap exit.

The compiler now has a complete process-local typed path from package/function ownership through
verified state effects and capability flow into failure-atomic oracle execution. Before ordinary
source or cached compilation can consume it, a later slice must give source-independent transaction
bodies an exact package-body/artifact identity and cache invalidation contract. Data-dependent values,
branches, calls, owner-bound borrows, external-call exclusion, deterministic backend execution, and
production persistence remain separate gates.

ADR 0348 later supplies the exact package-body hash portion of this gate. Cache-key participation
and stable payload transport remain open.
