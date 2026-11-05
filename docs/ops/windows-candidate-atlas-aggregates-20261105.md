# Fairy aggregate Atlas production, 2026-11-05

Later follow-ups below supersede an earlier remaining-work statement only for
their explicitly verified scope. Command-leaf proofs do not attest the complete
authored CLI or other final queue gates.

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

## Aggregate impact command follow-up

`project_atlas_source_mode::aggregates` classifies exact retained source under
the shared work bound. The actual `experimental atlas-impact` leaf selects the
aggregate Complete/signature/instance factories together when a struct/resource
template is present. Ordinary/function-only source keeps its existing route.
Neither a selected symbol nor a map/cache header can select or upgrade the mode.

`TestCandidateAtlasAggregateCommand` passes Windows low/optimized in 144.66s
(76.73s/56.90s). Its three-package source contains two scalar Cell instances,
nested resources with declared effects and foreign cleanup, a zero-field resource
with empty cleanup, an unused template, and a generic cross-package cleanup
helper. It queries five original declarations plus the helper's full SymbolId,
compares exact Go v3 impact nodes and source/interface/current-image contexts,
and requires the resource cleanup caller to appear. Exact four-call admission,
three-call refusal, package cutoff, stale lock, relocked unused-template field
changes and workspace nonmutation are verified: 11 native invocations per profile.

The first low run failed only on a test assumption that changing unused aggregate
fields preserved the ordinary interface hash. A public aggregate header edit can
change that interface as well. The corrected verifier still compares every
ordinary/generic hash to independent Go and requires generic/source/instance
drift; it no longer invents a field-change identity contract.

```powershell
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlasAggregateCommand$'
```

The same verifier is wired into opt-in `TestCandidateAtlasCLI`; that refreshed
full authored-main/dispatcher replay remains pending. Standalone MCP aggregate
session and edit selection/overlay/application acceptance remain separate, as do
SDK, native Linux and the broader queue. The CLI's unsupported-declaration text
no longer claims that all generics are unsupported. This expands the existing
experimental CLI source family without a stable language/SDK ABI or wire revision.

The existing function-only generic command and SDK/copied-closure refusal
regression passes both profiles in 145.69s. Scoped vet, Go formatting and diff
checks pass. The aggregate command's successful-path source/arguments remain
unchanged after its native run; the following edit changes only the unsupported
declaration message and the opt-in Go full-CLI verifier registration.

## Aggregate MCP session follow-up

The existing read-only MCP query consumes the same exact-source factory
selection. `TestCandidateMCPAggregateCommand` passes both Windows profiles in
101.18s (low 35.55s, optimized 51.43s), using two real stdio processes and eight
tool requests. Each session initializes, returns Go-exact aggregate impact,
isolates an unknown package, refuses stale template source/lock, then recovers
after an explicit Go-oracle relock without restarting. Request IDs, current
image and package contexts, source/lock nonmutation and absence of host paths or
raw source in replies are checked. Pipes, deadline and child lifetime are owned
through failure and normal EOF.

```powershell
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestCandidateMCPAggregateCommand$'
```

The verifier is registered in the full authored CLI test; that refreshed full
replay remains pending until the remaining command integration is ready. This
adds native acceptance for the already shared query path, not another MCP tool,
permission, write operation, runtime ABI or runner. Aggregate editing, SDK,
native Linux and the complete final queue remain open.

## Aggregate body-edit follow-up

The actual edit selector now derives complete root declaration headers and
SymbolIds through the same Complete-bound schema used by the Atlas producer.
Original and edited source independently select their ordinary or aggregate
factory from retained source. Exact root ownership, one private safe ordinary
non-entry target, expected raw source hash, body extent and immutable overlay
verification remain required; generic/aggregate/public/entry declarations do
not become editable. Source publication retains the existing locked fresh
recheck, staged sync, final fingerprint/byte comparison and one rename.

`TestCandidateEditAggregateCommand` passes both Windows profiles in 182.45s
(low 100.29s, optimized 70.63s), with 22 native invocations. The root contains
generic declarations, an attributed noneditable main, and a private function
whose parameter is an imported concrete aggregate. Its replacement introduces
a new Cell<bool> specialization; the full edited factory must check it afresh.
Independent Go preview reports match exact source spans/hashes and verified
state. Native publication matches expected bytes and changes only the target
source, leaves no stage/lock files, and preserves fairy.lock. Invalid type,
unknown field, injected sibling, four noneditable subjects, stale lock and stale
source hash refuse without changing workspace bytes.

```powershell
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateEdit(AggregateCommand|Command)$'
```

The first simpler aggregate edit matrix also passed in 148.79s. The final
matrix adds the new specialization, attributed main and complete publication
tree comparison. The combined aggregate/ordinary edit command above passes in
578.002s; ordinary publication, fault and staged-source/dependency/lock controls
pass both Windows profiles (395.36s). Scoped CLI vet, Go formatting and diff
checks also pass. The same verifier is registered in the authored full-CLI test;
that refreshed complete image replay remains pending. This expands the existing
experimental edit workspace family, not its editable-target policy or wire ABI.

## Full-CLI C partition boundary

The refreshed authored CLI initially fails before native compilation in 17.022s:
one body packet has 2,097,164 bytes, twelve over the unchanged 2 MiB consumer
limit. The Go partition packer counted function bodies without their shared
header include. Packing now includes that prefix; it still preserves indivisible
function boundaries. A public-emitter regression covers one byte below, exactly
at and one byte above the actual two-function packing boundary for both target
profiles, without requiring a C toolchain.

`go test -mod=readonly -count=1 -timeout 5m ./internal/cbackend -run
'^Test(Partition|SmallBuildMatchesStreamingUnit)'` passes in 40.352s, including
the existing Windows low/optimized separate-file native executions. Scoped
backend vet and diff checks pass. The refreshed full-CLI attempt now emits 24
parts, largest 2,096,962 bytes; its complete command result remains pending.
No transport ceiling, per-command timeout, source admission or runner changes.

## Original producer development failures and remaining work

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

The follow-ups establish aggregate impact/MCP/edit command-leaf acceptance.
Refreshed full authored-CLI, SDK source production, aggregate check/build/run/test,
native Linux, consecutive self-build, latest
distribution/clean-machine use and same-commit hosted CI remain open. Native
Linux stays in the batched queue. No stable public API/SDK ABI, wire schema, DB,
dependency, default timeout/cache policy, target, runner or repository-hygiene
change. No push, release or deployment.
