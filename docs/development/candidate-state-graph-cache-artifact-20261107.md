# Key-bound graph cache transport and restoration, 2026-11-07

ADR 0523 binds the independently derived graph cache identity to the existing
exact-source envelope. Restoration returns an actual checked common Complete
only after fresh source/package admission and recomputation of the complete key.
This source/test slice was implemented by the coordinator while separate Sol
high workers owned the memory-cache and key files.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphCacheArtifact$'
```

Final Windows low/optimized PASS in **30.801s**, **22 processes/profile / 44
native processes**, no selected skips. A Go-authored outer framing constructor
checks exact source retention and source-envelope framing. The independent Go
source/body oracle checks returned Complete identity for owner 0/7/511 rebinding
and the budget/lifetime case; canonical key bytes themselves are independently
covered by the ADR 0522 prerequisite, rather than credited twice here.

Coverage includes source/comment drift, coordinate, environment and standard
library mismatch; valid alternate key splices; valid alternate source-envelope
splices, including a source-valid packet paired with the wrong key; wrong
family marker, trailing/truncated bytes, oversized section counts and complete
payload sizes 8 MiB-1, 8 MiB and 8 MiB+1. The first two payloads have valid outer
framing but invalid inner content and are correctly classified 1001; the
one-over bound is 1002. These are not claims that a maximum-sized valid source
graph fixture exists.

Encoding and restoration both pass exact/one-short and already-overdrawn work
controls under one ledger. Re-encoding a restored Complete preserves bytes.
Returned Complete remains valid after destruction of request source/coordinate
buffers. This fixture executes deterministic compatibility snapshots; the thin
production environment wrappers are source-checked, not separately executed.
Actual environment/key derivation is covered by the preceding key suite.

The first preparation fails because the Go oracle fixture uses reserved module
name `transaction`; it is renamed to `ledger`. The next fails workspace loading
because a probe-only body-identity import is absent from its isolated closure;
the test explicitly supplies that fixture module. No product refusal is relaxed.
The first narrower execution passes in 26.691s; the final command supersedes it
after adding library/source-valid-key-splice and exact payload-bound controls.
Logs are `.cache/candidate-state-graph-cache-artifact-20261107.log`,
`candidate-state-graph-cache-artifact-closure-20261107.log` and
`candidate-state-graph-cache-artifact-final-20261107.log`. The intermediate
workspace failure is retained in tool session `44719`, not claimed as a saved
log after the final log replaced that path.

Fresh complete-source `check` and `check --locked` pass **721 source files /
9,013 MIR functions**, empty stderr. All **1,191 authored files** match the
checked isolated snapshot; only its successful lock is copied back. Snapshot
locator: `.cache/state-graph-cache-artifact-check-latest.json`. Scoped vet,
formatting, diff and UTF-8/LF/final-newline checks pass.

There is no graph disk store, ordinary CLI cache selection or native graph
execution claim. Full Go suite, allocation-failure injection, complete self-build,
distribution and same-commit hosted acceptance are unexecuted. Native Linux
stays batched. Public CLI/SDK/runtime ABI, DB, dependencies, default limits,
repository and runners are unchanged. No push, dispatch, release or deployment.
