# Go Atlas edit overlay SDK authority, 2026-11-05

## Reproduction and correction

Actual Vec body-edit acceptance exposed a Go reference defect. Original Atlas
preparation used the workspace-owned authenticated SDK policy, but
`sourceedit.validateOverlay` called the policy-free generic compiler directly.
A valid replacement using `Vec<bool>` therefore failed semantic/MIR validation
even though the same complete edited workspace passed fresh Atlas preparation.

The overlay now uses `workspacecompiler.CheckGeneric`, the existing current-source
owner. It authenticates the exact selected SDK bytes, AST and identity before
deriving the SDK package's private storage admission. It preserves generic
aggregate support, full semantic/Flow/MIR verification and the existing public
error classification. No blanket client-package intrinsic permission is added.
Both preview and apply's fresh revalidation use this correction.

## Windows verification

`TestAtlasEditSDKVectorOverlay` reproduces preview failure before the correction.
Afterward it passes preview and exact-byte atomic apply of a newly requested
`Vec<bool>` body. Wrong bool arguments and an attempted client `storage_size`
intrinsic reject without writes. Preview preserves every workspace byte; apply
preserves the graph lock, unrelated source and body file, and leaves no staging
or write-lock file.

The SDK overlay, ordinary preview/apply and dependency-drift/link regressions
pass on Windows. The actual combined command also includes the ongoing native
fixture's independent Go specialization oracle:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestAtlas(EditSDKVectorOverlay|EditPreviewApplyAndRejectEscapes|EditPlanRejectsDependencyDriftAndLinks|SDKVectorEditOracle)$'
go test -mod=readonly -count=1 -timeout 5m ./internal/sourceedit
go vet -mod=readonly ./internal/sourceedit ./internal/cli
git diff --check
```

The combined CLI command passes in 2.092s; the sourceedit package passes in
1.341s. Vet, formatting and diff checks pass. The Go specialization oracle is
part of the following Vec command fixture; native Fairy body-edit acceptance is
not inferred from these Go results. The original failed preview is retained in
`.cache/atlas-sdk-vector-go-edit-before-20261105.log`, and the passing command is
in `.cache/atlas-sdk-vector-go-edit-after-20261105.log`.

The first test draft accidentally selected a legacy non-Vec symbol-catalog
route and failed before exercising editing. Its corrected fixture selects a
current v3 symbol; that version reproduces the actual overlay defect described
above. Earlier native harness attempts also used inadmissible i16 storage;
those are fixture failures, not evidence of a new Vec source admission.

Native Linux stays batched. Complete authored CLI, SDK/user-aggregate combinations,
self-build, distribution and hosted CI remain separate gates. No public API,
SDK/runtime ABI, map schema, DB, dependency, product timeout, target, repository
hygiene or CI runner changes. No push, dispatch or publication.
