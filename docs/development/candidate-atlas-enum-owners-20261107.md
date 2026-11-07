# Enum-owned Atlas functions, 2026-11-07

Status: the existing Go-supported enum owner family is connected to Fairy Atlas
production and the actual impact command on Windows low/optimized. Linux stays
in the maintainer's combined batch. The broader implementation goal stays open.

## Source and ownership boundary

Go's `ownerSymbol` already includes enums, and `checkOwnerReceiver` admits the
same value/shared/exclusive receiver modes as for other nominal owners. Fairy's
`module_signatures` previously accepted only struct/resource owners. It now
retains enum nominal identity, creates its existing kind-4 type even for unused
associated functions, and verifies enum receivers against that exact module and
declaration. `module_unit_method_receiver` admits that enum kind when selecting
direct or borrowed nominal receivers, including inferred factory results.

Owner locality, nongeneric-owner and exact receiver identity checks remain in
place. Pre-value owner selection grants no borrow/mutation permission: ordinary
argument, type, Flow and IR verification still run. No grammar, runtime opcode,
Atlas schema, ownership rule, limit or language contract is expanded.

## Independent native observations

A disjoint Sol High worker owns only the new enum-oracle/producer test; the
coordinator owns production, command expectation migration and integration.

`TestAtlasEnumOwnerSourceOracle` passes ten positive and four negative current
Go Atlas v3 cases in 0.93s (package 1.057s). Each positive reaches actual map
preparation, rather than merely passing Go source checking. Cases cover unused
and called static functions, all three receiver modes, same-leaf functions in
two enum namespaces, external C calls, imported public owners and inferred
factory results. Go rejects the guessed `State::ready` expression syntax; the
fixtures use its admitted contextual variants/factories instead.

`TestCandidateAtlasEnumOwnerProduction` passes in 269.77s (package 269.904s).
Ten sources, copied-fresh/restored Complete and two packages produce forty maps
per profile, eighty total. Complete artifacts are really encoded, decoded and
readmitted against current source before restored production. Each profile's
fresh/restored output is byte-equal for the same source. Independent Go maps
supply declaration/symbol/interface, receiver, call, span and specialization
facts. File/module records and the running producer's SHA-256 are also checked.
All fourteen completeness rows retain their declared exact/partial/unavailable
meaning; the FFI case cannot become exact. Wrong/missing owners, unqualified
fallback, mismatched receivers and stale-source reuse refuse.

The previous enum-based unsupported-source control becomes a positive command
control. `TestCandidateAtlasEnumOwnerCommand` passes low/optimized in 117.83s
(package 117.981s): both current-Go-derived visible/full selectors succeed with
locked source, while an unknown selector fails with invocation code 2 and no
stdout. Workspace bytes remain unchanged. The full-main harness registers this
as `enum-owner`; generic, closure and owner-qualified leaf commands share the
same updated control. Their unchanged broader matrices are not rerun here.

Logs are `.cache/atlas-enum-owner-20261107.log` and
`.cache/atlas-enum-owner-command-20261107.log`. Initial oracle fixtures use
unsupported syntax; initial observer preparation also catches an inferred
numeric type and path-dependent resource ownership. A helper with an early
return preserves resource rules while performing fresh/restore selection. Those
attempts remain failed observations; no product rule or warning is relaxed.

The selected Windows commands are:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestAtlasEnumOwnerSourceOracle$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateAtlasEnumOwnerProduction$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateAtlasEnumOwner(FixturesHaveValidSource|Command)$'
go vet -mod=readonly ./internal/cli
```

The isolated current-source `check` and `check --locked` pass in 17.323s /
15.284s: 755 files / 9,542 MIR functions, empty stderr. All 1,226 authored files
and the copied lock match the snapshot. Go production/embedded SDK inputs are
unchanged, so the verified `fce79c8e772fccfd3956c2cc6e395fdb45b3ba9c318243816263aa0d88152fa5`
bootstrap is reused. Receipt:
`.cache/atlas-enum-owner-source-20261107-all-check.json`. Gofmt and diff checks
pass. No source-of-truth language/schema contract changes are required.

Full authored-main replay, wider source combinations, refreshed self-build and
distribution, the full Go suite and native Linux remain separate. No public
API/SDK ABI, DB, dependency, repository-policy or runner change; no push, hosted
dispatch, release, deployment or publication.
