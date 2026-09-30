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

## Remaining capability work

Each item needs its own bounded design and implementation commit sequence.

| Item | Required implementation and acceptance |
| --- | --- |
| Signed 64-bit CSV values and totals | Add executable `i64` through typed HIR, MIR immediates and verification, artifact transport, interpreter, checked C arithmetic, layout/FFI boundaries and SDK formatting. Test signed endpoints, checked add/subtract/multiply, zero division, minimum divided by -1 and both native profiles. Syntax recognition alone is insufficient. |
| Passing an existing mutable reference to a shared parameter | Define a call-bounded shared reborrow that retains the original owner, freezes competing mutation for its live extent and does not create an independent mutable alias. Test argument ordering, escaping results, retained aliases and whole-root conflicts across interpreter/native lowering. |
| Slicing an exclusive byte view | Define exclusive subview reborrowing and last-use behavior, bounds traps and return origins. A copied slice descriptor must not permit overlapping mutable access. Test both omitted bounds and existing owner loans. |
| Repeated array initialization | Define initializer evaluation count and zero-length behavior; admit only an explicitly supported copy-value subset initially. Do not duplicate resource owners or effects. |
| An integer literal on the left of `1048576 / rows` | Define contextual literal selection without implicit variable-to-variable conversions. Preserve range diagnostics and checked arithmetic, including grouped literals and comparisons. |

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
