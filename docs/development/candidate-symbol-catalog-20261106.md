# Fairy symbol catalog implementation, 2026-11-06

This is part of the existing complete implementation queue. It does not replace
that objective with a parser-only or v3-only goal. Native Linux execution stays
batched; no push, dispatch or release is authorized by this record.

## Required sequence

- [x] Exact `atlas-symbols list` argument selection with Go parser comparisons.
- [ ] Current-admitted declaration selection, canonical JSON/checksum and exact
  selection/output budgets for catalog v1, v2 and v3. Preserve full-ID ordering,
  filtered-set cursor membership, original generic declarations and omissions.
- [ ] Schema-specific current-source map production. The existing Fairy wire
  owner admits v1/v3 only and the producer emits v3; v2 requires its own wire,
  semantic/current-source and source-production admission. Do not relabel v3
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
