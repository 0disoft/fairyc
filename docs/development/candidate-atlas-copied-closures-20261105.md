# Fairy Atlas copied-closure source family, 2026-11-05

Scope: carry the existing bounded copied-local-closure family through actual
Fairy Atlas production/current-source admission/join and impact/MCP/ordinary
root-body editing. This does not introduce first-class, retained, asynchronous
or cross-thread closures. The full implementation/final queue stays active;
native Linux remains batched and no remote publication is authorized.

## Sequential work

1. **Go source prerequisite fixed and verified.** Five independent v3 sources
   cover copied mutable captures, zero captures, four captures plus four
   parameters, module-wide helper numbering across two owners, and two concrete
   generic owner implementations. Original closure source spans, private scalar
   helper shapes, exact call endpoints and canonical map roundtrips are retained.
   The generic case exposed a Go snapshot failure: two specialized helpers have
   different declaration names but the same written closure span, causing
   `semantic snapshot documentation owner is duplicated` before a map could be
   prepared. Documentation now joins the complete package/module/kind/name key
   and exact span. Shared-span names stay distinct; wrong names/modules/spans,
   duplicate exact declarations and repeated attachments still reject. The
   existing lookup and snapshot representations do not change.
2. **Actual Fairy production implemented and verified.** Complete-source
   declaration collection now admits prepared closures, while the old scalar-only
   collector remains closed. The new `project_atlas_closures` projection binds
   written helper spans to checked pure helper frames and concrete owner frames.
   It emits ordinary owners first and then g0-sorted concrete owners, with one
   module-wide helper ordinal matching Go. Unused generic templates emit no
   helper. Exact private names are shared by declaration/symbol hashing, function
   summaries and call targets; executable IR and written source stay unchanged.
   Seven independent Go fixtures include mixed ordinary/generic owners and an
   unused generic owner. All pass native Windows low/optimized production,
   current-source admission and complete joins in 134.22s. Work exhaustion and
   missing coordinates reject in both profiles. The full combined ordinary,
   generic, static-operation and closure production replay passes in 462.853s
   without selected skips, including ordinary 81.41s, generic 120.80s and static
   operation 126.27s. Actual command acceptance remains the next gate.
3. Exercise impact, real read-only MCP and permitted ordinary root-body edits,
   including closure changes and freshness/refusal/publication controls. Update
   full authored-main registration and replay after the actual source family works.
4. Keep other Atlas families, final Go/Fairy comparison, consecutive self-build,
   distribution/clean-machine, native Linux and hosted same-commit gates separate.

## Go prerequisite evidence

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/semanticsnapshot
go test -mod=readonly -count=1 -timeout 3m ./internal/atlasmap
go test -mod=readonly -count=1 -failfast -timeout 2m -v ./internal/cli -run '^TestAtlasCopiedClosureSourceOracle$'
go vet -mod=readonly ./internal/semanticsnapshot ./internal/cli
```

Windows results: snapshot package 0.145s; map package 2.059s; all five CLI source
oracles pass without skips in 0.370s. The snapshot owner test includes five
explicit refusal mutations. Scoped vet, Go formatting and diff checks pass.
The initial generic-source oracle failure is retained in
`.cache/atlas-copied-closure-generic-diagnostic-20261105.log`; final logs are
`.cache/atlas-documentation-owner-20261105.log`,
`.cache/atlas-documentation-owner-map-20261105.log`,
`.cache/atlas-copied-closure-oracle-final-20261105.log` and
`.cache/atlas-documentation-owner-vet-20261105.log`.

No Fairy product source, grammar, public CLI signature, snapshot/map wire,
SDK/runtime ABI, DB, dependency, target, timeout, hygiene or runner change in
this prerequisite. This Go correction is not Fairy/native closure Atlas evidence.

## Production validation

The old blanket refusal returned `101`. After opening only Complete-source
collection, ordinary helpers worked but the generic fixture exposed one
`_fy_closure_choose_1` declaration where Go requires two distinct concrete-owner
helpers. The projection corrects that reproduced product gap. A separate harness
error compared an in-memory Go partition to a decoded one; the final test uses
canonical v3 codec roundtrips on both sides, as the existing producers do.
Intermediate compilation failures in the new implementation are not passes.

The expanded Go source oracle passes all seven cases in 0.522s. Its ordinary-
before-concrete and unused-template expectations are independent of the Fairy
projection. Source `check` and `check --locked` pass with 642 source files and
8,103 MIR functions; the lock is refreshed for the added source module. Scoped
Go vet, formatting and diff checks pass. Actual impact/MCP/edit command and new
full authored-main image acceptance remain separate pending work at this point.

```powershell
go test -mod=readonly -count=1 -timeout 2m -v ./internal/cli -run '^TestAtlasCopiedClosureSourceOracle$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlas(Production|GenericProduction|StaticOperationProduction|CopiedClosureProduction)$'
go vet -mod=readonly ./internal/cli
```

Logs: `.cache/atlas-copied-closure-oracle-expanded-20261105.log`,
`.cache/atlas-closure-production-regression-20261105.log`,
`.cache/atlas-copied-closure-source-check-20261105.log`,
`.cache/atlas-copied-closure-source-locked-20261105.log` and
`.cache/atlas-copied-closure-vet-20261105.log`.

This is a private source projection, not new closure syntax, first-class or
retained callbacks, public CLI/SDK ABI, map wire, DB, dependency, target, timeout,
runner or repository-policy change. Native Linux remains batched; no push,
dispatch, release or deployment.
