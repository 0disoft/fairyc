# Fairy local-state mutation/schema prerequisites, 2026-11-07

Status: pure canonical overlay, append-only schema and current-header
reconstruction implemented; actual Windows provider/native binding tracked
separately. Two disjoint high-reasoning workers own codec mutation and header
reader files; the coordinator owns integration and exact commit-scope checks.

The overlay worker's low/optimized command passes in **25.975s**, comparing
actual Go LocalPersistence/reference bytes for insertion, replacement, deletion,
nil/empty preservation, unmentioned rows, empty/equal/absent-delete version
semantics, maximum versions, layout-before-version guard order, incompatible/
append-only upgrades, immutable caller input and exact/one-short work. Sorting
includes **65,536 unsorted rows**. After source LF normalization, the
coordinator's overlay test independently passes in **23.73s** inside
`.cache/state-local-store-final.log`. That combined command exits 1 only for
the separate Windows directory-rename fixture; its overlay observations are
retained, not relabeled as an aggregate success. The corrected Windows provider
matrix is recorded separately.

The schema worker passes both profiles in **31.49s**. Coordinator regression
on the final source passes in **40.564s**, including all **89 schema fixtures
per profile** plus existing record/empty-layout/sealed-owner cases. Checks cover
old/new/foreign layouts, seven codec identities, zero fields, copied source
lifetime, exact/one-short/overdrawn ledgers and malformed spelling/index/kind/
codec/numeric/hash/name bounds. A canonical record beyond **8MiB** roundtrips
through header reconstruction and the complete record decoder. Separate
structural-only examples deliberately retain checksum/body damage until the
full decoder refuses it. Log: `.cache/state-local-record-regression-final.log`.

Commit-scope isolated `check` and `check --locked` pass in
**13.856s / 13.287s**, **746 source files / 9,377 MIR functions**, empty stderr.
All **1,216 authored .fy files** match snapshot
`.cache/state-store-pure-4ebabcd1`. Only the two still-uncommitted Windows
provider/native wrapper modules are excluded from that scope. Record:
`.cache/state-store-pure-check.json`; its verified lock is used for this commit.
Scoped Go vet, gofmt and diff hygiene pass.

```powershell
go test ./internal/cli -run '^TestCandidateStateLocalOverlay$' -count=1 -timeout=15m
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateStateLocal(Codec.*|Schema)$'
```

Structure/byte histories remain 16MiB/1GiB, including prepaid bounded layout
validation. Linux, exhaustive allocation injection, actual provider/native
orchestration, ordinary CLI and final acceptance remain separate. Public
CLI/SDK ABI, DB/dependencies/runner policy are unchanged. No push or dispatch.
