# Drop Elaboration v0

The Fairy candidate's unit 28 carries static ability masks into its owned HIR/MIR.
Implicit drop paths reject recursive no-drop values. Private operation 27 is Burn:
it consumes one authorized direct no-drop root and invokes no user cleanup or field
glue. This candidate transport tag is separate from the Go MIR operation enum.

- Status: Draft Stage 0 contract
- MIR source: `docs/mir/mir-v0.md`
- Language source: `docs/language/0001-values-and-resources.md`

## Input

Drop elaboration consumes verified Flow CFG ownership states, lexical scope-exit sets, aggregate initialization progress, call-argument temporaries, and typed HIR resource definitions with declaration-ordered fields and classified drop bodies. It runs before MIR is handed to an interpreter or backend. A nonempty body is a compiler-private typed function linked from the resource definition; the link cannot disappear or be reconstructed during elaboration.

## Obligations

Every transition to `Live` creates one cleanup obligation. A move, consuming call, successful ownership transfer, authorized burn, or drop removes it. `Dead` has none. A source-visible complete resource root that becomes `MaybeLive` is rejected during Flow analysis. Drop flags are reserved for compiler-owned partial construction or later explicitly specified conditional obligations, not for silently accepting a path-dependent source local.

Drop flags are not attached to every resource automatically. They are introduced only for genuinely conditional obligations.

## Cleanup edges

The compiler first creates a direct cleanup chain for each normal exiting edge. The chain drops live resources in reverse lexical declaration order and resets exited regions after their inner resources. Correctness is verified before identical suffixes are merged.

`return`, `break`, `continue`, and ordinary fallthrough clean only scopes actually exited by the edge. A trap and `Unreachable` have no cleanup successor.

## Drop operation

When a flagged resource is dropped, MIR clears the flag before invoking the drop glue. This prevents reentrant code or an internal failure path from observing a second live obligation. Stage 0 `drop` is infallible and cannot unwind. The drop receiver is a non-owning internal execution view of the same resource root, even though the current private callable ABI transports it by value.

The verified MIR program owns one resource definition per canonical resource type. Construction,
field reads, interpreter cleanup, and future backends consume that definition rather than inferring
field order or drop behavior from operations. For a nonempty definition the interpreter records the
resource drop, enters `dropping`, invokes the linked private body, then recursively drops fields in
reverse declaration order. A second drop during `dropping` is rejected.

## Burn operation

Language 12 lowers an authorized direct no-drop local or parameter to a distinct MIR `Burn`
operation. Burn consumes one live root and clears its ownership obligation without invoking its drop
body or recursively dropping fields. `V1104` requires a direct local operand with the exact resource
type; `V1101` rejects a second discharge. The semantic boundary admits only declaration-owned inert
resources, so MIR never reconstructs burn authority and backends must not lower Burn as Drop.

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

## Fairy candidate unit 26

The candidate retains nested resource fields and lowers their complete-root
structural cleanup to typed operation 26. Its verifier consumes constructor
field donors exactly once and rejects by-value resource field extraction.
The C transport invokes the exact retained outer private body, then recursively
cleans declaration-order fields in reverse. Partially completed literal fields
and call arguments remain separate temporary obligations; propagation cleans
these in reverse completion order without calling the unfinished outer body.
The candidate has a 4096-local per-function bound in addition to its existing
16384-local program arena. No partial-construction flags are exposed to source.

Unit 27 extends the candidate's typed structural drop to owned Option/Result.
Construction transfers payload donors; payload extraction discharges the complete
subject and produces the selected payload. Ignored captures retain an arm-scope
temporary obligation. Scope cleanup merges these temporary obligations with roots
in reverse lexical source order, walking inner scopes first.
