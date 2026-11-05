# Fairy symbol catalog implementation, 2026-11-06

This is part of the existing complete implementation queue. It does not replace
that objective with a parser-only or v3-only goal. Native Linux execution stays
batched; no push, dispatch or release is authorized by this record.

## Required sequence

- [x] Exact `atlas-symbols list` argument selection with Go parser comparisons.
- [x] Explicit canonical v2 map framing with unchanged v1/v3 reader selection.
- [ ] V2 semantic/current-source admission before a catalog can trust a map.
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
