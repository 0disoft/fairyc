# Stage 0 MIR Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

MIR is the only executable semantic form shared by the interpreter and backends, but a whole textual MIR snapshot would freeze block numbering, temporary names, cleanup-block layout, and other implementation choices that do not affect Fairy C behavior. The opposite extreme, checking only final program output, would miss incorrect evaluation order, double cleanup, dropped cleanup, surviving `?` syntax, and trap edges that accidentally unwind.

Valid source cannot directly exercise missing terminators, invalid drop flags, unwind successors, or absent unsafe provenance. Those compiler-defect paths are isolated in the separate named-seed mutation contract rather than being mixed into source acceptance or forcing a general serialized MIR format.

## Decision

1. `spec/fixtures/mir/manifest-v0.json` is the machine-readable Stage 0 acceptance index for source-linked MIR normalization facts.
2. Every case is parser-, resolver-, type-, and Flow-valid, then lowers through MIR verification. The initial manifest contains source-pass cases only.
3. Assertions use canonical function names and half-open UTF-8 source-byte spans. They do not expose MIR block numbers, local numbers, temporary names, drop-flag slots, or generated C.
4. Ordered-evaluation assertions freeze the left-to-right sequence and require side-effecting children to be materialized before their parent operation.
5. Call assertions require a normal successor and forbid an unwind successor. Move assertions require explicit cleanup-obligation transfer.
6. Cleanup-path assertions freeze only the observable action order attached to a source exit. An empty action list is meaningful after a resource was moved away.
7. Result-branch assertions require exactly-once evaluation, moved success and error payloads, a dead input temporary on both branches, and ordered cleanup on the propagated-error path.
8. Trap-path assertions use canonical semantic categories, have no successor, and contain no cleanup action. Normal completion of the same checked operation remains a separate cleanup path.
9. Eliminated-form assertions prove that source-only `?`, implicit drop, nested side effects, or short-circuit syntax does not survive verified MIR.
10. The first cases cover ordered nested calls, a resource call move, reverse lexical cleanup, `Result` error cleanup, and division trap paths. Later v0 cases add source-linked ownership transfer through a resource-bearing `Option` match and RHS-before-drop replacement of live or dead resource destinations.
11. Partial construction, Region `break` and `continue`, backend variant-layout identity, and cleanup-suffix identity remain explicit deferrals. Region fallthrough, direct return, Result propagation, nested cleanup, and reset ordering are source-linked observations. Verifier mutation coverage belongs to `spec/fixtures/mir-verifier/manifest-v0.json`.

## Rationale

Source-linked events are stable enough to compare across a Go reference compiler, an interpreter, and later backends without turning the first internal graph layout into a compatibility promise. Separate named-seed mutation recipes cover verifier defects while keeping arbitrary MIR serialization outside this contract.

## Consequences

- A future runner must reject a case as setup failure if parsing, resolution, typing, Flow analysis, or MIR verification fails before observation comparison.
- Equivalent cleanup graphs may use different block layouts as long as their source-linked action order and path behavior agree.
- The fixture schema versions repository test data, not a stable MIR exchange ABI.
- The current Go runner proves only the listed source-linked subset; it does not prove a backend.

## Revisit conditions

Version the schema before exposing graph identity, serialized instructions, drop flags, unsafe provenance payloads, interpreter outputs, or backend-specific representation. Verifier mutation recipes evolve under their separate schema and do not broaden this source-observation contract.
