# ADR 0019: Stage 0 Result must-use dataflow

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Semantic source: `docs/language/0004-results-traps-and-control-flow.md`
- Fixture source: `spec/fixtures/result/manifest-v0.json`

## Context

The first Result Flow slice rejected a bare Result expression and classified direct return,
discard, match, and propagation. That was not enough once a Result could be stored in a local,
passed to another function, overwritten, handled in only one branch, or carried around a loop. A
statement-only scan could report the obvious typo while allowing the same unhandled value through a
different control-flow shape.

## Decision

Every evaluated `Result<T, E>` has one must-use obligation. The obligation is discharged by return,
`discard`, exhaustive Result `match`, `?`, or by-value transfer into another Result local or
parameter. Storage transfers the obligation to the destination; it does not count as terminal
handling. A Result parameter begins its function with one pending obligation.

Flow rejects `E4101` at the producing expression when a pending obligation reaches a normal scope
or function exit, an earlier return edge, or an overwrite. A branch join remains pending when any
reachable incoming path is pending. A loop back edge cannot overwrite an obligation left pending by
an earlier iteration; zero-iteration and repeated-iteration paths are both part of the analysis.

Must-use is independent of copyability and resource ownership. Copyable Result payloads do not
permit an evaluated Result to be silently ignored, while resource-bearing Results still obey the
separate whole-root move and active-payload rules. A later evaluation of a copyable Result is a new
use and must itself appear in a handling or transfer position.

The v0 Result fixture schema is unchanged. Its case set now covers direct and stored handling,
parameter transfer, branch-produced locals, all-branch handling, partial-branch failure, pending
parameter and local exits, overwrite failure, repeated-loop overwrite, and early-return failure.

## Consequences

- `experimental check-semantics` can reject stored and path-dependent unused Results rather than
  claiming only direct expression coverage.
- Function callees own the obligation transferred into Result parameters; callers cannot launder an
  ignored Result through a helper whose parameter is never handled.
- Result handling facts remain source-linked terminal observations. Internal pending-state identity,
  block numbering, and iteration strategy are not public schema fields.
- Cross-package summary reuse remains deferred until public package interfaces exist; whole-source
  analysis still applies the same rule across every supplied module.

## Rejected alternatives

- Warn only on expression statements: misses stored, overwritten, branch-dependent, and parameter
  obligations.
- Treat assignment as handling: permits silently overwriting the previous failure value.
- Require every Result to be move-only: conflates must-use policy with payload copyability.
- Add an implicit drop at scope exit: makes ignored failures look successfully handled.
