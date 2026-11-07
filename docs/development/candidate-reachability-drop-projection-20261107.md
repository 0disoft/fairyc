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
diagnostic evidence, not source-bound acceptance, a cache hit or a fixed point.

## Normal source-bound build at b4008e0e

The fresh full CLI seed/help passes in 239.176s (205 Go-emitted C parts).
`.cache/current-selfbuild-reach-drop-20261107/receipt.json` then records
`passed=true`, full `file_cli::run` scope, both development/release images,
`goFrontendInvoked=false` and `runtimeGoBuilderRequired=false`. The authored
companion's prior receipt is reused only after all its source/runtime hashes
are checked; neither changed module is in that 36-source closure.

The normal, uninstrumented 646-source compiler emits a 55072792-byte C packet
in 522.053s with empty stderr. Development compilation takes 48.709s and
release compilation 663.291s, both followed by exact frozen-version checks.
Packet SHA-256 is
`9142ab65d252d51cde0e7ed3695ca21bb8e8795f422164f34236b969154ff54f`.
This deliberately runs with `GOMAXPROCS=1` while a separate native worker uses
the other CPU slot; its serial release time is not a two-worker benchmark.

The build revision is exactly `b4008e0ebf7ecfc777eea62502cbdab687508b87` with
`modified=false`. This proves the drop-projection source revision, before the
later enum/switch layout-lookup integration. It does not prove a fixed point
or full Go/Fairy behavioral parity for either revision. Linux native checks
remain batched; CI, push, publishing and release packaging are not run here.
