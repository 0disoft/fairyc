# Fairy Region byte operations, 2026-11-06

Part of the complete implementation queue. The original Go implementation already
admits these operations; this change connects the Fairy candidate implementation.
It does not finish Atlas production, broader source admission or final acceptance.
Native Linux remains in the maintainer's combined batch; no push or dispatch.

## Implementation

`module_unit_region_builtin.fy` names the three private operation IDs: existing
`place`, `alloc_zeroed_bytes` and `copy_bytes`. Ordinary and specialized source
calls, SDK type demand, type checking, origin analysis, retained Region metadata,
detached-IR verification and CFG receiver evaluation use that same classification.
Source spelling alone never grants a Region capability.

The count is contextually checked as `usize`. Copy accepts a shared or mutable
byte view; it retains no borrow of the input after completion. Both byte operations
return `Result<ref mut u8[], alloc::AllocError>` bound to the original Region.
Plain object placement keeps its original inert-owner restrictions. Other input
types, raw pointers and manufactured Region capabilities remain rejected.

The C emitter reuses existing allocation/runtime status and lifetime contracts.
Zeroing uses `fy_std_region_allocate_zeroed_v0`; copying uses
`fy_std_region_copy_bytes_v0`, performing one byte copy without a preceding zero
fill. The existing runtime explicitly handles null data for empty copies. No new
runtime symbol, public ABI, SDK declaration or grammar production is introduced.

The semantic interpreter uses its existing bounded, stable per-execution heap and
view descriptors. Count/step exhaustion remains an explicit interpreter limit;
it is not an observation of the host allocator's out-of-memory behavior. Generated
C uses the actual existing host allocation runtime.

Checked placement descriptors survive an earlier terminal call argument or a
terminal allocation operand. Descriptor presence is not an execution observation:
the verified CFG still excludes the unevaluated allocation. The original Go
fourteen-case corpus includes both partial argument orders and all three terminal
operands. A later added trap-before-next-statement case identified a further
Region metadata traversal prerequisite, recorded in the Atlas relation plan.

## Windows validation

`TestCandidateRegionBytesExecution` compiles the actual authored compiler source
closure. Each generated compiler runs with an empty PATH, checks a current locked
SDK workspace, executes its retained IR in the Fairy interpreter, and emits C.
The Go harness then compiles that exact C with the existing private candidate
consumer policy and executes it under both native profiles.

The eight cases cover zero-length allocation, zero initialization and mutation,
empty copy, NUL/0xff bytes, array-copy isolation, mutable-view-copy isolation, a
Region parameter, and ordinary user methods with the same operation names.
Expected values are source assertions; native programs must
exit zero with empty output. No Go subprocess supplies the compiler's result.

The first harness incorrectly requested the partitioned C builder with empty
framing and failed its input guard. A direct strict public-C consumer then
reported existing private-transport warnings. The final harness uses exactly the
same nonempty single-buffer framing and private consumer policy as the existing
aggregate C tests; no compiler warning level or product limit was changed.
Those failed attempts remain in `.cache/candidate-region-bytes-execution-first-20261106.log`
and `candidate-region-bytes-execution-second-20261106.log`.

The final command passes in 371.984s (body 371.85s; low 279.97s, optimized
73.29s), with no selected skips:

```powershell
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateRegionBytesExecution$'
```

Log: `.cache/candidate-region-bytes-current-20261106.log`. The sixteen generated
compiler processes each run the independent Fairy interpreter and emit C. The
two consumer profiles execute thirty-two compiled products. Neither count is a
native Linux result or a full Go/Fairy command comparison.

The Region-only commit's unlocked and locked checks pass for two instances,
677 source files and 8,428 MIR functions; the lock is updated and then verified.
The separately authored, uncommitted Atlas allocation collector was temporarily
held outside `src` during these two checks and restored afterwards. This keeps
the committed lock bound to exactly the committed source files. Logs:
`.cache/region-bytes-commit-check-20261106.log` and
`region-bytes-commit-locked-20261106.log`. Source and final acceptance remain
separate from the Atlas allocation collector/map/CLI work.

No public CLI/API, DB, dependency, runner, target-maturity or repository-policy
change is made. Linux native execution, full Go/Fairy comparison, consecutive
self-build, clean-machine distribution and same-commit hosted CI remain open.
