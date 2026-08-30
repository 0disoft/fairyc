# ADR 0354: Reassemble cached MIR and state bodies under one fresh symbol graph

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/packageartifact/stable_artifact.go`,
  `internal/contractstatemir/stable_rebind.go`, `internal/workspacecompiler/session.go`
- Depends on: ADR 0351, ADR 0352, ADR 0353

## Context

ADR 0353 made each persistent cache hit carry both stable package MIR and its exact contract-state
body. The existing all-hit path still discarded the state member. Rebinding the two members
independently was not sufficient: stable MIR assigned a new deterministic function-symbol namespace,
while state-body rebind selected the current HIR function symbol. Those numbers could differ even
when both artifacts described the same package coordinate and logical function.

## Decision

1. Extend stable MIR reassembly with a process-local result that contains the verified MIR program
   and every coordinate-owned logical function name mapped to its newly assigned symbol. The
   existing MIR-only rebind API remains a wrapper over this stricter result.
2. Add an aligned state-body rebind entry point. It retains the ADR 0351 expected-hash, coordinate,
   current-layout, current-signature, fresh-capability, and verifier checks, then requires every
   cached state transaction's logical function to resolve through the MIR reassembly binding for the
   same package coordinate. A missing or duplicate logical binding rejects combined reassembly.
3. On a persistent all-hit, retain complete compilation artifacts, reassemble ordinary MIR first,
   group its fresh bindings by package coordinate, and rebind every exact cached state body through
   those bindings. Publish the result only when every function-owning package participates and every
   state transaction shares its owning MIR function symbol.
4. Add verified compiler-private state transactions to `workspacecompiler.Result`. Fallback and
   process-local-cache paths lower them from the current HIR. Persistent all-hit paths use only the
   cached body operations after the current exact body identity has been computed for key selection.
   Combined reassembly does not consult the current HIR operation side table, and `Session.Check`
   itself performs no state mutation.
5. Treat combined MIR/state reassembly failure like existing mixed-context reassembly failure:
   abandon the complete all-hit result and run ordinary verified lowering. Do not quarantine an
   individually valid record solely because another record or the current package closure cannot be
   combined with it.
6. Keep package-cache key v4, compiler persistent identity `/2`, stable compilation v1, stable MIR
   v3, stable state-body v1, and disk-record v1 unchanged. Their bytes and validation contracts do
   not change; this decision consumes already admitted data under the exact compiler-build identity.

## Validation

The state rebind tests require explicit aligned symbols and reject a missing MIR binding. A
workspace-level cold/warm regression injects one source-independent state layout and a real
write-then-read transaction into a two-package analysis. The cold path lowers and stores two complete
compilations. An independent warm Session must report two hits, invoke ordinary MIR lowering zero
times, assign the cached transaction exactly the symbol held by `answer::value` in reassembled MIR,
create authority distinct from the cold source-lowered capability, and execute the cached transaction
through the failure-atomic oracle to commit and read `42`. The warm harness computes the expected
body hash and key, then removes the current HIR operation side table before combined reassembly so
the returned program cannot be a second source-lowered copy.

## Compatibility

This is a compiler-private result and reassembly change. Language `fairyc-language/12`, SDK
`0.0.0-stage0.54`, public-interface v5, runtime ABI, CLI schemas, package/cache artifact bytes,
targets, lockfile, and Atlas contracts remain unchanged.

## Consequences

A persistent all-hit now returns ordinary MIR and executable compiler-private state transactions
under one verified fresh function-symbol graph. Source syntax, ordinary `compiler.Result` state
integration, data-dependent transaction control flow, native backend lowering, runtime persistence,
migration, gas, Merkle proofs, reentrancy, chain integration, and deployable targets remain closed.
