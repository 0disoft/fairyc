# Grouped static operation metadata replay, 2026-11-07

Scope: the isolated `fairyc/.cache/grouped-static-20261107` checkout at baseline
`c5b6c9b430037de0b57cf1d0bc53a3fd520aa7f1`, following ADR 0549.
This worker owns only:

- `examples/lexer_candidate/src/static_operation_identity.fy`
- `examples/lexer_candidate/src/static_operation_group_identity.fy`
- `internal/cli/candidate_grouped_static_pipeline_test.go`
- This report.

No ADR, admission/catalog implementation, Go genericsource implementation, queue,
lock, main checkout, commit, remote, or Linux change is part of this worker's work.

## Implementation

The identity helper gathers at most sixteen declaration IDs, filtering both the
raw group label and the root declaration's `syntax::module_index`. It sorts
members by operation name and each member's providers by scalar result name,
then serializes one flat array with the existing original-function JSON writer.
The digest domain is exactly `source-static-operation-group-v1`. Validation
continues to belong to `static_operation_declarations::validate`.

Every member gets the same definition digest. Each member's metadata retains only
its own providers. Provider digests still use `source-static-provider-v1`, and
the member-specific binding still uses `source-static-bound-operation-v1` over
the existing package definition plus the new group definition. The ungrouped
serialization path and `source-static-operation-v1` domain remain intact.

## Replay design

The Go test calls real `genericsource.PreparePackages` on original root/helper
sources. It writes expected roots, specialization IDs/keys/type bindings/edges,
and public template closures. A separate raw-AST recipe computes the exact group,
provider and bound digests without consulting Go's private operation maps.
The generated Fairy fixture compares actual source catalog metadata and original
operation/provider JSON, then uses the existing source planner/public template
fixture to compare all expected identities.

The seven cases are base, selected provider body edit, unused sibling body edit,
unused sibling's unused bool body edit, unused sibling type-parameter rename,
unrelated private body edit, and legacy ungrouped declarations. The five grouped
mutations preserve source length. The first four relevant mutations change
dependent roots/closures; the private edit preserves roots, specializations and
public templates. The legacy case independently checks the old identity recipe.
Two separately parsed modules joined through `module_ast_join` share one label
and check module-local group identity. The same fixture also has a focused native
module-scope test in both existing `run` profiles.

The native source reuses every existing thirteen-scalar provider/observer body,
adds a complete sibling bundle, and uses the existing
`tests/static_operation_pipeline.fy` helper. That helper builds checked body and
program/project IR, serializes/decodes it, restores `core::Complete`, partitions
actual generated C, and exposes all thirteen transport signatures to the C
observer. Both low and optimized compiler producers run the same frozen source
copy and C parts. Each output is built and run with low and optimized consumers;
all four combinations must exit 42 with empty output.

## Executed checks and attempts

Preparation: `gofmt -w internal/cli/candidate_grouped_static_pipeline_test.go`,
`go vet ./internal/cli` with `GOMAXPROCS=1`, and
`git diff --check -- examples/lexer_candidate/src/static_operation_identity.fy`
passed.

Native command (also `GOMAXPROCS=1`):

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateGroupedStaticOperationPipeline$' -count=1 -timeout=15m -v
```

Attempt 1 failed after 51.00 seconds: all seven real Go oracle expansions passed,
the 104-part low producer built, then its actual Fairy execution exited 134
without stderr. This is a failed native check, not a pass or a preparation-only
result. Its log remains `.cache/grouped-static-pipeline-attempt1.log`.

Attempt 2 failed after 58.66 seconds. Diagnostic checkpoints proved that all seven
actual Fairy metadata/AST/planner/public-template cases passed before the failure.
The failure was this worker's module-scope fixture trying to parse two modules as
one source, rather than joining separately parsed sources. The fixture now uses
the real `module_ast_join::join` API. No production identity correction was needed
for this failure. Log: `.cache/grouped-static-pipeline-attempt2.log`. Frozen
source and low producer image remain in
`.cache/grouped-static-pipeline-2064979039/`.

The final replay follows the coordinator's grouped-only validation phase fix.
No source, work, runtime, or native stack limit has been widened.

The focused module-scope replay passed in 14.59 seconds (14.712 seconds package
wall time), including both ordinary and release/locked native runs:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateGroupedStaticOperationModuleScope$' -count=1 -timeout=3m -v
```

Log: `.cache/grouped-static-module-scope-attempt1.log`.
After the fixture correction, focused `go vet ./internal/cli`, read-only
`gofmt -l` and scoped whitespace checks of all four owned files passed again.
Native pipeline attempt 3 **passed** in 227.57 seconds (227.778 seconds package
wall time), with the corrected join fixture and the coordinator's final grouped
admission source frozen into its private copy. Log:
`.cache/grouped-static-pipeline-attempt3.log`.

- Both low and optimized producers: 104 C parts; all seven real source metadata,
  original AST JSON, planner and public-template oracle cases passed; joined
  module isolation passed; project/program/body restoration into `Complete`
  succeeded and produced actual partitioned C.
- low/low, low/optimized, optimized/low and optimized/optimized generated programs:
  exit 42, empty output; thirteen scalar transport signatures/values checked by
  the original C observer.
- No Go-expanded source was supplied to either Fairy producer, and no existing
  provider body was replaced. Both producers used one frozen source closure.

The successful attempt cleaned up its private source/image workspace; failed
attempt 2's workspace and all attempt logs remain. No owned process remains
running. No commit or remote action was made by this worker. Normative ADR/API,
DB, dependency, runner and wire-version contracts are unchanged; the new private
identity helper and Windows regression tests are the only code additions.

Full CLI/cache, final self-build/fixed-point, distribution, hosted CI, Linux,
release and remote checks are separate coordinator gates.

## Final source freeze

Receipt finalized after the coordinator independently observed the 227.778-second
PASS and all four generated products returning 42. No check was rerun for receipt
finalization, and this worker made no code edit after the successful replay.
HEAD remains `c5b6c9b430037de0b57cf1d0bc53a3fd520aa7f1` in this isolated checkout.

The following SHA-256 values fingerprint authored files at receipt finalization;
the successful temporary full import-closure workspace was already cleaned.
The last three are coordinator-owned inputs read only for this receipt.

| File relative to the isolated checkout | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/static_operation_identity.fy` | `9916cce77a478c95a705d57b254b7b136c3fe60ce256800f5ca7fe7a45ad1c39` |
| `examples/lexer_candidate/src/static_operation_group_identity.fy` | `40909928c5985170cb7eebbd6563d9c86327534acfd16afd809deaa87aad21aa` |
| `internal/cli/candidate_grouped_static_pipeline_test.go` | `c56b075e3bef32bbfe1bc8114269cb6e68521376f74e794a3df6e09d45d66d67` |
| `examples/lexer_candidate/src/static_operation_declarations.fy` | `89ff06cfcdc4d7c487c0e7e93f15843d341d1856679e010a707bba3d6f058442` |
| `examples/lexer_candidate/src/static_operation_groups.fy` | `ded814bfec20dc2a23e1c8406505013d285dabbca5fc2ad741871eb6fd4ddd6c` |
| `examples/lexer_candidate/src/generic_source_catalog.fy` | `5904a9ae330160cefa5de5d5b050b387e33e83c1625fd8171dc432d6173a885e` |

Final worker scope remains exactly the four files listed at the top. There are
no remaining worker blockers, live sessions or commits. ADR 0550 Result work is
not started and requires the coordinator's separate assignment and source/Go gate.
