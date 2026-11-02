# Fairy candidate copied local closures v1

The Fairy-authored compiler host implements the bounded family in
[ADR 0496](../adr/0496-explicit-copied-local-closures.md). This does not promote
first-class closures, a public closure ABI, or full candidate CLI support.

A unique, untyped outer-body `let` binds a closure with zero to four explicit i32
captures, zero to four i32 parameters, an i32 result, and one pure arithmetic
expression of at most 64 nodes. Captures copy when execution reaches the binding.
Later mutation of the original local does not change the snapshot. Calls remain
direct and positional, including nested if/while/classic-for scopes; ordinary
local shadowing applies. Return and tail-expression bodies are accepted.

`body_copied_closure.fy` retains written syntax and canonical Go AST JSON.
`copied_closure_admission.fy` checks the bounds; `copied_closure_lower.fy` creates
one private helper and immutable snapshot locals. Virtual tokens and an owned
`copied_closure_tables.fy` table identify generated helpers and snapshots without
adding generated names to the source bytes. The name walker resolves these
private coordinates; written helper-prefix names and closure value escapes reject.
Ordinary name/type/Flow/call/IR verification still runs after lifting.

Lifted helpers overlap their source owner's span. `copied_closure_reach.fy` selects
reachable expression, statement and entry arenas for body projection, owned IR
and current-source artifact comparison. Span containment alone cannot select the
executable body. Scalar helper annotations remain concrete inside a generic
source function; the parent's generic type parameters do not defer them.

Original function JSON remains independently owned through copies and source
joins. Source dependency walking and the closed package graph use a fresh written
projection from the owned scanner, before lifting; closure capture/parameter/result
types and its body are traversed without inventing a callable dependency. Public
template and specialization identities therefore include written closure facts.
Whole-project encode/decode/restore rechecks current source before C publication.

Closure copy/pass/return/store, annotations, nested declarations, mutable or
implicit captures, resources/borrows/pointers, labeled calls, impure or oversized
bodies and view/vector for-in with an active closure remain closed. Retained,
asynchronous and cross-thread closures remain excluded.

[Windows evidence](../ops/windows-candidate-copied-closures-20261102.md) covers
private C11 transport and artifact restoration. Candidate C17, interpreter,
fresh-session persistent cache and full CLI acceptance remain separate. Native
Linux is intentionally deferred in the maintainer's combined validation queue.
No new MIR opcode, allocator, runtime ABI, dependency or cache wire version.
