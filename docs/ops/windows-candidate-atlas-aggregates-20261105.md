# Fairy aggregate Atlas production, 2026-11-05

## Scope

The explicit Complete-bound aggregate producer now retains original generic
struct/resource declarations, concrete aggregate layouts and nonempty cleanup
summaries in separate v3 specialization records. It uses the real checked
program, source identities and scalar argument tuples. Source-only projections
cannot supply executable authority. The ordinary/function-only factories retain
their admission boundaries.

`project_atlas_aggregate.fy` projects exact concrete field names, order, source
spans and types from verified body layouts. It checks layout/source ownership,
retains repr(C), and publishes resource drop kind and effects from verified
cleanup metadata. Structs have no function/cleanup body; empty resources have
no executable cleanup summary. Checked empty cleanup frames must still join the
exact original resource/type identity before omission. A nonempty cleanup frame
cannot be omitted by that rule.

The source header adapter retains struct repr(C) and explicit resource effects
without treating unused bodies as checked. Go snapshot support for those facts
is separately committed in `a0283e74`. The bounded instance projection is
committed in `7bba5379`.

## Validation matrix

`TestCandidateAtlasAggregateProduction` prepares eight three-package workspaces:
ordered/imported aggregates, unused templates, two scalar instances,
nested/deferred cleanup, partial construction error propagation, loop exits,
concrete repr(C), and a zero-field resource with empty cleanup. Independent Go
v3 preparation/encoding/decoding supplies the comparison partitions.

For both Windows MSVC low/optimized, the real Fairy probe:

- freshly checks and links source through the aggregate Complete factory;
- derives original declarations, public interfaces and SymbolIds;
- emits all package maps, revalidates each against independent current-source
  expectations and joins the complete workspace;
- returns the maps for strict Go `DecodeV3` and comparison of all original
  declarations, symbols, ordinary/generic interface hashes, concrete aggregate
  records, specialization IDs/arguments and ordered ordinary/cleanup calls;
- rejects exhausted work and missing coordinates without workspace mutation.

There are 48 native probe invocations and 48 returned package maps across the
two profiles. Other executable observations remain explicitly unavailable in
the producer's completeness records; this is not a claim of full Go snapshot
relation parity. PATH is empty during native replay, so no Go child supplies
the candidate's facts.

The final aggregate-production root passes in 246.69s (low 172.43s,
optimized 60.54s), including every authored scenario and refusal control.

The final combined command also exercises the unchanged ordinary map producer,
function-only generic producer/current reader/join and source-only generic
header projection:

```powershell
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlas(AggregateProduction|GenericProduction|GenericSchema|Production)$'
```

The combined command passes in 477.908s with no selected skips. The remaining
roots pass both profiles: generic-function production 123.16s, source-only
generic headers 29.03s, and ordinary production 78.87s. Scoped
`go vet -mod=readonly ./internal/cli`, Go formatting and `git diff --check` pass.

## Development failures and remaining work

Initial development runs failed on a Go test's comparison of a slice-containing
struct, a fixture that constructed a resource outside its owning module, and
the remaining function-only generic declaration collector. Those were corrected
before the final matrix. The first eight-case low run passed seven cases but
rejected the empty resource: it treated a checked empty cleanup frame as a
missing executable summary. Two focused diagnostic runs localized the refusal;
the exact-owner empty-frame fix then passed the focused low case in 45.49s.
These failed commands are not counted as acceptance passes.

The earlier instance projection passes both Windows profiles in 106.21s,
including four refusal controls per each of six source scenarios. Its ordinary
Atlas type/function-production regressions pass in 183.966s. This evidence does
not replace the final producer replay above.

Actual aggregate impact/MCP/edit command dispatch, SDK source production,
aggregate check/build/run/test, native Linux, consecutive self-build, latest
distribution/clean-machine use and same-commit hosted CI remain open. Native
Linux stays in the batched queue. No stable public API/SDK ABI, wire schema, DB,
dependency, default timeout/cache policy, target, runner or repository-hygiene
change. No push, release or deployment.
