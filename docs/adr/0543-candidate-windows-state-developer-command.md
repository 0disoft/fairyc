# ADR 0543: Actual authored Windows state developer command

- Status: Accepted for the existing explicit private native-graph command family
- Date: 2026-11-07

`tools/dev/fairyc-state/main.fy` is an actual authored process entry. It delegates
to `module_state_command_windows`, reads the real process argv and preserves the
existing explicit init/execute/snapshot syntax and exit classes. It is separate
from the Go developer entry and adds no command to the public main CLI. The
three-argument Go-MIR form is explicitly unsupported, with no fallback.

Help and invocation refusal precede source/asset/storage I/O. The admitted source
file is read through the retained-root Windows reader, freshly checked and bound
to its ASCII basename/module identity, matching the Go developer source rule.
The authored source ceiling remains 64KiB. The raw source helper does not invent
a path identity; the command owns that binding. A Unicode/spaced parent directory
does not change the source basename or program identity.

The reader preserves the existing rooted non-reparse regular-file policy. It
copies from one retained file, checks before/after metadata and length, reads at
most 64KiB per syscall and one EOF sentinel, closes explicitly on success and
uses owning cleanup on failures. Cumulative 16MiB work includes path/data/sentinel
costs. Explicit cancellation precedes host access, occurs between chunks and is
checked before success; native/token/close failures retain stage/code/OS cause.
Drive-relative leaves are refused. UNC/device/ancestor-reparse behavior is not
broadened beyond the existing rooted contract.

The actual executable finds pinned runtime/frame bundles next to its own image:
`fairyc-state-graph-runtime.pack` and `fairyc-state-graph-frame.pack`. Bundles are
read with a 1MiB bound and admitted by their existing authoritative source pins.
No Go compiler/executor or arbitrary provider supplies product execution results.

Execute prepares and builds the actual sealed native image before touching the
store, then takes one snapshot, runs once and attempts at most one complete CAS.
Init/snapshot never resolve MSVC, build the graph image or execute its body.
Policies match the existing Go developer path: 16MiB/65,536-entry/64KiB-value
store; 1,024 writes, 1MiB write bytes, 64KiB value, 1,024-byte map key; bounded
60s native commands. Busy/layout/version conflicts remain exit 7; reject 6,
trap 8, invocation 2, source 3 and I/O/backend 4.

Outcome now retains its sealed producer's complete 32-byte transport program
identity in a private ordinary record. `persistent::matches` binds report metadata
to the exact program, including different bodies with the same layout. This
adds constant space and no allocation, fallible operation or cancellation poll
after acknowledged publication. It exposes no arbitrary Outcome constructor.

Execute JSON matches Go v2, including typed encoded reads, cost, exact receipts,
reject rollback and trap omission of Outcome. Known publication cannot become a
cancellation/retry result. Acknowledged inconsistency retains its publication and
the existing Go warning text; output/construction failures warn that an executed
commit may already be published. Init's create/snapshot/report phase is likewise
not revoked by a token arriving after entry. No automatic retry exists.

The explicit `run` borrows a token. ADR 0544 subsequently connects the actual
default process entry to a local event and console Ctrl-C/Break registration. A
repeatable developer build/distribution recipe, forced command-level conflict/
late cancellation/broken-output observations, broader path/source admission and
all final queue gates remain open. Public CLI/SDK/native ABI, database/provider,
dependencies, repository policy and CI/runners remain unchanged.

Evidence: `docs/development/candidate-windows-state-command-20261107.md`.
