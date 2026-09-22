package nominalhir

import "github.com/0disoft/fairyc/internal/compiler"

// LowerAnalyzedStoragePatch checks a build-local nominal overlay before the
// existing storage-aware lowering and verification boundary runs exactly once.
// The analysis and patch must originate from the same in-process request.
// Exact erasure leaves the original scalar HIR unchanged; no baseline MIR is
// required or cloned. This experiment does not admit nominal source syntax,
// transport nominal identities, or change stable workspace compilation.
func LowerAnalyzedStoragePatch(analysis compiler.Analysis, patch BoundaryPatch) (compiler.Result, error) {
	if err := validateAnalysisPatch(analysis.Semantic, patch, nil, nil); err != nil {
		return compiler.Result{}, err
	}
	return compiler.LowerAnalyzedStorageTransfers(analysis)
}
