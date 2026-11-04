# Hardening review follow-up, 2026-11-04

Windows local evidence is recorded per slice below. Native Linux execution remains
queued for one combined validation; no Linux pass is implied.

## 1. Stable contract-state type depth

Reject more than 256 compound levels before long prefix suffixes reach type
interning. Boundary and 64 KiB prefix-bomb tests run on Windows. Linux: rerun
`go test -mod=readonly ./internal/contractstatemir ./internal/packageartifact`.

## 2. Windows execution ownership

Owned target processes start suspended, enter a kill-on-close Job, then resume.
Normal completion and timeout terminate/wait for descendants before temporary
image removal. Transient image-release/sharing failures receive bounded cleanup
retries; permanent cleanup failures are returned. Windows subprocess tests cover
stdout inheritance, trap receipts and descendants holding the owned image.
Linux: no Windows Job execution claim; rerun the common cdriver suite later.

## 3. Windows project object residency

Candidate publication uses the image-only project builder. Objects are still
validated individually; only paths survive until linking. Existing object-returning
APIs retain their behavior under a 256 MiB aggregate diagnostic-object budget.
MSVC tests compare exact image bytes and execute ordinary/stack-reserved images.
Linux: rerun common candidatebuild/cdriver tests in the combined pass.

## 4. MIR initialization-state snapshots

Single-predecessor chains transfer owned state instead of retaining/copying maps
for every block. Joins and forks retain independent snapshots under a cumulative
1,048,576-entry copy budget. Entry backedges count the implicit caller edge.
Windows MIR tests cover a 2,048-block chain, loop convergence, budget boundaries,
state isolation and the existing ownership/merge mutation cases.
Linux: rerun `go test -mod=readonly ./internal/mir`.

## 5. Generic materialization

The whole expansion checks its cumulative 1,048,576-node output budget before
cloning specializations. Declaration collection no longer makes discarded AST
copies. Windows genericsource tests cover inclusive boundaries, source immutability,
zero work and arithmetic overflow. Linux: rerun genericsource/compiler suites.

## 6. Expanded array operands

One lowering operation shares an inclusive 1,048,576-operand allowance across
functions. Reservation precedes array slice allocation; repeat evaluation remains
single-shot. Windows MIR tests cover exact/cumulative limits, overflow, malformed
shape precedence and cancellation. Linux: rerun MIR/compiler/backend suites.

## 7. Streaming cache publication

Ordinary/generic/storage-transfer persistent workspace miss paths use scoped batches; single Put keeps
its immediate collection. Checkpoints cap pending publications at 256 entries or
16 MiB (or smaller caller bounds); closing/error paths collect remaining records.
No encoded-record collection or mutable cross-process index is introduced.
Windows packagecache/packageartifact tests pass. A 32-record batch asserts exactly
one full collection; boundary, byte/entry retention, failed collection, idempotent
close and borrowed-handle lifetime tests pass. Linux: rerun packagecache,
packageartifact and workspacecompiler suites in the combined pass.

## Validation scope

Windows focused C-driver subprocess/Job/trap tests, MSVC project equivalence,
candidatebuild, contractstatemir, MIR, genericsource, packagecache and packageartifact
checks passed. The expanded `internal/compiler` suite reached its local five-minute
package timeout during native cases, so that attempt is incomplete, not a pass.
The exact active nominal-generics native case and core Check cases are rerun separately.
No full `go test ./...`, Linux execution, hosted CI dispatch, push or release is claimed.

The separated nominal-generics native case and six core Check cases passed on
Windows. The current ordinary workspacecompiler suite and focused eight-package
vet also passed. The 256-block/256-local MIR allocation benchmark compared the
baseline verifier through a Go overlay against the changed verifier (three iterations
per measurement): 4,964,394 to 28,360 B/op, and 2,599 to 283 allocations/op.
Elapsed timings are not used: concurrent unrelated machine builds varied CPU load.

After extending batching to generic/storage-transfer publication, focused generic,
storage-transfer and persistent tests in packageartifact/workspacecompiler passed,
and vet for all affected cache packages passed again.
