# ADR 0504: Source state graph helpers and checked call projections

- Status: Accepted for the explicit experimental graph entry point
- Extends: ADRs 0379, 0381 and 0503

The subsequent [ADR 0505](0505-source-state-read-borrows.md) connects root
borrow/view/copy source syntax while retaining helper-owned borrow refusal.

Allow one source `transaction` and up to 32 private, safe, nongeneric, no-argument
void helpers in the same explicit source file. Every helper must be reachable
from the root and the complete call graph must be acyclic. Calls to the root,
foreign/indirect/imported/receiver/value-returning calls, arguments and locals
shadowing a transaction function refuse. Every helper path explicitly returns,
rejects or traps; only the root may commit. The root plus helpers retain the
existing aggregate 1,024-operation budget and per-body 64-block bound.

Source graph blocks may now contain exact helper calls, with canonical cost 1.
Helpers execute in the existing root snapshot/overlay; return resumes the
caller, while rejection/trap rolls back the complete attempt. Helper bodies
remain ordinary state graphs or legacy state bodies, verified independently.
Source and MIR helper borrows stay closed in this step.

## Ordinary owner projection

The explicit source adapter first obtains the normal semantic schema and
function identities from detached declaration/body skeletons. It structurally
admits every original executable statement into the state graph. It then
constructs a separate call-only AST projection in dense graph-block order and
passes that projection through normal name resolution, typing, declared-effects
and Flow analysis. Fresh function identities are rebound by checked names;
caller-owned AST is never modified.

The ordinary HIR is a call identity/effect projection, not an alternative
executable implementation of the storage graph. Graph branches, operands,
storage operations and exits remain owned by the explicit state side table and
its verifier. Native execution uses the admitted graph, never the call-only
ordinary MIR. This does not establish current-source Complete/cache authority.

HIR-to-state lowering may admit graph calls only when their block-order call
sequence exactly matches the current ordinary HIR call projection. All existing
same-package, no-argument/void signature, effect, closed-owner, recursion,
helper-count, reachability and atomic publication checks remain. Stable rebind
also checks the current projection; stale/missing calls cannot restore a body.
Existing linear helper and no-call graph encodings and hashes do not change.

## Borrow boundary

A graph call whose declared transitive effects contain `storage_write`
invalidates all prior local borrow handles for subsequent static use. The
callee may mutate any physical key. A read-only helper preserves them; an
ordinary value copied before a writing call survives. This conservative check
prevents newly admitted graph calls from bypassing existing exact-key borrow
invalidation. Runtime epochs remain an independent check, not a replacement
for static admission. Passing/returning borrows, owning them in helpers and
crossing block edges remain closed.

## Windows evidence

Source tests cover transitive conditional helpers, exact root/shared-overlay
values and costs, rejection rollback, recursion/unreachable helpers, wrong
signatures/arguments, shadowing and missing effects. Boundaries admit 31/32
helpers and refuse 33; combined root/helper operations admit 1,023/1,024 and
refuse 1,025. Stable graph-helper encode/decode/rebind and stale-call refusal
pass; direct MIR tests check read-only/write/copy-before-call borrow boundaries.

The authored `examples/contract_state_graph/helpers/transaction.fy` passes the
actual source compiler/native handoff and Windows low/optimized C framing/CAS.
Twelve executions cover commit, helper rejection and helper trap, twice each
with store close/reopen and independent Go MIR/snapshot comparisons. Each
attempt takes one snapshot and zero/one CAS; the host never replays an attempt.

Linux remains batched. Source borrow syntax, dynamic byte/text state schemas,
Complete/artifact/cache/ordinary-CLI integration and full final acceptance stay
in the queue. No stable language/CLI/SDK ABI, DB/provider, dependency, default
limit, repository-policy or runner changes accompany this private extension.
