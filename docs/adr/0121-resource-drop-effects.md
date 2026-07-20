# ADR 0121: Declare and propagate resource drop effects

- Status: Accepted
- Date: 2026-07-21
- Language source: `docs/language/0008-effects-and-capabilities-v0.md`
- Ownership source: `docs/language/0001-values-and-resources.md`
- MIR source: `docs/mir/drop-elaboration-v0.md`

## Context

ADR 0120 made a nonempty resource drop body executable, but deliberately limited that body to the
empty effect set. That prevented a safe owning `File` from releasing a host handle in compiler-
scheduled cleanup: the destructor could not declare `filesystem`, and an enclosing function could
not be checked for authority it exercised only through an implicit drop.

Treating every use of an effect-bearing resource as effectful would be incorrect. A function that
moves the complete resource to its caller performs no cleanup. The requirement belongs to each
actual cleanup edge, not to the type merely appearing in a signature or expression.

## Decision

1. A resource with a nonempty drop body may declare one canonical `@effects(...)` set. The set is
   copied to its compiler-private drop function and preserved on the HIR and MIR resource
   definition. An empty drop body cannot declare effects.
2. Direct calls inside the drop body use the ordinary caller-superset rule. The declared resource
   effects therefore bound every external capability used by its private glue.
3. Resource Flow checks every compiler-scheduled cleanup edge. Normal scope and function exit,
   explicit return, live-value reassignment, and loop `break` or `continue` require the enclosing
   function to contain the resource drop-effect set.
4. A complete-root move clears the cleanup obligation before the effect check. Passing or returning
   ownership therefore does not charge the sender for cleanup it does not perform.
5. `Option<R>` and `Result<R, E>` require the union of effects for resource-bearing payloads. The
   active runtime variant still determines which destructor executes.
6. `E7102` identifies both an effectful direct call and an implicit cleanup whose required effects
   are absent. Its related span points to the callable or resource effect declaration.
7. The MIR verifier rejects a noncanonical resource effect set or private glue whose effects differ
   from its resource definition.

## Consequences

- Effect-bearing resource ownership is now expressible without making moves spuriously effectful.
- A future safe Windows `File` can use infallible filesystem-bearing drop glue, while fallible
  explicit `close(self)` remains a separate typed `Result` operation.
- This decision authorizes the effect contract only. It does not itself add Windows file symbols,
  host I/O to the MIR interpreter, or a safe File API.

## Verification

- semantic tests cover normal exit, early return, nested scope exit, loop exit, reassignment, and
  moved-return exemption;
- semantic fixtures preserve resource effect facts and reject a missing implicit-drop effect;
- HIR-to-MIR lowering preserves the declared set, and MIR verification compares it to private glue;
- the full Go suite remains the platform-neutral oracle; native host behavior is verified only by
  the local Windows/MSVC lane, without WSL.
