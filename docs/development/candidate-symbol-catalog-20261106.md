# Fairy symbol catalog implementation, 2026-11-06

This is part of the existing complete implementation queue. It does not replace
that objective with a parser-only or v3-only goal. Native Linux execution stays
batched; no push, dispatch or release is authorized by this record.

## Required sequence

- [x] Exact `atlas-symbols list` argument selection with Go parser comparisons.
- [x] Explicit canonical v2 map framing with unchanged v1/v3 reader selection.
- [x] Explicit v2 semantic admission with function-only generic declarations.
- [ ] V2 independent current-source admission before catalog queries.
- [ ] Current-admitted declaration selection, canonical JSON/checksum and exact
  selection/output budgets for catalog v1, v2 and v3. Preserve full-ID ordering,
  filtered-set cursor membership, original generic declarations and omissions.
- [ ] Schema-specific current-source map production. The existing default Fairy
  wire owner retains v1/v3 selection and the producer emits v3; explicit v2
  framing above still needs semantic/current-source and source-production
  admission. Do not relabel v3
  maps or catalog output to simulate v1/v2 support.
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
