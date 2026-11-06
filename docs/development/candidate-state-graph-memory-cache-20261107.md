# Explicit state-graph memory cache, 2026-11-07

ADR 0521 adds an owned, bounded source-envelope cache. It returns an actual
common Complete after cold build or full current-source restoration. It does
not cache fabricated proof fields or select a normal workspace command mode.

The disjoint GPT-6.1 Sol high worker changed only the new source and test. The
parent reviewed both, owns this receipt and checked the staged source snapshot.

## Windows evidence

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphMemoryCache(Sealed)?$' -count=1 -v -timeout=15m
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphMemoryCacheSealed$' -count=1 -v -timeout=5m
```

The first command exits 1: native tests PASS in **26.76s**, but one of four
ownership tests expects E1008 while the compiler correctly refuses with E1006.
Changing only that expected diagnostic makes the second command PASS in
**22.903s** (test **22.75s**), all four refusals. Native product/probe inputs did
not change between these commands, so their successful results are retained.
There is no fabricated combined successful command.

Native low/optimized each execute 35 processes and 78 build requests: totals
**70 processes / 156 requests**, including **130 successful Complete returns**
and **26 expected request errors**. Four additional constructor refusals are
included in the process count, outside build-request counts. No tests skip.

Four independently checked Go source/body/interface fixtures cover local reads,
branch joins, transitive helpers and borrow/view/copy. Cold, warm and same-
coordinate owner rebinding agree on decoded-program body bytes, canonical body
identity, public descriptor and public/source hashes. Source/comment/coordinate
changes invalidate reuse. Mutation/destruction of request bytes cannot alter
owned cached packets or returned Complete values.

Disabled, entry-full and byte-full sessions retain cold fallback. Cold store,
warm restore and these three fallback routes pass exact/one-short shared-work
controls. Invalid requests, failed work and exhausted meters do not insert
entries, reset the ledger or report hits. Constructor bounds cover 512/513
entries and 8 MiB/one-over byte limits.

The worker did not save log files: execution evidence is its tool records,
agent `01a1b290-1106-74d0-a451-92002dc35f58`, sessions `36856` and `2538`.
The parent snapshot's actual `check` and `check --locked` both pass
**719 source files / 8,995 MIR functions** with empty stderr. All **1,189 authored
files** match that checked snapshot, whose successful lock is copied back.
Snapshot locator: `.cache/state-graph-memory-check-latest.json`.

Scoped vet, formatting, diff and UTF-8/LF/final-newline checks pass. Full Go
suite, allocator failure injection, ordinary CLI cache selection, filesystem
graph persistence, native graph execution and final acceptance are unexecuted.
Linux execution remains batched. Public CLI/SDK/runtime ABI, DB, dependencies,
default limits, repository and runners are unchanged. No push or dispatch.
