# ADR 0024: Stage 0 loop control and cleanup

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0004-results-traps-and-control-flow.md`
- MIR source: `docs/mir/drop-elaboration-v0.md`
- Fixture sources: `spec/fixtures/semantic/manifest-v0.json`, `spec/fixtures/resource/manifest-v0.json`, `spec/fixtures/result/manifest-v0.json`, `spec/fixtures/mir/manifest-v0.json`, `spec/fixtures/mir-verifier/manifest-v0.json`, and `spec/fixtures/interpreter/manifest-v0.json`

## Context

The grammar and AST already represented `break` and `continue`, but the executable Stage 0 slice
could not type-check their loop context, model their Flow edges, or lower cleanup across a lexical
Region. Treating either statement as ordinary fallthrough would let code after it affect analysis;
jumping directly to a loop target would skip live resource drops and Region reset.

Resource and Result Flow need different facts on these edges. A resource edge must return to the
loop header with the same outer-root state, while every pending Result local leaving the current
scope must be handled. A `continue` edge also participates in the next iteration and can expose an
overwrite that a single body walk misses.

## Decision

1. `break` and `continue` target the nearest enclosing `while`. Either statement without an
   enclosing loop is rejected during type checking as `E1012` and produces no executable MIR.
2. Both statements end the reachable portion of their source block. Name and type analysis may
   still diagnose the statement itself, but later statements do not contribute Flow state to that
   edge.
3. Resource Flow captures normal body fallthrough, `break`, and `continue` states independently.
   Every reachable edge must preserve the loop-header state of resources declared outside the loop;
   a mismatch is `E1105` with declaration, move when present, and loop-header spans.
4. Result Flow requires pending locals in scopes exited by either edge to be handled. Continue
   states feed the repeated-iteration analysis, so producing another Result over a pending
   destination is rejected. Break states contribute to the loop exit and do not become back edges.
5. MIR lowering maintains a nearest-loop stack. `break` targets the loop exit and `continue`
   targets the loop header. Ordinary body fallthrough targets the same header as `continue`.
6. Before any of those gotos, lowering cleans only scopes exited by the edge. It drops resources
   inside the innermost exited Region, emits one `region-reset` carrying exit kind `break` or
   `continue`, then drops remaining loop-body resources in reverse declaration order. Outer-loop
   resources stay live. Normal body fallthrough drops body-local resources before its back edge.
7. The MIR verifier accepts the closed reset exit-kind set `fallthrough`, `return`, `propagate`,
   `break`, and `continue`. A verified break-reset seed must fail with isolated `V1403` if its reset
   is removed.
8. Interpreter fixtures execute break cleanup, continue cleanup across repeated iterations, and
   nearest-target behavior for nested loops. This proves verified-MIR execution only; no native
   backend or linked executable is implied.

## Consequences

- The source-to-interpreter slice now executes the Stage 0 loop-control forms already reserved by
  the grammar without turning unreachable source into a Flow predecessor.
- Resource and Result obligations remain compiler errors instead of being repaired by hidden drop
  flags or implicit Result discard.
- A backend can implement loop control from verified gotos and cleanup operations without
  reconstructing source scopes.
- Traps remain non-unwinding and do not run loop or Region cleanup.

## Rejected alternatives

- Lower the jump before cleanup: leaks source obligations and can leave an active Region at the
  target.
- Treat `continue` as body fallthrough only in Flow: misses explicit edge state and repeated-loop
  Result overwrite.
- Allow labelled or value-producing loop control now: expands syntax, target resolution, and join
  typing beyond the accepted Stage 0 grammar.
- Repair mismatched outer resources with implicit drop flags: weakens the whole-root ownership
  rule instead of diagnosing the source program.
