# Raw graph cache disk transport, 2026-11-07

ADR 0524 admits the graph envelope family into the existing rooted byte store.
The disjoint Sol high worker modified three source lines and added one test;
the parent owns source checks, lock, documentation and this commit.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphDiskTransport$' -count=1 -v -timeout 10m
go test -mod=readonly ./internal/cli -run '^TestCandidateCacheDiskWindows$' -count=1 -v -timeout 10m
```

New Windows low/optimized transport tests PASS **8.293s**, eight leaf cases.
The existing ordinary cache regression PASSes unchanged **22.994s**, two native
profile cases, including its five authored fixture groups. There are no selected
skips. New opaque payloads test transport only, never source/Complete admission.

Actual filesystem operations cover all three markers; binary bytes and digest-
bound generation names; put/load/candidates/next/sweep; corrupt-data rejection
and exact observed collision repair; missing routes; entry/byte retention; live
handles and staging cleanup. Each process opens real private temporary roots.
Old ordinary fixture assertions are retained.

Initial preparation fails on three missing probe semicolons. The next run has
four leaf failures caused by incorrect test byte totals/retention limits. Only
test expectations/setup are repaired. Final results supersede these attempts;
they are not combined into one fictitious successful run. Ignored logs are
`.cache/graph-disk-transport-native-1.log` through `native-3.log`,
`graph-disk-transport-ordinary-regression.log` and
`graph-disk-transport-checks.log` (all names share the graph-disk-transport prefix).

Fresh complete-source `check` and `check --locked` pass **721 source files /
9,013 MIR functions**. All **1,191 authored files** match the checked copy; only
its successful lock is copied back. Snapshot locator:
`.cache/state-graph-disk-transport-check-latest.json`. Scoped vet, formatting,
diff and UTF-8/LF/final-newline checks pass.

This is raw byte transport, not graph disk cache reuse, source admission or
ordinary CLI selection. Full Go suite, Linux native execution, allocator-failure
injection and final acceptance remain unexecuted. Public CLI/SDK/runtime ABI,
DB, dependencies, default limits, repository and runners are unchanged. No push,
dispatch, release or deployment.
