# Scalar vector iteration connection, 2026-11-07

Follow-up to `efc51df3e1160f9a965ec72511fce6f912ec8196` in the detached
closure worktree. Parent interfaces/value_place/value_scalar, main and lock are
not modified. No full CLI/selfbuild/ZIP/Linux/remote run is performed here.

## Values/admission step

The missing connections were ref Vec type 7 -> vector type 13 -> scalar element
lookup and the iteration source-family guard. The new
`module_unit_value_index::vector_iteration_element` is deliberately separate
from ordinary indexing: `descriptor[index]` remains rejected. It accepts only a
shared descriptor of authentic SDK vector type with scalar kinds 1..6. The
iteration guard requires an immutable named descriptor and an exactly matching
shared-reference item. Resource/nested-vector elements, mutable descriptors,
exclusive or copied items and unrelated receivers are not admitted by this path.

`TestCandidateCopiedClosureCalls` passed low/optimized (53.579s) after the Values
change. The stronger checked-body replay then passed low/optimized (60.187s),
including both the closure program and ordinary-function control. Both now must
succeed; retaining a code-2 refusal is no longer a pass. Existing origin/loan
analysis rejects in-loop owner mutation with 2101 and returning a local vector
item with 2102. Mutable descriptor, exclusive/wrong item and ordinary Vec
indexing refusals also pass.

Logs: `.cache/vector-for-in/values.log` and `checked-boundaries.log`.
This step alone does not establish generated vector execution. CFG/C capture
connection and final artifact/native execution evidence remain pending.

## Item name scope audit

The for-in parser (`bodies.fy`) creates its target with `node_value(17,...)`.
It is a binding/pattern node, not a kind-2 ordinary name reference. The copied
closure expression rewriter rejects closure values only for kind 2. Rewriting
the target before adding the item therefore does not resolve it as the outer
closure. The body gets a helper=0 item binding; restoring the saved environment
length after the body exposes the outer closure again. The view fixture's
same-name `add` loop passed the prior full artifact-restored C execution in both
profiles (`.cache/closure-for-in/authored-pipeline.log`, 169.458s).
