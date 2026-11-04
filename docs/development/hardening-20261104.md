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
