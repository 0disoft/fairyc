# Current-source graph disk cache reuse, 2026-11-07

ADR 0526 connects the actual rooted filesystem store to pre-compilation graph
routes and current-source/key-checked Complete restoration. The coordinator
owned this source/test while two disjoint Sol high workers supplied raw-store
transport and lookup prerequisites. It does not substitute in-memory state for
cross-process disk reuse.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateStateGraphDiskCache$'
```

Final Windows low/optimized PASS **38.791s** (test **38.66s**), **49 native
processes/profile / 98 total**, no selected skips. Actual generated Fairy C
executes the cache API against private real directories with an empty PATH.
Independent Go source/body/interface facts check returned decoded-program
identity, public/source hashes and exact source bytes. These are cache API
executions, not native execution of the contained state transaction program.

Cold writes become hits in new processes. Same-coordinate owner 511 rebinding
preserves reuse. Source comments, compiler compatibility and standard-library
changes invalidate lookup. The production adapter using actual compiler image/
SDK environment is executed cold and warm. Additional cold/warm fixtures cover
local reads, branch joins, transitive helpers and borrow/view/copy.

Disk checks independently verify every retained generation's SHA-256 name,
marker and lack of stage leftovers. A corrupt file under its original name is
repaired to exact original bytes. Newer correctly content-addressed wrong-key
and ordinary-family candidates must fail semantic admission before an older
valid hit; additional verification work confirms they were actually examined.
Both observed files remain intact. A cache containing only a semantically bad
generation safely cold-builds/stores a valid result and hits on the next run.
Source failure preserves the entire existing byte inventory and original
phase-8/stage-1 Core diagnostic. No forged or partial Complete is emitted.

Read routing, directory scan, raw I/O and semantic verification all have exact/
one-short budget controls; encoding has the same controls during explicit save.
Cold source exact/one-short/overdrawn controls preserve its own meter and refuse
publication on failure. Already exhausted cache counters fall back to cold
source success with no stored generation or meter reset. Cached verification
and independent cold source histories are never conflated.

One-entry retention is enforced. Two independent processes race through the
same immutable generation without extra files or stages; a later process hits
the result. Incompatible semantic generations are skipped conservatively, not
deleted as proven corruption. No latency claim is inferred from fresh source
verification, and no power-loss or globally locked retention claim is made.

The initial preparation fails on two duplicate local names in the native probe
(E0102); they are renamed. The first execution then passes in 39.933s. Expanded
budget/helper controls pass in 33.335s; the final command supersedes both after
adding digest-valid semantic splices, cold-source boundaries and explicit native
process accounting. Logs are ignored `.cache/state-graph-disk-cache-attempt1.log`,
`state-graph-disk-cache-attempt2.log`, `state-graph-disk-cache-expanded.log` and
`state-graph-disk-cache-final.log`. Failed preparation is not a product pass.

Fresh complete-source `check` and `check --locked` pass **723 source files /
9,034 MIR functions**, empty stderr. All **1,193 authored files** match the
checked isolated snapshot; only its successful lock is copied back. Snapshot
locator: `.cache/state-graph-disk-cache-check-latest.json`. Scoped vet, formatting,
diff and UTF-8/LF/final-newline checks pass.

The explicit graph cache API is complete for the admitted family. Ordinary
workspace source binding/command selection, native graph lowering/execution/
state persistence, wider families and final acceptance remain open. Full Go
suite, Linux native execution and allocator-failure injection are unexecuted.
Public CLI/SDK/runtime ABI, DB, dependencies, default limits, repository and
runners are unchanged. No push, dispatch, release or deployment.
