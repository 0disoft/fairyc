# ADR 0496: Explicit copied local closures

- Status: Accepted for bounded Go Stage 0 and Fairy candidate prototypes
- Date: 2026-11-01

```text
fn i32 main() {
    let i32 offset = 40;
    let add = fn [i32 offset] i32(i32 x) { return offset + x; };
    return add(2);
}
```

The new expression retains explicit typed captures, parameters, result and body in CST/AST.
A capture is a named value, copied once when the closure binding is reached. Later assignment
to the original variable does not change the snapshot. Execution admits at most four i32
captures, four i32 parameters, an i32 result and one returned pure arithmetic expression of
at most 64 nodes. Literal, parameter/capture paths, grouping, unary +/- and binary +,-,*,/,%
are the complete body family. Captures cannot be mutated or implicitly inferred.

The initial execution path requires a direct untyped let binding in the function's outer
body and direct positional calls within that local scope. Closure copying, passing, returning,
assignment, storing, type annotations, nested closure declarations and resource cleanup bodies
remain closed. Calls may occur in nested if/while/classic-for scopes; ordinary local shadowing applies.
View/vector for-in combined with an active copied closure remains separately closed.
Captured resources, borrows, raw pointers, asynchronous/retained calls and cross-thread delivery
are excluded. A closure is not an extern-C callback or a public runtime/ABI value.

Lowering structurally lifts one private helper per literal and inserts i32 snapshot locals;
call arguments remain normal ordered MIR arguments. No heap, runtime environment, dynamic call
or new MIR opcode is introduced. Generated helper/snapshot names are compiler-private and
must not become source-callable. Typed-HIR, Flow, checked arithmetic, MIR verification and
ordinary package artifacts retain authority. Generic preparation must detach and substitute
the new AST edges; source hashes include the written body/capture facts and public generic
closure hashes include those facts before cached reuse.

Parser syntax was delivered in `8f3f3401`; execution and the Windows native receipt are
delivered in the commit containing the [bounded expansion receipt](../ops/windows-language-expansion-20261101.md).
The Fairy-authored candidate now has bounded syntax, private typed lifting, written-source
identity and restored-artifact C transport evidence in the [candidate closure architecture](../architecture/candidate-copied-closures-v1.md)
and [Windows receipt](../ops/windows-candidate-copied-closures-20261102.md).
Broader closures, full CLI/cache acceptance and stable language promotion remain outside this slice.
No CLI flag, runtime ABI, dependency, DB or runner changes are made. Linux native checks stay
in the maintainer's batched queue.
