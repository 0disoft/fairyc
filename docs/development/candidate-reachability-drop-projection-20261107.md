# Checked CFG drop projection

The normal current-source compiler previously passed project linking and failed
in C emission with phase 5/stage 0/code 1002. An authenticated detached replay
isolates the refusal to the reachability work ledger, without changing a limit.

At work 1048576, the profiler had visited 8299 of 10292 frames. It recorded
637322 CFG-block visits, 278010 target-lookup work, 2013 foreign-contract lookup
work, 49075 calls and 68506 mark attempts. No pending cleanup type had yet been
processed. Thus the prior specialization-index experiment alone did not remove
this failure.

`module_unit_cfg_model` now retains original drop-block IDs during its existing
charged block-validation walk. Construction validates the borrowed input vectors
before moving them into the immutable graph, then runs the same iteration,
switch, defer and Region checks with the same work ledger. The public `check`
signature still independently validates all blocks and collects no projection.
Allocation failure returns 1003 and publishes no graph prefix.

Reachability reads only these recorded drop blocks, in exactly the previous
block order. Frame checks, calls, per-site charging, repeated marks, nested
cleanup dependencies and fixed-point processing are retained. This changes the
cleanup-dependency pass from O(B) block visits to O(D) drop-site visits after CFG
verification, using O(D) private scalar-ID storage. It does not grant new proof,
artifact, cache-hit or publication authority. CFG block/work limits and the
whole-project ledger are unchanged. No CFG wire format exists to change.

## Windows evidence

- Original profiler: `.cache/current-project-reach-profile-20261107/run.json`.
  The foreign logger is test-only; neither it nor diagnostic error labels are
  product code.
- Production source `check` and `check --locked` pass: 755 source files,
  9536 MIR functions, 19.681s / 15.707s. The complete copied tree also contains
  fixtures and tests; its 1225 `.fy` files are not the compiler's source count.
- Fresh optimized diagnostic probe build passes in 68.861s.
- `TestCandidateReachabilityWork` passes Windows low and optimized (2/2) in
  an isolated worktree. Its two production-file hashes match the integrated
  source. Repeated-call/drop fixtures of size 1, 16 and 128 use 82, 412 and
  2876 work respectively; foreign-only size 128 uses 915 in both profiles.
  It checks independent expected frame/type sets, cycles, separate generic
  contexts, nested Option/Result cleanup, dead frames, original CFG-block order,
  repeated public checks, exact work boundary/overflow, malformed input and a
  later defer-check rejection. Test-only IR constructors remain isolated from
  production APIs. No old/new fixture timing reduction is asserted.
- `.cache/current-project-reach-drop-replay-20261107/run.json`: the same 157
  checked IR packets (496065867 bytes) link and emit successfully in 40.790s,
  with 10292 frames, 81 C units, a 1842785-byte header and project work 9989821.
  Input/archive/image and changed production-source hashes remain unchanged;
  stderr is empty. The unchanged bounds are 1 MiB per C phase and 16 MiB total.

The detached replay uses the original frozen 646-source snapshot. It is
diagnostic evidence, not current-source normal acceptance, an executable build,
a cache hit, a fixed point or full Go/Fairy parity. The current-source normal
retry is recorded separately when completed.
Linux native checks remain batched. CI, push and release are not run here.
