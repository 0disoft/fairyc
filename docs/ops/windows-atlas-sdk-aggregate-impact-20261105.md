# Windows mixed SDK/user aggregate impact command, 2026-11-05

## Scope

The actual Fairy `experimental atlas-impact` command is exercised on one source
containing authenticated SDK Vec<i32>, a public client struct also named Vec<T>,
and a public client Flag<bool> resource with a nonempty assertion cleanup. The
private leaf constructs all three and calls the SDK's empty/push/at functions.
The independent Go v3 current-source call graph contains 127 resolved calls.

The command fixture selects the SDK's original generic at declaration by visible
ID, then the client's leaf, Vec and Flag by full ID. Every result compares
schema, subject, nodes, contexts, traversal and omission facts with the independent
Go query, preserving explicit partial coverage and the actual executable hash.
An empty PATH prevents use of a Go child or companion. No prebuilt map supplies
the native command's current source facts.

Exact call admission succeeds; one fewer allowed call and one fewer package must
refuse with exit 2, empty stdout and a diagnostic. Changing source without
refreshing the lock must refuse with exit 7 and preserve all workspace bytes.
After independent current-source preparation refreshes the lock, querying the
same SDK subject must match the refreshed Go context. Successful, cutoff, stale
and refreshed queries all require workspace immutability.

The shared test helper now accepts explicit source/SDK subjects and verifies that
all requested root declarations were found. Its ordinary SDK fixture retains the
existing query and failure checks. No product source or default behavior changes
are needed beyond the composition correction at `2b1a2d2a`.

## Commands and results

The mixed native command passes both Windows profiles in 509.294s: low 421.52s,
optimized 75.94s. Each profile performs four initial subject queries, two cutoff
refusals, a stale-source refusal and a refreshed successful query. All result
comparisons and workspace immutability checks pass. The ordinary SDK optimized
control passes in 74.990s (profile 62.10s), retaining its original SDK/leaf queries,
limits and stale/relocked behavior. No selected test is skipped.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasSDKUserAggregateCommand$'
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateAtlasSDKCommand$/^optimized$'
go vet -mod=readonly ./internal/cli
```

Scoped vet, Go formatting and diff checks pass. Logs use
`.cache/atlas-sdk-aggregate-impact{,-ordinary,-vet}-20261105.log`.

The new verifier is registered in the existing full authored-main acceptance,
which is not rerun for this command-leaf slice. Product source and the example
lock are unchanged, so the source checks and producer regressions at `2b1a2d2a`
remain valid. Mixed read-only MCP, body edits, full authored-main, final whole
compiler comparisons, consecutive self-build and Windows distribution acceptance
remain required. Native Linux stays batched. Hosted same-commit execution awaits
an authorized push/dispatch. No grammar, public CLI, ABI, map wire, DB, dependency,
timeout, target, repository hygiene or CI runner change. No push or dispatch.
