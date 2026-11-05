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
2. **Actual Fairy production remains open.** Its declaration collector currently
   refuses `syntax::closure_prepared` sources. Use the independent Go fixtures
   before changing that admission; preserve actual private helper names, capture
   parameters and exact call/source identities rather than inventing declarations
   or dropping helper calls. Generic helper names and original template facts
   require separate concrete-owner binding and current-source verification.
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
