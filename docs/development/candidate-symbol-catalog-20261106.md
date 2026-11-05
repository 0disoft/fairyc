# Fairy symbol catalog implementation, 2026-11-06

This is part of the existing complete implementation queue. It does not replace
that objective with a parser-only or v3-only goal. Native Linux execution stays
batched; no push, dispatch or release is authorized by this record.

## Required sequence

- [x] Exact `atlas-symbols list` argument selection with Go parser comparisons.
- [x] Explicit canonical v2 map framing with unchanged v1/v3 reader selection.
- [x] Explicit v2 semantic admission with function-only generic declarations.
- [x] V2 independent current-source admission before catalog queries.
- [x] Current-admitted declaration selection, canonical JSON/checksum and exact
  selection/output budgets for catalog v1, v2 and v3. Preserve full-ID ordering,
  filtered-set cursor membership, original generic declarations and omissions.
- [x] Schema-specific current-source map production from authenticated Complete
  facts, including ordinary SDK modules and used/unused SDK Vec source. The
  explicit v1/v2/v3 factories retain whole-source family checks; no relabeling or
  conversion of maps/catalogs. The SDK and ordinary regression evidence follows.
- [x] Actual authored dispatcher, help, stale-source/lock and failure-atomic
  command acceptance for the admitted source families under both Windows profiles.
- [ ] Complete current-source full CLI replay across all registered verifiers.
  The scoped symbols replay below covers five of thirty-five registrations.
- [ ] Consecutive self-build, distribution, same-commit hosted and deferred
  native Linux acceptance from the original queue.

## Argument selection

`project_catalog_options.fy` accepts the existing required UID, explicit nonempty
kind set and caller limits, optional exact module/qualified name, canonical full
cursor, version and lock flag. It performs no filesystem, network or stdin I/O.
Repeated scalar flags/switches, repeated kinds, unknown/equal-sign flags,
missing/flag-shaped values and noncanonical/overflowing numbers reject before
any workspace can be read. Five kinds retain canonical lexical order in a
single bounded bit mask. Default and explicit version 1 stay distinct from
explicit versions 2 and 3 in the selected request.

`TestCandidateCatalogOptions` compares every selected field with Go's actual
`parseAtlasSymbolsOptions`: 105 argument cases and three invalid actions per
profile, 216 native processes across Windows low/optimized. Coverage includes
the full 31-argument request, 256/257-byte UID, 511/512/513-byte module/name,
65,535/65,536/65,537 symbols, 1 GiB minus/exact/plus-one byte limits, all five
kinds, qualified names, full/visible/uppercase cursors and decimal overflow.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateCatalogOptions$'
```

The final command passes in 33.237s with no selected skips. Log:
`.cache/atlas-symbol-catalog-options-20261106.log`. Earlier preparation attempts
failed on a missing test-only writer dependency, unsupported resource-field
mutation/fixed arrays and a multiple-static-origin return. The final code uses
owned local buffers, a scalar kind mask and slices of one static kind table;
no compiler admission rule or product contract was weakened.

The authored workspace also passes `go run -mod=readonly ../../cmd/fairyc check`
and `check --locked` from `examples/lexer_candidate`: 644 source files and
8,125 verified MIR functions, with the tracked source lock refreshed. Scoped
CLI vet, Go formatting and diff checks pass. The native parser probe covers the
selected module closure; these checks do not attest a rebuilt full CLI image.

This proves request selection only. It does not attest catalog bytes, data
production, actual CLI dispatch, lock behavior or any final queue gate. No
public CLI/SDK ABI, dependency, DB, workflow or repository policy changes.

## Explicit v2 framing

`project_atlas_wire::read_v2` selects only exact `fairyc.fymap/v2` plus its
matching encoding and logical model. The existing `read` keeps v1/v3 and rejects
v2. Both share canonical JSON shape/string checks, byte/work limits and the
domain-framed map checksum. V2 rejects aggregate/cleanup specialization bodies;
this transport owner grants neither semantic validity nor current-source Ready.

`TestCandidateAtlasV2Wire` uses four independent Go `EncodeV2`/`DecodeV2`
ordinary/generic package inputs. Nineteen framing cases plus four boundary
modes per Windows profile pass all 46 native executions: exact input bytes
survive unchanged; truncated, duplicate/unknown, resealed wrong-version,
noncanonical, corrupt-checksum, empty-added-hash and aggregate-body inputs
reject without output. Exact-size input succeeds; one-short bytes, exhausted
work and the legacy entry point reject.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateAtlas(V2Wire|Wire)$'
```

This command passes in 42.832s with no selected skips (v2 25.96s; unchanged
v1/v3 wire regression 16.67s). Log: `.cache/atlas-v2-wire-20261106.log`.
The unchanged v3 generic wire root passed both profiles in 25.24s in the first
combined attempt, whose overall result failed at the new fixture's generic-hash
seed precondition. That test-only seed now selects an actual generic package;
the valid v3 result is reused, not counted as a full combined-command pass.
Log: `.cache/atlas-v2-wire-before-20261106.log`. An earlier Go test build failed
on a test-helper argument mismatch before native execution.

The final authored workspace passes `check` and `check --locked`: 644 source
files, 8,127 verified MIR functions, refreshed source lock. Scoped CLI vet,
formatting and diff checks pass. No refreshed full authored-main image, complete
repository suite, native Linux or hosted CI execution is claimed for this slice.

V2 semantic/current-source admission, all-version source production and actual
catalog command selection remain open. This does not complete `atlas-symbols`
or any broader implementation/final acceptance family.

## V2 semantic admission

`project_atlas_validate::validate_v2` accepts only v2 documents and shares the
existing key/span/SymbolId/function/specialization/completeness checks. Generic
function signatures retain bound parameter indices; v2 rejects generic
aggregate declarations, nominal type arguments and binders in fields/variants.
The ordinary and v3 factories reject v2 explicitly. Existing v3 aggregate
declaration admission is unchanged; no current-source Ready is created here.

`TestCandidateAtlasV2Validation` compares six independent Go v2 package maps
from ordinary, used-generic and unused-template sources. Resealed mutations
are checked against Go `DecodeV2` before native execution: invalid/missing
IDs/owners, arity/types, duplicate bindings/order, observed-body provenance,
local implementation endpoints, counts and omitted template uncertainty.
All thirteen admitted scalar names are represented across original/modified
bindings. Snapshot-wide completeness counts remain allowed. A valid unused v3
aggregate source, framed as v2 for the negative probe, rejects semantically.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 7m -v ./internal/cli -run '^TestCandidateAtlas(V2Validation|GenericValidation|Validation)$'
```

The final combined command passes in 76.048s with no selected skips: v2 19.05s,
v3 generic/aggregate 26.92s, ordinary 29.86s. The v2 root executes 41 cases and
three explicit entry/work-budget controls per profile, 88 native processes.
Log: `.cache/atlas-v2-validation-20261106.log`. The first attempt failed before
native compilation on an inferred integer type in the new field-arity branch;
explicit `usize` fixes it without changing admission policy. Its log is
`.cache/atlas-v2-validation-before-20261106.log`.

The authored workspace passes `check` and `check --locked` with 644 sources and
8,128 MIR functions; the tracked source lock is refreshed. Scoped CLI vet,
formatting and diff checks pass. The current-source test is a subsequent slice,
not evidence supplied by these semantic checks.

Independent v2 current-source admission, catalog selection/encoding, all-version
production, actual dispatcher and all final acceptance gates remain required.

## V2 current-source admission

`project_atlas_current::read_v2` owns explicit v2 framing and function-generic
semantics before comparing the independent caller expectation. Its returned
`Ready` retains the checked partition only after all compiler, ordinary/generic
interface, language, UID/instance/coordinate/source/kind/runtime, feature,
dependency/reference and file facts match. The shared work ledger remains
bounded. Existing ordinary/v3 current readers do not discover or reinterpret v2.

`TestCandidateAtlasV2Current` uses the same six independently checked source
packages, never expectations learned from the map under test. It covers every
compatibility field, file/reference inventory, malformed expectations, exhausted
work and cross-version readers. A real unused public-template body edit leaves
the ordinary interface unchanged but changes the generic/source identities;
the old map rejects under both Go `NewQueryIndexV2` and the Fairy current reader.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 7m -v ./internal/cli -run '^TestCandidateAtlas(V2Current|Current|GenericJoin)$'
```

The combined command passes in 157.111s with no selected skips: v2 current
74.91s (129 native processes per profile, 258 total), ordinary/v3 current 30.99s,
and the existing complete generic/aggregate current-source join/impact 50.86s.
Initial log: `.cache/atlas-v2-current-crlf-20261106.log`; the canonical-checkout
follow-up below supersedes this input snapshot for the final working tree.

### Canonical-checkout follow-up

The pre-commit audit found CRLF working bytes in the changed current module and
ten otherwise clean authored files, while Git stores LF (`eol=lf`). Source
digests intentionally preserve raw bytes, so a lock computed from those working
bytes would not describe a clean Git checkout. Those files now use the LF bytes
already stored in Git. The ten otherwise clean files have no Git content diff;
no grammar or compiler policy changes. The source lock is refreshed below from
the canonical working tree.

The two earlier current commands passed against their respective working bytes:
157.111s before normalization, then 94.202s after normalizing the changed module
alone. Their logs are retained as `.cache/atlas-v2-current-crlf-20261106.log` and
`.cache/atlas-v2-current-single-lf-20261106.log`; neither attests final all-LF input.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidate(Atlas(V2Validation|GenericValidation|Validation|V2Current|Current|GenericJoin)|CatalogOptions)$'
```

All seven roots pass on the final all-LF source in 134.914s with both Windows
profiles and no selected skips: current 20.47s, generic join 29.27s, generic
validation 23.04s, v2 validation 14.29s, v2 current 19.86s, ordinary validation
13.83s and catalog options 14.00s. Log:
`.cache/atlas-v2-canonical-acceptance-20261106.log`. The v2-specific/current/parser
matrices retain 88, 258 and 216 native processes respectively; older regressions
are additional coverage, not included in those counts.

Final canonical-source `check` and `check --locked` pass with 644 files and
8,129 MIR functions. Scoped CLI vet and Go formatting remain valid on unchanged
Go inputs; final diff checks pass. All tracked authored source now has matching
LF index/working-tree endings. No actual full CLI image, complete Go suite,
native Linux, self-build, distribution or hosted-CI acceptance is claimed here.

This completes v2 current-source partition admission only. Catalog page selection
and checksums, schema-specific actual source production, new v2 downstream
query/command acceptance and final self-build/distribution/Linux/hosted gates
remain open. Historical v1/v3 join acceptance does not attest v2 joins or output.

## Current-admitted catalog pages

`project_catalog::encode` accepts an independently current-admitted `Ready` and
the parsed exact catalog request. It requires the same map/catalog version and
package UID; it performs no workspace, cache, network or publication I/O. Exact
module/name/kind filters define both the total and cursor membership before any
page is selected. An absent module fails; an absent name is a complete empty
page. The validated symbol table already has strict full-ID order and one
original declaration per symbol. Two linear passes preserve that order, with
constant-time declaration projection through the existing checked association;
no additional sort, specialization cloning or repeated declaration scan occurs.

`project_catalog_projection.fy` retains the exact Go query/result field layout,
including owner-qualified names, public/unsafe/foreign flags, generic-interface
context, selected generic completeness and template-body uncertainty. Selection
cost uses the mandatory context/filter record plus one separator byte and each
admitted entry plus one separator byte. Go query nil arrays remain `null` for
this cost calculation; final top-level context arrays normalize to `[]` exactly
as the Go encoder does. Empty completeness Reasons remain `null`. Budget
omissions report the exact remaining count and continue after the last selected
full ID; selection that cannot fit the first record fails without output.

`project_catalog_json.fy` shares the canonical string quoting and bounded work
owner, but uses the catalog CLI's 1 GiB maximum rather than inheriting the map
writer's unrelated 4 MiB ceiling. Encoding constructs one canonical compact JSON
value with no trailing newline. V1/v2/v3 checksums use their own domain version
bytes, schema/encoding and length-prefixed logical payload. An exhausted work
ledger or final byte limit exposes no partial buffer. This writer does not
convert map schemas or attest source production for a requested version.

`TestCandidateCatalog` derives six package partitions from three real source
workspaces, with independent current expectations and Go query/encoder oracles.
It compares every successful output byte and independently recomputes checksum
framing. The Windows low/optimized matrix runs 292 native processes per profile,
584 total: all five declaration kinds, multiple modules, equal ordinary and
owner-qualified leaves, case-sensitive absent names, used/unused function and
aggregate templates, one-entry pagination including after the last symbol,
absent/out-of-filter cursors, mandatory/first-entry/full/empty selection limits
and final output limits at one-short/exact/one-more boundaries. Stale expectations,
corrupt maps, exhausted work, wrong UID and encoder-level cross-version requests
reject without stdout. The cross-version case first admits the map under its
actual schema and then asks the encoder for a different version.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateCatalog(Options)?$'
```

The final command passes in 69.969s with no selected skips: catalog pages 56.39s,
unchanged request-parser regression 13.46s. The parser adds 216 native processes,
so the combined command runs 800. Log: `.cache/atlas-symbol-catalog-20261106.log`.
The initial page-only replay passed in 53.045s, before strengthening the encoder's
own version-refusal probe; it is retained at
`.cache/atlas-symbol-catalog-before-20261106.log`, not counted again. Initial
authored checking rejected a new direct u32-to-u8 cast before any native run;
the implementation now indexes the three admitted checksum domain bytes.

Authored `check` and `check --locked` pass with 647 source files and 8,145 MIR
functions; the tracked lock reflects final LF source bytes. Scoped CLI vet,
Go formatting and diff checks pass. Source-index and execution-queue progress
are updated. No public CLI/SDK ABI, dependency, DB, workflow or repository-policy
changes; no full authored-main rebuild, full Go suite, self-build, distribution,
native Linux or hosted CI acceptance is claimed. Native Linux remains batched.

Next are schema-specific production from authenticated Complete source facts and
actual `atlas-symbols list` dispatch/help/lock/command acceptance. `atlas-calls`,
`atlas-pack`, the broader source families and all original final gates remain
open. This slice closes only the catalog page/output owner.

## Schema-selected Complete source generation

The explicit `project_atlas_produce::package_version` factory obtains its program
from the supplied Complete, reauthenticates current source/signatures/coordinates
through `project_atlas_source_version::admit`, and checks the whole source closure
before creating a versioned package. V1 rejects every generic declaration; v2
admits function generics and rejects generic aggregate declarations even when
unused or owned by a dependency. Existing package factories retain v3.

Version-specific facts are computed from the retained Complete interfaces and
source-specialization inventory. V1 derives only the ordinary interface. V2
retains generic interfaces for every package only when the checked source closure
has templates, matching Go's ordinary versus generic workspace selection. V3
retains its explicit generic source interface even for ordinary input. A package
factory rejects facts whose generic-identity presence contradicts that selection.
No generated map is stripped, relabeled or reinterpreted as another version.

The header and envelope writers render the selected schema directly. Ordinary v1
omits generic interface fields and the two generic completeness relations; v2/v3
preserve original declarations and separate observed specializations. Checksums,
counts, omissions and independent current expectations belong to that same
selection. `produce::ready` selects explicit v2 current admission; the original
v1/v3 readers keep their prior selection. All output remains private until the
full encode/current/query path succeeds.

`TestCandidateAtlasSourceVersion` compiles actual retained workspaces with Fairy,
produces the selected maps, runs Fairy current admission and joins, and renders
the selected source-backed catalog. Its Go source oracles separately check
declarations, SymbolIds, ordinary/generic interfaces, implementation identities,
aggregate layouts and observed function/cleanup calls. Go also admits every map
against independent source/file expectations and the independently hashed native
compiler image. The catalog bytes/checksum are compared with the Go writer over
that admitted map; this does not claim that the Fairy non-call fact collector
has Go's broader coverage. Its omissions remain explicit.

The source matrix includes ordinary foreign/resource/enum/type families,
owner-qualified methods beside an ordinary same-leaf function, used and unused
function templates, used aggregate templates, and unused dependency-owned struct/
resource templates. Negative controls exercise exhausted source admission work,
missing coordinates and contradictory ordinary/generic fact selection.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$'
```

The six-family matrix passes in 288.017s with both Windows profiles and no
selected skips: low 201.28s, optimized 70.52s. Each profile executes 23 native
processes and admits/compares 24 maps, 46 processes and 48 maps total. There are
24 successful source/current/join/catalog paths, twelve unsupported-family
refusals, and ten work/coordinate/contradictory-facts controls across the profiles.
Log: `.cache/atlas-source-version-20261106.log`. No existing full authored-main
image or native Linux acceptance is supplied by this private production probe.

The initial harness failed before native execution on a missing fixture-module
closure, then on untyped fixture locals. The first native comparison used raw Go
memory against codec-roundtripped declarations; it now compares both through the
correct version's canonical codec. A subsequent negative probe used a broken-stream
helper instead of stdin transport; it now uses the existing actual stdin process
helper. These failures are retained separately at `atlas-source-version-before`,
`-integration`, `-native` and `-canonical` logs under `.cache/`, each suffixed
`-20261106.log`. They do not establish product failures.

The next replay exposed an actual v2 implementation mismatch: ordinary source
was incorrectly given the v3 lane's always-present generic interface identity.
The final producer follows the checked global template inventory and matches the
independent Go source expectation for every package. This failed replay is
`.cache/atlas-source-version-ordinary-generic-before-20261106.log`; the successful
final matrix above supersedes it.

An additional explicit SDK source attempt stopped at the test helper's one-minute
process cap on low `sdk-unused/v1` (exit 1, empty output, 60.01s). It did not reach
either SDK family acceptance. The existing SDK-specific helper uses a separate
three-minute diagnostic cap, documented in
`docs/ops/windows-candidate-atlas-sdk-vector-20261105.md`. That attempt is retained
as `.cache/atlas-source-version-sdk-20261106.log`; it is not a pass or a product
timeout change. SDK-specific early refusal/production/current/catalog acceptance
is the next independent slice. The six-family successful inputs and product code
were unchanged by that test-only fixture addition/removal, so their valid result
is retained without repeating 46 native processes.

Authored `check` and `check --locked` pass with 648 source files and 8,153 MIR
functions; the lock reflects final LF bytes. Scoped CLI vet, Go formatting and
diff checks pass. Source index and execution-queue progress are updated. No public
CLI/SDK ABI, DB, dependency, repository-policy or runner/workflow changes. Full
repository Go tests, legacy full command images, self-build, distribution and
hosted CI remain later gates; native Linux stays batched. No push or dispatch.

## SDK source selection and production

`project_atlas_source_version::select` provides refusal-only bounded source
preflight before expensive Complete construction. A successful selection grants
no current-source or map authority: `admit` still reauthenticates the exact
Complete/source/signatures/coordinates and repeats the same family checks at
publication. Unsupported templates anywhere in the closure still reject.

The SDK probe uses the existing SDK-specific three-minute diagnostic process
limit; no product timeout changes. Both used and unused `std::vec` sources reject
v1/v2 and produce independently admitted v3 maps/catalogs. The replay passes in
304.204s (test body 304.05s), with twelve native processes, twelve independent
maps and eight unsupported-family refusals over both Windows profiles. Low
unsupported-family refusals finish in 0.25..0.37s, whereas the earlier attempt
exhausted the generic helper's one-minute limit before a verdict. Log:
`.cache/atlas-source-version-sdk-preflight-20261106.log`.

The separate final preflight regression includes all six original families and
ordinary SDK `std::num`/`std::bytes` under v1/v2/v3. It passes in 462.263s (test
body 461.74s; low 364.30s, optimized 77.98s), with 52 native processes and 66
independent maps. Together the two final disjoint selections cover nine source
families, 64 native processes and 78 independent maps. The SDK-scalar low cases
take 60.31..66.77s and use the existing diagnostic helper; they are not measured
against or evidence of a changed product timeout. Source/current/join/catalog
oracles, work/coordinate/contradictory-fact controls and workspace immutability
pass. Log: `.cache/atlas-source-version-preflight-regression-20261106.log`.

Authored `check`/`check --locked` pass with 648 source files and 8,154 MIR
functions. Scoped CLI vet, Go formatting, LF byte audit and diff checks pass.
This closes version-selected source production for the named families only.
The actual symbols dispatcher, full CLI, self-build, distribution, Linux batch
and hosted acceptance remain open. No public API/SDK ABI, dependency, DB,
workflow/runner or repository-policy changes; no push or dispatch.

## Actual authored symbols command

`project_catalog_source::query` freshly loads the complete workspace under the
requested ordinary lock policy, performs refusal-only version selection, builds
the actual ordinary/aggregate Complete, and obtains source schemas, observed
specializations and independent current expectations from that Complete. It
produces the requested family for every package before selecting the exact UID,
so a filter cannot hide unsupported declarations in another package. All
package call/external totals belong to that same checked workspace. The selected
package re-enters version-specific current admission before catalog encoding.
The existing bounded SDK source/query ledger separation remains explicit.

`project_catalog_cli::arguments` validates arguments before workspace I/O and
writes only the fully encoded result. Workspace/lock failures return 7; source
refusals return 3; selector/budget failures return 2; internal/output failures
return 4. Both namespace and `list` help requests use the existing public help.
`project_experimental_cli` now dispatches `atlas-symbols` directly to this Fairy
implementation, with no child Go compiler, map/cache lookup or map publication.
Unlocked queries refresh the ordinary lock; locked queries preserve all bytes.

The source-backed command oracle starts with independently checked Go source
partitions, exact declarations/IDs and interface/specialization facts. It
declares the existing narrower candidate collection policy, including empty
generic-specialization/unavailable-template-body classes for ordinary v2/v3,
without learning expected facts from a candidate response. Go strict canonical
decoding/current admission and the actual Go catalog writer then derive exact
output bytes, selection sizes, full-ID cursors and independently recomputed
checksums. Non-call collection is still explicitly unavailable; this is not a
claim of Go's broader non-call Atlas coverage.

The full authored-main verifier registers three additional groups in
`candidate_atlas_cli_test.go`: `catalog-ordinary`, `catalog-sources`, and
`catalog-sdk`. Ordinary requests cover all three versions, exact/absent names,
one-entry continuation, selection/final output at one-short/exact/one-more,
unknown package/module/cursor, argument rejection before I/O, broken stdout,
locked source drift and unlocked lock refresh. Separate source inputs cover
all five declaration kinds in each schema, owner-qualified names, used/unused
function templates and aggregate/resource templates. SDK inputs exercise
ordinary scalar v1 and used Vec v3. PATH is empty and no companion is installed.

Initial authored checking caught a missing switch-statement terminator before
native execution; it was corrected. Two checks invoked from the wrong working
directory failed before reaching the authored checker and supply no evidence.
The first Go command also failed at compilation on an unused test import; log:
`.cache/catalog-cli-before-20261106.log`. The first native full image exposed
missing command help; `.cache/catalog-cli-integration-20261106.log` records that
failure, fixed in the command owner. The next replay completed help and v1
queries but rejected the oracle's omission of ordinary v2 generic completeness;
`.cache/catalog-cli-command-20261106.log` records that failed expectation. The
corrected oracle explicitly derives those empty classes from the existing
candidate collection contract. Repeated byte/page checks now use a small real
workspace; all complex declaration families retain separate v1/v2/v3 checks.

The combined five-group low/optimized attempt hit its Go parent's fifteen-minute
limit during low generic source tests, after the ordinary command verifier had
checked all 44 invocations. It is a failed command, not full CLI acceptance:
`.cache/catalog-cli-final-20261106.log` (900.272s). No orphan child of that test
parent remained after termination. Final profile/group receipts are recorded
below; no product timeout or compiler warning policy was changed.

Final bounded commands reuse only an authenticated generated-C/profile/toolchain/
image digest. The low image has source ID
`007285e769ca0ba027e51335bbb0a3f584024c4c825d12813ff06ff74eb7b77c` and SHA-256
`097067cb377cdb4e8db5b4fbdebe1a3ac377ed9c066f6c26d5d71b634b5c9fb4`.
The optimized source/image IDs are
`d645e1cfd46d587c56e26449f192c367210a431e9e42cf1076cc43c4c1d7201b` /
`af2c9e435c5bd671d8b87889470a6b36b09a3c6bade3604b16759bc597082613`.
Each command freshly regenerates the C input before deciding whether reuse is
valid. The C input has 24 units, a 1,996,907-byte header, a 153,115-byte runtime
and a largest unit of 2,097,059 bytes. No stale source/image result is reused.

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/catalog-cli-20261106'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$/^catalog-ordinary$'
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$/^(catalog-sources|catalog-sdk)$'
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$/^(impact-ordinary|help)$'
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/optimized$/^(catalog-ordinary|catalog-sources|catalog-sdk|help|impact-ordinary)$'
```

| Final scope | Result | Log under `.cache/` |
| --- | --- | --- |
| Low ordinary command, 44 native invocations | PASS, 605.333s; assertions 585.41s | `catalog-cli-low-ordinary-20261106.log` |
| Low eight source inputs/five family refusals plus two SDK inputs | PASS, 282.344s; source 120.59s, SDK 140.50s | `catalog-cli-low-sources-20261106.log` |
| Low existing impact and ten help invocations | PASS, 172.973s | `catalog-cli-low-regression-20261106.log` |
| Optimized all five groups above | PASS, 179.152s | `catalog-cli-optimized-20261106.log` |
| Unchanged Go symbol command regression | PASS, 11.846s | `catalog-go-regression-20261106.log` |

The disjoint final profile/group selections verify 87 native CLI invocations per
profile, 174 total: 88 ordinary catalog, 26 source-family, four SDK, twenty help
and 36 existing impact observations. This covers five of thirty-five full-main
registrations, not the unselected thirty or generated consumers. The low SDK
inputs take 70.17s/70.33s under the unchanged SDK-specific diagnostic cap;
optimized takes 2.18s/2.60s. These are observations, not a product performance or
default-timeout acceptance claim. The failed combined attempt remains failed.

Authored `check`/`check --locked` pass with 650 source files and 8,159 MIR
functions. Scoped CLI vet, Go formatting, LF audit and diff checks pass. The
source index and original implementation queue retain the broader outstanding
work. This implements the existing experimental symbols surface in Fairy; no
stable CLI, grammar, SDK ABI, dependency, DB, workflow/runner or hygiene-policy
changes. Full Go tests, complete current full CLI, self-build, distribution,
native Linux and same-commit hosted acceptance remain open. Native Linux stays
batched; no push, dispatch or publication. Actual `atlas-calls` and `atlas-pack`
are the next command implementations; the full objective remains active.
