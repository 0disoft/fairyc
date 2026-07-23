# Atlas Phase 4 exit evidence

- Status: Current Stage 0 exit gate satisfied; later Atlas capabilities remain separately gated
- Exit gate source: `docs/product/01-roadmap.md`
- One-way boundary decision: `docs/adr/0203-enforced-one-way-atlas-import-boundary.md`
- Scope: current repository `main`, ordinary Go test lane, and the separately documented Windows
  smoke lane

## Claim boundary

Phase 4 ends when four narrow properties are executable:

1. Atlas failure cannot become compiler semantic input;
2. stale or incompatible maps fail closed;
3. unknown graph edges remain explicit; and
4. attributed text cannot grant authority.

Passing this gate does not enable `impact`, semantic `diff`, stable anchors, MCP, structural edits,
dependency-map joins, a decoder, persistence defaults, model calls, or a stable Atlas CLI. Those need
real consumer and completeness evidence under the later capability gates.

## Evidence matrix

| Exit condition | Owning oracle | Current evidence | What it does not prove |
| --- | --- | --- | --- |
| Compiler semantics do not depend on Atlas | `internal/architecturecontract` | `TestRepositoryKeepsAtlasImportsOneWay` parses every Go file below `cmd/` and `internal/` and permits direct Atlas/export imports only inside the export layer, `internal/cli`, and the Windows evidence harness. `TestAtlasImportBoundaryDetectsForbiddenImportsAndKeepsAllowedAdapters` requires exact violations for injected compiler and MIR imports. | A broken Atlas package can still prevent the statically linked repository from building; package initialization and process/plugin isolation are not covered. |
| Stale or incompatible maps fail closed | `internal/atlasmap` current-source/query/store lanes | `TestMapStoreRejectsStaleIncompatibleCorruptAndConflictingGenerations`, `TestValidateCurrentRejectsEveryStoredSourceIdentityAxis`, and `TestQueryIndexRejectsInvalidPartitionsSelectorsRequestsAndBudgets` reject exact source, file, line-map, compiler, interface, corruption, and contradictory query state rather than serving a best effort result. | Cross-version migration, remote map acquisition, physical power-cut durability, and a default store policy are absent. |
| Unknown graph edges remain explicit | `internal/semanticsnapshot`, `internal/atlasmap`, and `internal/contextpack` | Snapshot completeness retains unavailable/partial reasons. `TestQueriesExposeUnjoinedDependencyTargetsWithoutGuessing` requires dependency-map omissions and retained external relations; show/around tests require incoming-package omissions. Context Pack tests preserve those omissions and completeness fields. | Dependency-map joins, workspace-wide reverse callers, indirect-call recovery, or a complete impact graph are not implemented. |
| Text cannot grant authority | `internal/attributedtext` and `internal/evidencebundle` | Model tests fix every text record to `untrusted-text`; exact-source and documentation fixtures retain printable prompt-like commands only as data. Evidence Bundle tests require wrapper `none`, fact `compiler-fact`, and source/documentation `untrusted-text`, while facts and child bytes stay separate and checksummed. | A downstream host that ignores the labels is outside the compiler boundary; this is not a model-security, authentication, or tool-sandbox proof. |

## Oracle separation

The four rows are independent. A canonical checksum is not authentication. A passing import scan is
not stale-map evidence. A text label is not proof of a downstream consumer's permission handling.
Native Windows smoke is broad regression evidence for the compiler and target pipeline, but it is
not substituted for these cheaper stage-local oracles.

The ordinary Go test lane is the change gate for all four properties. Hosted Ubuntu and Windows Go
tests execute the same architecture and Atlas package tests through `.github/workflows/frontend.yml`;
only an observed exact-commit workflow result is hosted evidence. Local success alone is not a claim
that hosted CI passed.

## Reopening Phase 4

Reopen this gate when any of the following changes:

- an Atlas/export package or permitted adapter is added, removed, split, or moved;
- stable compiler commands begin consuming persisted Atlas data;
- map currentness, compatibility, completeness, omission, authority, or origin semantics change;
- attributed text enters Context Pack compiler-fact fields;
- a decoder or untrusted Atlas input is admitted; or
- an optional process, plugin, service, or model boundary is introduced.
