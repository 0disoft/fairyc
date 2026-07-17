# Stage 0 MIR Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

MIR is the only executable semantic form shared by the interpreter and backends, but a whole textual MIR snapshot would freeze block numbering, temporary names, cleanup-block layout, and other implementation choices that do not affect Fairy C behavior. The opposite extreme, checking only final program output, would miss incorrect evaluation order, double cleanup, dropped cleanup, surviving `?` syntax, and trap edges that accidentally unwind.

Malformed-MIR mutation tests also need a serializable MIR test-input format. No such format is accepted yet, so pretending that source files can directly exercise every `Vxxxx` rejection would mix source acceptance with compiler-defect testing.

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
10. The first cases cover ordered nested calls, a resource call move, reverse lexical cleanup, `Result` error cleanup, and division trap paths.
11. Malformed-MIR verifier mutation, partial construction, reassignment, Region reset, resource-bearing tagged payloads, and cleanup-suffix identity remain explicit deferrals.

## Rationale

Source-linked events are stable enough to compare across a Go reference compiler, an interpreter, and later backends without turning the first internal graph layout into a compatibility promise. Separate negative verifier data can be added once the project chooses a bounded serializable MIR input that cannot be confused with a public interchange format.

## Consequences

- A future runner must reject a case as setup failure if parsing, resolution, typing, Flow analysis, or MIR verification fails before observation comparison.
- Equivalent cleanup graphs may use different block layouts as long as their source-linked action order and path behavior agree.
- The fixture schema versions repository test data, not a stable MIR exchange ABI.
- No case currently proves an implementation because neither the compiler nor the MIR runner exists.

## Revisit conditions

Version the schema before exposing graph identity, serialized instructions, drop flags, unsafe provenance payloads, interpreter outputs, or backend-specific representation. Add a separate mutation-input contract before claiming coverage of `Vxxxx` rejection paths.
