# Drop Elaboration v0

- Status: Draft Stage 0 contract
- MIR source: `docs/mir/mir-v0.md`
- Language source: `docs/language/0001-values-and-resources.md`

## Input

Drop elaboration consumes verified Flow CFG ownership states, lexical scope-exit sets, aggregate initialization progress, and call-argument temporaries. It runs before MIR is handed to an interpreter or backend.

## Obligations

Every transition to `Live` creates one cleanup obligation. A move, consuming call, successful ownership transfer, or drop removes it. `Dead` has none. A source-visible complete resource root that becomes `MaybeLive` is rejected during Flow analysis. Drop flags are reserved for compiler-owned partial construction or later explicitly specified conditional obligations, not for silently accepting a path-dependent source local.

Drop flags are not attached to every resource automatically. They are introduced only for genuinely conditional obligations.

## Cleanup edges

The compiler first creates a direct cleanup chain for each normal exiting edge. The chain drops live resources in reverse lexical declaration order and resets exited regions after their inner resources. Correctness is verified before identical suffixes are merged.

`return`, `break`, `continue`, and ordinary fallthrough clean only scopes actually exited by the edge. A trap and `Unreachable` have no cleanup successor.

## Drop operation

When a flagged resource is dropped, MIR clears the flag before invoking the drop glue. This prevents reentrant code or an internal failure path from observing a second live obligation. Stage 0 `drop` is infallible and cannot unwind.

## Partial aggregate initialization

Source aggregate literals evaluate fields from left to right into compiler-controlled storage. If field construction fails, only fields that completed initialization are dropped, in reverse completion order. The outer aggregate's drop glue is not invoked because the aggregate never became a live value.

Partial resource-field moves after construction remain forbidden and therefore require no field-level move flags.

## Reassignment

Resource reassignment uses this order:

1. evaluate the complete right-hand side into a temporary;
2. if evaluation propagates an error, clean the temporary path and leave the old resource live;
3. on success, drop the old resource and clear its obligation;
4. move the temporary into the destination;
5. mark the destination live and the temporary dead.

Reassignment when the old destination is `MaybeLive` is rejected before elaboration.

## Result propagation

The error branch of `?` moves the error payload into the return value, then traverses the normal cleanup chain for every exited scope. The success branch moves the success payload to its continuation. The original result temporary is dead on both branches.

## Verification cases

`spec/fixtures/mir/manifest-v0.json` now covers a resource call move, reverse lexical cleanup, `Result` error-path cleanup, resource-bearing tagged moves, live/dead reassignment ordering, Region fallthrough and scope-indexed cleanup/reset on direct return, Result propagation, nested exits, and nearest-loop `break` and `continue`, plus the absence of cleanup on a trap path. Loop-control cleanup drops inner Region resources, resets each exited Region with the edge's exit kind, then drops remaining body-local resources before the goto. Interpreter fixtures additionally prove those Region paths, repeated continue execution, nearest nested targets, successful and propagated-error reassignment traces, and consuming-close success and failure ownership. Failed aggregate fields, resource arguments evaluated before a failing later argument, and cleanup-suffix merging remain explicit deferred groups.
