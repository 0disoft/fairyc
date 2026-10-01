# Practical workload improvement boundaries

This page tracks implementation decisions from the nine local CLI workloads.
An unsupported executable subset is a capability gap, not automatically a
compiler correctness regression. Existing workload sources and recorded binaries
are historical evidence and are not silently migrated to a new SDK.

## Implemented

| Workload finding | Compiler/SDK change | Evidence boundary |
| --- | --- | --- |
| Directory analyzer could not enumerate children | SDK stage0.71 directory enumeration, ADR 0410 | Windows native enumeration and lifecycle; filesystem ordering remains caller-owned |
| Task-manager publication temporarily removed the live file | SDK stage0.72 `replace_file`, ADR 0411 | Public Fairy native linkage plus runtime disk-full, flush, sharing and forced-termination tests; physical power-loss durability is not claimed |
| Running compiler could not identify itself | `fairyc --version` | Embedded compiler/SDK/language/ABI/VCS metadata; absent build metadata stays unknown |
| Calculator could not use `state.position += 1` | Direct integer field compound assignment, ADR 0412 | Existing checked operations, loan checks, interpreter/artifact round trips and low/optimized Windows execution |
| CSV signed totals exceeded i32 | Executable i64 and SDK stage0.73 formatting, ADR 0413 | Checked extrema/arithmetic, artifact transport and low/optimized execution; migrated example retains original workload evidence |
| Existing mutable references could not be passed to shared readers | Call shared reborrows, ADR 0414 | Original origins, retained aliases, tagged results, whole-root conflicts, artifact/interpreter and low/optimized execution |
| Existing mutable byte views could not be sliced | Exclusive byte subviews, ADR 0415 | Parent/owner freeze and last use, returned origins, checked bounds, artifact/interpreter and low/optimized execution; retained siblings remain rejected |
| Workloads expanded buffer initialization by hand | Repeated scalar arrays, ADR 0416 | One initializer evaluation, independent scalar elements, compile-time counts bounded to 1..=4096; empty arrays and ownership duplication remain rejected |
| Literal-left buffer sizing required an extra typed temporary | Contextual integer operands, ADR 0417 | Literal-only trees receive peer/result context; typed values stay exact, ranges and checked operations remain, runtime order is preserved |

The task-manager application now consumes the prepared replacement API in
`examples/task_manager_cli` version 0.1.1. Its 118 CLI regression cases and 20
generated-Fairy storage-fault/termination scenarios pass. See
[task-manager prepared-save evidence](task-manager-prepared-save.md) for restart,
manual orphan disposition and the unchanged power-loss recovery boundary.

## Remaining capability work

The admitted capability work above is implemented. Revalidate the nine original
workloads once against the resulting compiler, preserving their historical
sources and evidence. This page does not claim that revalidation has run.

## Storage faults and restart recovery

Disk exhaustion, forced process termination and power loss are separate acceptance
items. The current prepared-file primitive removes the application-level
delete/backup/publication gap; it does not supply an application storage format.

- Disk exhaustion: retain the live generation after first/partial write, sync or
  publication failure; report exact OS error and completed prefix. Current tests
  inject errors without filling a host drive. Real quota/full-volume testing
  remains a separate environmental proof.
- Forced termination: cleanup cannot be relied on. Current tests kill bounded
  children during preparation and at pre/post-publication checkpoints, checking
  exact live and orphan bytes. Arbitrary interruption inside the OS move is not
  exhaustively proven.
- Power loss: implement a separately validated previous generation or journal,
  explicit record length/version/checksum, commit ordering and deterministic
  startup selection. Reject truncated or corrupt generations; do not promote an
  orphan temporary file solely because it exists. Define cleanup and concurrent
  writer ownership before adding automatic recovery.
- Durability proof: data flush and write-through errors can be injected today,
  but injected errors cannot reproduce controller-cache loss, torn metadata or
  filesystem recovery. Physical power-cut/reboot testing requires a disposable
  lab/storage target and a declared filesystem/device matrix. It has not been
  performed on the user's computer and is not an SDK guarantee.

The generation/journal recovery layer and physical power-loss validation remain
open. They must not be marked complete solely because `sync` or `replace_file`
returns success. See ADR 0411 for the public contract and restart protocol.
