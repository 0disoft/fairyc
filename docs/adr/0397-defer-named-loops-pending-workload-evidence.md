# ADR 0397: Defer named loops pending workload evidence

- Status: Deferred
- Date: 2026-09-22
- HEAD: `d203cbca4b7a96a18f6ad39053c14e0989080ed0`

## Decision

Named-loop break/continue (loop labels) are deferred: no implementation and no normative grammar, language, or SDK pin change.
Scoped inspection of the actual nested loops in the Fairy-written `lexer_candidate` sources found no compelling multi-level break/continue need there; this does not claim no such need exists anywhere.

## Evidence

| source | function | nested-loop behavior today |
|---|---|---|
| [scanner.fy](../../examples/lexer_candidate/src/scanner.fy) | `keyword` | early `return true` on match |
| [scanner.fy](../../examples/lexer_candidate/src/scanner.fy) | `symbol_width` | must inspect all candidates for the longest match |
| [main.fy](../../examples/lexer_candidate/src/main.fy) | `main` | inner frame-input failure returns the whole function |
| [numbers.fy](../../examples/lexer_candidate/src/numbers.fy) | `integer` | nested digit loops return errors; carry propagation must finish |
| [diagnostics.fy](../../examples/lexer_candidate/src/diagnostics.fy) | `report` | inner loop selects the minimum and must complete |
| [bodies.fy](../../examples/lexer_candidate/src/bodies.fy) | `expression_inner` | constructor-field inner `break` must still consume `}` and append the record node before the outer `continue` |
| [attributes.fy](../../examples/lexer_candidate/src/attributes.fy) | `read` | args inner `break` must still consume `)` |
| [integrity.fy](../../examples/lexer_candidate/src/integrity.fy) | `valid` | generic-link loop validates all links, `return false` on failure |

Optional `rg` scope note: no `var bool` found/done/stop/matched flags in the candidate sources; that is not proof of global absence.

## Current cleanup order (unchanged)

`lowerLoopControl` in [lower.go](../../internal/mir/lower.go) calls `emitDeferredCallsForLoopExit` (reverse registered selected calls, see [ADR 0217](0217-bounded-lexical-defer.md)) before `emitLoopControlCleanup`: drop resources within the innermost exited region, then Region reset walking outward, then remaining target-body drops. Outer scopes are preserved; traps do not unwind; `continue` targets the while header but the for post-step once; `break` skips the for post-step.

## Conditional future constraints (not accepted syntax)

If named loops are ever proposed, all of the following are conditions, not accepted syntax: lexical `while`/`for` labels in a label namespace separate from value names; reject duplicate active labels but permit reuse in disjoint scopes; reject unknown, out-of-scope, non-loop, and cross-function targets; no `goto`, block labels, or value-carrying `break`.
Target identity must resolve consistently across typecheck, resource, Result, borrow flow, and MIR, not just a `lowerLoopControl` nearest-loop patch.
Labelled loops are currently absent: nearest-loop control is [ADR 0024](0024-stage-0-loop-control-and-cleanup.md) and scalar `for` is [ADR 0213](0213-classic-scalar-for-loop.md); neither is changed by this deferral.

## Future fixture matrix (conditional)

2/3-level nested `while`/`for` mixes with `break`+`continue`; exactly-once LIFO defer/drops/Region reset; outer resource survives; exited borrows end and pending Results are rejected; outer-resource mismatch `E1105`; a post counter distinguishes `continue` from `break`; label error and shadow-namespace cases; old unlabeled loops unchanged; interpreter plus MIR verify plus Windows low/O2 native. C timing makes no automatic zero-cost claim under the 8% cumulative policy.

## Reopening gate

Reopen only with a real checked-in before/after workload showing named loops remove an outer-exit flag without skipping required work; otherwise helpers, `return`, and current loops suffice.

## Scope

Docs-only: no implementation, no tests run. Product scope stays in [02-spec.md](../product/02-spec.md); validation names stay in [VALIDATION.md](../../VALIDATION.md); index in [README.md](README.md). Prior label work is separate: [ADR 0396](0396-position-preserving-argument-labels.md).
