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
Values alone does not establish generated execution. The separate CFG/C step
and final artifact/native evidence below complete the scalar vector connection.

## CFG/C step and final execution

CFG iteration proof mirrors the narrow descriptor/element checks. Existing
loop-item locals, capture/step blocks, bounds and cleanup remain unchanged.
C capture reads the data pointer and initialized length once from the existing
authenticated six-field SDK Vec layout. `module_unit_c_sdk::vector_layout`
exposes its existing private layout index; existing 48-byte/8-byte alignment and
field-offset static assertions still govern the descriptor. The consumer reuses
the existing SystemHeap/private vector storage support. No new MIR opcode,
allocation policy, public header/ABI, cap, timeout or ownership rule is added.

`TestCandidateCopiedClosurePipeline` parses the root plus actual bundled
`internal/sdkstdlib/sources/optional/std/vec.fy`, with explicit import binding,
separate owner IDs and coordinates. It runs complete authored compilation,
artifact encode/decode/restore and C publication. Both view and scalar vector
fixtures execute from one program: empty loop, first/last item, ordered visits,
snapshot 40 after source mutation to 100, direct/adjacent calls, item-name
shadowing and the outer closure after the loop. Expected result is 42 with no
stdout/stderr.

Final PASS: 179.891s, 102-part purpose-built compiler host, low and optimized.
Each host's restored C also passes both low and optimized consumer profiles:
four executions, all exit 42 and empty output. This is not a full authored CLI,
selfbuild, fixed point or general vector execution result. Scalar Vec<i32> is
the runtime representative; other scalar element widths were not individually
replayed here. Resource/nested elements and retained/asynchronous cases remain
outside the explicit guards. RSS, memory cost and speedup were not measured;
timing includes concurrent same-host parent/Browniec work.

Commands (Windows, this detached worktree):

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosureCalls$' -count=1 -timeout 6m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosurePipeline$' -count=1 -timeout 8m -v
bun tools/run-go.ts vet -mod=readonly ./internal/cli
git diff --check
gofmt -l internal/cli/candidate_copied_closure_test.go internal/cli/candidate_copied_closure_pipeline_test.go
```

Logs in `.cache/vector-for-in/`: `checked-boundaries.log` (PASS),
`artifact-pipeline.log` (PASS), `vet.log` (PASS, empty). The first pipeline attempt
used `.module_index` instead of catalog Binding's `.source` in the new fixture;
Go lowering rejected it before any C build. Its failure is preserved separately
as `artifact-pipeline-fixture-field-failure.log` and is not a pass. Production
semantics were not changed for that fixture error. `receipt.json` records exact
commits, file hashes and commands. The pipeline was built from `852304d7` plus
the uncommitted CFG/C and pipeline files whose final hashes are recorded there.
The subsequent local commit retains those tested bytes.

Prior Go copied-closure/native/cache evidence is reused from
`.cache/closure-for-in/go-tests.log` (compiler 11.705s, workspace 3.348s): its
Go sources and shared fixtures are unchanged by this continuation. Full CLI,
full source/locked checks, Linux, ZIP, remote and CI are deliberately skipped;
parent owns the merged lock and whole-compiler acceptance.

## Item name scope audit

The for-in parser (`bodies.fy`) creates its target with `node_value(17,...)`.
It is a binding/pattern node, not a kind-2 ordinary name reference. The copied
closure expression rewriter rejects closure values only for kind 2. Rewriting
the target before adding the item therefore does not resolve it as the outer
closure. The body gets a helper=0 item binding; restoring the saved environment
length after the body exposes the outer closure again. The view fixture's
same-name `add` loop passed the prior full artifact-restored C execution in both
profiles (`.cache/closure-for-in/authored-pipeline.log`, 169.458s).
