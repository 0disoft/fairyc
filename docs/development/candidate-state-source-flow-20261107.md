# Candidate state source control-flow lowering — 2026-11-07

## Ownership and API

This worker owns only `examples/lexer_candidate/src/module_state_graph_source_flow.fy`
and this receipt. The source driver, context, operand and binding modules, tests,
lockfile, queue and indexes belong to the parent or the other workers.

`flow::build(p, declaration, layouts, helpers, capability, remaining, spans, exit, work)`
uses the requested fixed signature and returns `Result<m::Graph,u32>`.
Declaration indexes are zero based; helper indexes are one based. Every emitted
operation and header uses the supplied root capability, including helper bodies.
Headers use layout 0 and entry 0. `flow::effects(p, syntax::Declaration, span, work)`
is also available for consolidating the driver's attribute extraction.

## Implemented behavior

The implementation follows `internal/compiler/state_graph_source.go`'s
`newBlock`, `block` and `branch`, and `internal/compiler/state_graph_operands.go`'s
statement lowering. It delegates immutable bindings and operands to their
separate worker modules.

- Explicit root commit, reject and literal-category trap; void return only in
  helpers. Statements or a tail expression after a closed path are rejected.
- Structured `if`, a final tail `if`, block arms and an implicit empty else.
  Conditions are grouped ordinary bool references or found-result references,
  as resolved by the operand module. Else-if without an explicit block remains
  closed, matching the Go source admission's block-arm requirement.
- Dense reachable DAG blocks with a 64-block bound. Both successors are created
  before visiting either arm; a join is created only if at least one arm remains
  open. The join retains exactly the original parent binding prefix.
- At each edge, ordinary values and found flags receive fresh, globally unique
  ordinary IDs. Borrow IDs are cleared and never become block parameters.
  Copies and presence bits cross edges; owner-bound borrow handles do not.
- Plain helper-call statements resolve against the supplied sorted helper
  table; arguments and calls to `transaction` are refused. Independent program
  verification owns reachability, recursion and exact transitive effects.
- `remaining` enforces the caller's combined 1,024-operation allowance through
  the shared context emitter. Zero-operation aliases are not incorrectly
  limited to 1,024 statements. The parser's reverse statement links are walked
  once per source block and consumed backwards.
- The shared 16 MiB work ledger is charged before bounded traversal/allocation
  work. Private errors remain 1001 (shape), 1002 (work/size), 1003 (allocation).

Nested source traversal can revisit a false block whose ID precedes newly
allocated true-arm descendants. Raw operations therefore retain a block-owner
tag. Final stable counting scatter packs operations and source spans in block
ID order in O(operations + blocks), preserving within-block source order.
Edge argument slices are separately repacked in block ID order. Empty edge
argument slices retain the verifier's running offset; unused terminator fields
remain zero. Parameters already follow block creation order.

All mutable owning buffers are separate local `Vec`/`Buffer` variables passed
by reference. No mutable resource-field projections are introduced. Returned
graphs are unverified construction results; the program factory must still
verify them before any proof, artifact reuse or execution is authorized.

## Previous flow-worker checks

1. Syntax check with the existing Go bootstrap:

   ```powershell
   # Working directory: examples/lexer_candidate/src
   .cache/fairyc-state-graph-source-integration.exe experimental check-syntax module_state_graph_source_flow.fy
   ```

   Passed. The executable is addressed by its absolute repository path when
   running from that working directory. An earlier invocation from the repository
   root failed its source-relative module-name check; that is not counted as a pass.

2. Isolated dependency-closure check:

   ```powershell
   # Working directory: .cache/state-source-flow-worker-check-20261107
   ../fairyc-state-graph-source-integration.exe check
   ```

   Passed: **2 instances, 68 source files, 982 MIR functions**. The workspace
   copies the flow module and its current import closure; its lockfile and log
   stay in `.cache`. The initial check exposed three parser-sensitive leading
   dereference assignments after `if` expressions. These were replaced by plain
   mutable `Span` field assignments before the passing check.

The parent owns the independent Go oracle and integrated Windows low/optimized
native run, including 32-helper and 64-block cases. Its passing results are
recorded below; the flow worker did not duplicate that command. Linux
execution and allocation-failure injection are deferred. This worker does not
claim current Complete, cache restoration, persistence or transaction execution.

No source-of-truth contract, public CLI/SDK API, DB, dependency, runner policy,
shared source or lockfile changed in this worker scope. No staging, commit,
push, CI dispatch, release or deployment was performed.

## Integrated authored-source validation, 2026-11-07

The parent independently compared the actual source producer and restored
root/helper program with Go: 68 authored cases / 136 Windows low/optimized
processes, package 24.792s. Retained source and artifact regressions bring this
slice to 374 native processes and eight ownership refusals. Fresh bootstrap
`check` and `check --locked` pass 716 files / 8,917 MIR functions; scoped vet
and source hygiene pass. The separate operation-limit suite is not included.

Exact commands, work/ownership controls, failed-attempt accounting and full
integration records are in
[the source receipt](candidate-state-graph-source-20261107.md#integrated-authored-source-validation-2026-11-07).
The earlier module-specific checks above remain historical evidence.
Linux execution and allocator-failure injection remain deferred; ordinary
Complete/package/cache/command/execution admission remains separate.
