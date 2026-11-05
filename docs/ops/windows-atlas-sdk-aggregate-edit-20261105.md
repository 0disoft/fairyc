# Windows mixed SDK/user aggregate body edits, 2026-11-05

## Scope

The actual Fairy `experimental atlas-edit replace-body` command is exercised on
the shared mixed-source fixture: authenticated SDK Vec<i32>, a same-name public
client Vec<T>, a public Flag<bool> resource with nonempty cleanup, and a private
leaf that uses all three. The replacement keeps Flag's cleanup and changes both
the SDK Vec and client Vec to bool before returning a different i32 value.

`TestAtlasSDKUserAggregateEditOracle` independently verifies the Go preview and
derives current Go v3 facts for its resulting bytes. Both SDK Vec<bool> and client
Vec<bool> are absent from the original program and present afterward. The SDK
must additionally materialize empty/push/at/write_slot/take_slot bool instances.
This prevents a superficially successful report from substituting the old i32
specializations. The focused Go oracle passes in 0.727s.

The native verifier runs with PATH empty and compares exact preview and apply
reports with independent Go executions. A bool client field initialized with i32
must refuse with exit 2 and preserve every file. Direct editing of the SDK at
declaration must likewise refuse without writes. Successful apply retains all
four pre-publication source/overlay checks, publishes exactly the reported body
bytes and leaves graph-lock bytes unchanged with no staging/lock debris.
The old request must first fail against the stale graph lock; after independent
refresh, its old source digest must still fail. Both failures preserve current
bytes. Existing publication, source authentication and cleanup owners are reused.

This verifies checked source publication, not execution of the edited client
program or mixed-source cache hits. The original SDK declarations remain
noneditable; no source-plan or intrinsic capability is transferred to clients.

## Commands and results

The native command passes in 451.186s: low 363.26s and optimized 75.83s. Actual
apply takes 2m36.2262948s at low and 7.096246s at optimized, within the unchanged
test-only six-minute diagnostic limit. Both profiles retain all four source/
overlay checks; no product timeout or warning policy is changed. Preview/apply
report equality, exact published bytes, invalid-overlay and SDK-write refusal,
unchanged lock, debris cleanup and both old-request failures pass, with no
selected skips. These are acceptance durations rather than isolated benchmarks.
Scoped vet, Go formatting and diff checks pass.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 4m -v ./internal/cli -run '^TestAtlasSDKUserAggregateEditOracle$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateEditSDKUserAggregateCommand$'
go vet -mod=readonly ./internal/cli
```

Logs use `.cache/atlas-sdk-aggregate-edit{,-oracle}-20261105.log`. The shared
scoped vet log is `.cache/atlas-sdk-aggregate-commands-vet-20261105.log`.

The mixed editor verifier is registered in full authored-main acceptance, which
is not rerun for this command-leaf slice. More Atlas source families also remain
separate.
Product source remains unchanged from `2b1a2d2a`, retaining its source/lock and
producer results. Final whole-compiler comparisons, consecutive self-build,
Windows distribution/clean-machine and hosted same-commit CI remain open.
Native Linux stays batched. No grammar, public CLI, SDK/runtime ABI, map wire,
DB, dependency, product timeout, target, repository hygiene or CI runner change.
No push, dispatch, release or deployment.
