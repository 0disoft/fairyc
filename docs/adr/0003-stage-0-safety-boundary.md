# Stage 0 Safety Boundary

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Fairy C needs resource safety and useful borrows without inheriting a general lifetime algebra, partial-move state machine, pinning model, unwind runtime, or resource-aware arena before the first compiler can execute programs. Several individually attractive features combine into a Rust-sized implementation surface when they are admitted together.

## Decision

Stage 0 adopts these boundaries:

1. Resources use whole-value affine ownership. Resource field moves are rejected.
2. Source-level locals require initializers. Partial initialization exists only while building an aggregate expression inside compiler-controlled temporaries.
3. Named aggregates cannot contain safe borrow fields. Inert structs cannot contain resources.
4. Local borrows end at their last proven use. Place overlap is whole-root; field-disjoint and two-phase borrows are deferred.
5. A returned safe borrow derives from one parameter, one region parameter, or static storage. Multiple-origin returns are rejected.
6. Borrow-returning functions cannot be stored as function values, used through dynamic dispatch, or exposed as safe callbacks.
7. `Region` is a compiler-known lexical capability. Plain regions cannot contain values that need drop.
8. Fallible close operations consume ownership or return the resource explicitly in the failure value.
9. Traps do not unwind and do not run cleanup. A foreign `longjmp` or exception cannot cross a Fairy C frame.
10. Captured closures, async or coroutine state, thread capability traits, user-defined interior mutability, self-reference, `MaybeUninit`, unions, packed layout, atomics, and variadics are outside Stage 0.

## Rationale

These rules make ownership a small control-flow analysis rather than a hidden type-level proof language. They also make cleanup elaboration, C code generation, diagnostics, and a MIR interpreter implementable by a small team while preserving an explicit unsafe escape hatch.

The exclusions are not claims that a feature is inherently bad. They are refusals to commit to the machinery that feature would require before executable evidence exists.

## Consequences

- Some valid C and Rust APIs need handles, indices, regions, copying, or unsafe wrappers in Fairy C.
- Safe collection views and callback APIs are less expressive at first.
- Flow analysis can use a finite ownership state and a hidden single-origin relation.
- MIR has no unwind successor and backends receive explicit cleanup only for normal exits.
- Diagnostics must explain the redesign path, not merely reject the program.

## Revisit conditions

Revisit one exclusion at a time only when representative programs cannot express a safe API without pervasive unsafe code, the required analysis and MIR changes are specified, compile-fail and runtime fixtures exist, and the build-loop cost remains within measured budgets.
