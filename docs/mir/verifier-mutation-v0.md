# MIR Verifier Mutation Input v0

- Status: Draft Stage 0 fixture contract
- MIR source: `docs/mir/mir-v0.md`
- Decision source: `docs/adr/0015-mir-verifier-mutation-fixture-contract.md`
- Fixture source: `spec/fixtures/mir-verifier/manifest-v0.json`

## Boundary

This contract tests compiler-defect invariants without defining a general serialized MIR. A fixture names one closed, verified seed and applies one typed mutation to one fixture-local entity label. The labels exist only inside the harness and do not correspond to compiler block, local, temporary, symbol, or source-span identities.

## Runner protocol

For every case, the runner must:

1. construct the named seed from the seed definition owned by the harness;
2. verify the baseline and require no `Vxxxx` result;
3. confirm that the seed exposes exactly the entity labels declared by the manifest;
4. clone the baseline;
5. apply exactly the listed mutation to the listed target;
6. run the same verifier entry point used before interpreter and backend handoff;
7. compare the complete result with the one expected code and primary entity.

A seed-construction failure, baseline verifier failure, missing target, ignored mutation, more than one applied mutation, extra verifier code, or verifier crash is a harness failure. It cannot be counted as successful rejection.

`internal/mirverifyfixture` implements this protocol for every seed and mutation named by the
manifest. It deep-clones each verified baseline before mutation and compares the complete verifier
result rather than searching for an expected code inside a larger cascade. The resource-return seed
carries the verified `seed::Token` definition; removing only that definition isolates `V1001`
before any interpreter or backend can infer its shape. The normal-call seed resolves its own
internal symbol; replacing only that callee with an absent symbol isolates `V1001` before artifact
or backend handoff. The plain-region seed includes enter, place, and reset operations; removing
reset or replacing its closed exit kind isolates `V1403`. The break-region-control seed carries a
`break` reset followed by its loop-exit goto; removing that reset isolates `V1403` before return.
The checked-arithmetic seed has a complete addition check edge; removing its marker isolates
`V1501` at the continuation operation.

## Seed families

- `scalar-return`: initialized scalar return with one reachable terminated block;
- `resource-return`: one empty-drop resource definition and obligation, initialized drop flag, flag clear before drop, and clean return;
- `borrow-return`: one valid parameter origin and bounded loan;
- `plain-region`: inert placement that remains within one plain region;
- `break-region-control`: inert placement, a `break`-kind reset, and a goto to the loop exit;
- `checked-division`: explicit zero and signed-minimum checks with a non-returning trap block;
- `checked-arithmetic`: checked addition with an exact overflow marker, switch, continuation, and trap block;
- `normal-call`: one call with only a normal continuation;
- `normalized-control`: a Result branch after all source `?` syntax has been removed;
- `unsafe-operation`: an unsafe raw-address operation carrying source span and unsafe-scope identity.

The Go harness may represent these seeds however the internal MIR requires, but it must preserve their named semantic obligations and entity-label set. Changing a seed in a way that changes the targeted invariant requires a fixture contract review.

## Mutation rules

Each mutation is closed and typed: remove a resource type definition, remove a terminator, introduce a use before initialization, duplicate a resource discharge, remove cleanup, move a resource projection, remove drop-flag initialization, reorder flag clear after drop, extend a loan, add a return origin, mark a region value escaping, place a `NeedsDrop` value in a plain region, remove a Region reset, corrupt a required division check, remove a checked-arithmetic marker, add an unwind successor, reinsert a source-only form, remove a tagged-operation type, or remove unsafe provenance. A mutation names `supportingEntity` when its target alone would not identify the replacement value or exact check to remove.

Mutation application is deterministic and idempotence is not assumed. Applying the same recipe twice is a harness error rather than a second test case.

## Compatibility

`fairyc.mir-verifier-fixtures/v0` versions test recipes and seed vocabulary only. It does not stabilize internal MIR serialization, permit external backend plugins, make caches migratable, or expose a public compiler API. Arbitrary malformed graphs, mutation combinations, fuzz encoding, and crash-bundle layout remain deferred.
