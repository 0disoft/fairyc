# Fairy Windows local-state provider, 2026-11-07

Status: actual rooted local store, snapshot, overlay CAS and layout CAS
implemented and verified; compiled graph orchestration remains separate.

The coordinator composes the parallel codec/schema work into authored Fairy
file operations. No Go persistence/execution callback replaces the product
path. Go independently opens and mutates the same real authority/state files,
then verifies Fairy-produced bytes and snapshots. The original Go format and
full-state limits are retained.

The final command passes in **17.259s**:

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateStateLocalWindows'
```

The main matrix contains **nine scenarios in both Windows profiles**. It
checks Fairy creation and Go reopen, Go publication and Fairy snapshot, exact
canonical bytes, nil/empty/binary values, unsorted replacement/deletion and an
unmentioned preserved row. No-op/stale/exhausted versions, identical/append
layout publication, valid new-layout reopen and old-layout drift all pass.
Invalid mutations are refused before a deliberately blocked lease. Valid
requests observe that same real Go-held lock as busy, without modifying state.

Two actual Fairy processes open the store, take the same snapshot and wait
outside the lease. After the first publishes, the second performs its single
CAS and reports the exact stale version; only the winner's key exists on Go
reopen. The test schedules publication to separate version conflict from the
provider's independent nonblocking busy policy. It does not replay a loser.

An actual readonly destination forces rename failure after staging write/sync;
old state is byte-for-byte unchanged and the owned temporary file is removed.
Missing/corrupt records stay missing/corrupt instead of being initialized or
repaired. Every scenario checks that no owned `.tmp-` stages remain. Pure
admission separately checks overdrawn structure/byte histories and invalid
policy before filesystem access in both profiles; two sealed Store
construction/field-access cases refuse.

The first combined overlay/provider command exits **1** because its root
replacement fixture assumes Windows permits directory rename with an open
child authority. Overlay checks in that command pass independently; the
failure is retained in `.cache/state-local-store-final.log`. The corrected
fixture verifies either actual replacement or explicit Windows access/sharing
exclusion. On this host both profiles take the exclusion branch and then
successfully publish into the original root; no directory-replacement claim
is inferred. Corrected main matrix passes in **9.116s**, then the final expanded
matrix/admission/owner command above passes. Logs:
`.cache/state-local-windows-final.log`, `.cache/state-local-windows-complete.log`.

Earlier preparation attempts retain malformed nested-switch punctuation and
unannotated integer conditional inference failures. They are fixed in new
code without weakening compiler checks. The first real create/reopen case then
passes in **6.281s** before the broad native matrix.

Fresh commit-scope `check` and `check --locked` pass in **16.713s / 12.818s**,
**747 files / 9,394 MIR functions**, empty stderr. All **1,217 authored files**
match snapshot `.cache/state-store-store-c68aca71`; only the still-uncommitted
native graph wrapper is excluded. Record: `.cache/state-store-store-check.json`.
The snapshot's verified lock describes exactly this provider commit. Scoped
Go vet, gofmt and diff hygiene pass.

Public CLI/SDK ABI, dependencies, runner and repository policy are unchanged.
This adds the compiler-private local provider for the existing file format,
not a production DB/provider choice. Linux, exhaustive I/O/allocator failure
injection, forced-kill publication boundaries, compiled graph persistence,
context cancellation, ordinary CLI and final acceptance remain open. No push,
dispatch, release or deploy.
