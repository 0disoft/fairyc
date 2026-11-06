# Fairy-owned Windows graph native orchestration, 2026-11-07

Status: explicit private Fairy graph build/run/response admission implemented
and verified on Windows amd64. Linux execution remains batched. Persistent CAS,
ordinary command selection and final acceptance remain open.

Two disjoint parallel workers own the MSVC builder and capture allowance;
the coordinator owns execution/response integration, final review and commit.
Production preparation, compiler commands, image execution, request encoding
and response admission are authored Fairy code. Go supplies independent oracles
and tests; it does not replace those product stages.

## Implementation and admission

`module_state_graph_windows_build.fy` consumes Prepared plus the pinned runtime,
charges both source copies against the existing shared 16MiB preparation ledger,
compiles/validates two COFF objects and links/validates the owned PE. A sealed
Built retains the exact Prepared/image pair. Options and invalid work refuse
before host I/O; failed compile/link diagnostics retain their cause fields,
remove scratch-path prefixes and relinquish scratch. The builder adds no
independent 8MiB source policy.

`module_state_graph_windows_run.fy` derives Input from that same Built/Snapshot,
writes image and replay stdin into exclusive scratch, executes the image and
feeds actual received stdout into same-owner response admission. Options and
overdrawn work refuse first. Request/response failures preserve their consumed
ledgers; process failure retains stderr diagnostics and native status. Admitted
read/write/trap values survive cleanup of the child buffers and scratch. The
runner has eight distinct failure stages and grants no persistence authority.

The default capture allowance remains 64KiB across stdout and stderr. Explicit
allowances are validated in 1..64MiB before launch; graph execution opts into
64MiB. Excess bytes are drained, not retained. Pipe progress now polls process
completion without sleeping; only idle rounds wait, with deadline checks in
both cases. Trap-control receipt handling retains its separate bound.

## Windows observations

The builder worker's final low/optimized actual build checks pass, including
COFF/PE inspection, independent Go build output comparison, six pre-I/O option/
budget refusals per profile, forced compile/link errors, diagnostic scrubbing
and scratch cleanup. Five sealed-owner construction/mutation refusals pass.
The final source allowance is 16MiB; successful final worker results are reused
on unchanged builder sources rather than recompiling that matrix again.

The coordinator's final integration command passes in **81.623s**:

```powershell
go test -mod=readonly -count=1 -timeout 20m -v ./internal/cli -run '^TestCandidateStateGraphWindowsRun'
```

It executes **16 distinct graph sources in both profiles**, **32 actual Fairy
preparation/build/run/admission invocations**, plus the selected repeated exact/
one-short ledger and invalid-option controls inside those invocations. Cases
cover commit/reject/trap, six storage actions, all seven codecs, Unicode and
maximum u64, maps and original keys, helper calls/branches, borrow/view/copy,
seeded branch state and a **2MiB read at version 73 under a 4KiB write-value
limit**. Results match independent Go/MIR expectations. Each invocation checks
its temporary root is empty after success and selected failures. The returned
owner remains readable after destroying source input and later executions.
Sealed Built construction is independently refused. Log:
`.cache/graph-windows-run-final.log`.

The final capture boundary command passes **25 cases/profile, 50 total**, in
**25.813s** (`.cache/graph-windows-capture-final.log`). It covers explicit
input/environment entry points, minimum/exact/one-over binary/stdout/stderr/
combined limits, invalid 0 and 64MiB+1 before launch, unchanged default entry
points at 64KiB and 64KiB+1, and exact 64MiB capture. The old wait-per-read
implementation fails both maximum cases at **120.03s** in
`.cache/graph-windows-capture-attempt3.log`; that command remains a failure.
After progress polling changes, the final optimized maximum case passes in
**1.46s**. The initial corrected low maximum-only check also passes in 11.855s
including preparation, with the case itself at 9.33s. These are local timing
observations, not cross-machine performance claims.

The separate regression worker executes the existing **five authored I/O tests
in both profiles** in **12.947s** (low 4.14s, optimized 4.74s), preserving private
receipt/location decoding, output/control overflow draining, normal-vs-trap
classification, timeout job-tree termination and unchanged handle counts, plus
binary stdin and exact forwarded stdout/stderr. The receiver is rebuilt from
current `fairyc-candidate-process-reference` Go source; tiny freshly built C
children supply the trap receipt/ordinary exit protocol. This is capture
transport regression, not new compiler trap-lowering evidence. The independent
pure signed-exit test is not rerun. Log:
`.cache/graph-windows-capture-authored-regression.log`.

An initial orchestration attempt fails at a missing explicit return in the
builder's failure switch (`.cache/graph-windows-run-attempt1.log`). The builder
is corrected before final native runs. An attempted Go regression selector
matches no tests; that command is not counted as regression coverage.

Fresh isolated full-source `check` then `check --locked` pass in
**15.293s / 14.135s**, with **743 files / 9,300 MIR functions**, empty stderr.
All **1,213 authored .fy files** match the checked copy; only its successful
lock is copied back. Record `.cache/graph-windows-run-check-latest.json` uses
snapshot `.cache/graph-windows-run-check-7e6a9489` and bootstrap SHA256
`4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`.
Scoped `go vet -mod=readonly ./internal/cli`, gofmt, diff and UTF-8/LF hygiene
checks pass. No unchanged broad native matrix is rerun after documentation-only
edits.

Remaining scope: one-attempt persistent layout/version CAS, ordinary workspace
command selection, complete repository Go regression, full current-source
Go/Fairy/consecutive self-build and distribution/clean-machine acceptance.
Linux execution and same-commit hosted CI remain pending by user instruction.
Public CLI/SDK/runtime ABI, DB/provider, dependencies, CI/runners and repository
policy are unchanged. No push, dispatch, release or deployment.
