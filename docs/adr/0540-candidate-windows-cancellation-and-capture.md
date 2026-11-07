# ADR 0540: Explicit Fairy Windows cancellation and capture

- Status: Accepted for the compiler-private Windows host path
- Date: 2026-11-07

`windows_cancel` owns a sealed, non-inheritable Windows manual-reset event.
`none()` has no native handle and polls false without OS access. Named create
and open are distinct: create rejects an existing object without resetting or
signaling it; open requires an existing event. Signal is latched; no reset or
automatic retry exists. Invalid names and signaling none fail before OS access.
The private name family is `Local\fairyc-cancel-` with 1..128 ASCII suffix
characters from letters, digits, dot, underscore and hyphen. Names are trusted
local coordination inputs, not public credentials or hostile-user isolation.

Three closed compiler-private adapters provide exclusive create, existing open
and signal. They retain native errors and use the existing process host's wait,
close and error operations. Go and Fairy selectors admit precisely those names;
the existing hash-pinned runtime pack is regenerated from authoritative C. The
SDK/public native ABI, default process/deadline/output policy and Linux host
remain unchanged. No arbitrary system import bypass is added to product code.

`windows_run_capture` adds explicit `run_input_cancel` and
`run_environment_cancel` entry points taking a borrowed Token after the existing
arguments and output bound. All existing functions retain their signatures and
delegate with none. Options/bounds precede token observation. A signaled token
refuses before environment/input/launch access where observed; another checkpoint
immediately precedes launch, and every capture loop observes cancellation.
Observed cancellation is private problem kind 12, with the real child PID if
created. Failed token observation is kind 13 with its OS cause. Neither is
presented as timeout or a completed process outcome.

After launch, cancellation terminates the owned job tree, waits on the process,
and lets existing resource cleanup close the process/thread/job, input and pipe
owners. Bytes and trap receipts are not returned as a successful Captured result
after cancellation. Existing deadlines and bounded overflow draining remain.

This establishes explicit cross-process cancellation for the capture layer.
Graph builder/runner/persistence wrappers still require explicit token plumbing
and cancellation checkpoints before CAS. Default callers remain uncancelled;
OS console-handler/Ctrl-C integration is separate. This does not claim Go
context equivalence, cancellation inside filesystem publication or ordinary
state CLI selection.

Evidence: `docs/development/candidate-windows-cancellation-20261107.md`.
