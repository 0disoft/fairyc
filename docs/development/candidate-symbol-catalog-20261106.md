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
- [ ] Schema-specific current-source map production. The existing default Fairy
  wire owner retains v1/v3 selection and the producer emits v3. The explicit v2
  framing/semantic/current boundaries above now exist; production must still
  derive each requested version from authenticated Complete facts. Do not
  relabel v3 maps or catalog output to simulate v1/v2 support.
- [ ] Actual authored dispatcher, help, stale-source/lock and failure-atomic
  command acceptance; then current full CLI under both Windows profiles.
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
