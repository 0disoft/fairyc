# Stage 0 Resource Flow Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Resource ownership is specified as `Live`, `Dead`, and `MaybeLive`, but a compile-fail filename alone cannot prove when a move occurs, whether a cleanup obligation remains, or which control-flow join introduced path dependence. Reusing the general semantic-fixture schema would either hide these facts or silently broaden an already versioned contract.

## Decision

1. `spec/fixtures/resource/manifest-v0.json` is the machine-readable Stage 0 acceptance index for whole-root resource Flow analysis.
2. Every case is parser-valid, resolves, and type-checks before resource assertions run. `E00xx`, `E01xx`, and `E10xx` results are fixture setup failures unless a future schema explicitly says otherwise.
3. A resource-state assertion identifies the binding declaration, an observation span, whether the observation is immediately before or after that span, the expected `Live`, `Dead`, or `MaybeLive` state, and the complete-root cleanup-obligation count.
4. A complete resource root has exactly one cleanup obligation while `Live` and zero while `Dead`. Source-visible `MaybeLive` is rejected by Flow and cannot be repaired by an implicit user-visible drop flag.
5. Compiler-owned partial construction and backend cleanup flags are not source resource states. Their later MIR fixtures use separate contracts.
6. Compile-fail assertions list the complete independent `E11xx` diagnostic set with a primary byte span and role-labelled declaration, move, owner, join, or loop spans. Full prose and internal Flow block IDs are not frozen.
7. The initial cases cover a whole-value move, a resource that remains live until scope exit, equal dead states across an `if`, use after move, resource-field partial move, a `Live`/`Dead` branch join, and a loop back-edge mismatch.
8. Later v0 cases add live replacement and dead-slot reinstallation. Fallible close, resource-bearing `Option` and `Result`, partial aggregate construction, and concrete drop traces remain MIR or interpreter concerns unless a new source-state assertion requires this Flow schema to change.

## Rationale

The before-or-after observation makes a move transition testable without serializing the entire Flow CFG. Binding and source spans remain stable enough for diagnostics while allowing the compiler to renumber every internal node. Keeping source ownership apart from MIR drop flags prevents an implementation convenience from weakening the Stage 0 rule that path-dependent resource roots are rejected.

## Consequences

- `internal/resourcefixture` completes parser, resolver, and type phases before comparing Flow facts
  across every v0 case.
- A pass case may assert multiple states at one operand span to prove the transition caused by that operand.
- A fail case may expose `MaybeLive` as the rejected computed state, but no later source operation may treat that state as usable.
- All 12 current cases execute against the Go Stage 0 implementation, including mismatched resource
  state on explicit break and continue edges. MIR and interpreter fixtures
  separately prove replacement ordering, consuming-close ownership, tagged payload moves, and drop
  traces without widening this source-state contract.

## Revisit conditions

Version the schema before adding field-level ownership, conditional source drop flags, partial moves, or an assertion that depends on MIR block identity or backend cleanup representation.
