package semantic

import "context"

// AnalyzeExperimentalStatePackages selects dynamic bytes/text storage codecs
// only for the explicit state-source compiler. Ordinary analysis retains the
// language-13 five-codec vocabulary. This grants no native or cache authority.
func AnalyzeExperimentalStatePackages(input PackageInput) (Result, error) {
	return AnalyzeExperimentalStatePackagesContext(context.Background(), input)
}

func AnalyzeExperimentalStatePackagesContext(ctx context.Context, input PackageInput) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	bindings, err := validatePackageInput(input)
	if err != nil {
		return Result{}, err
	}
	return analyzeContextWithOptions(ctx, input.Files, bindings, nil, nil, analysisOptions{dynamicStateCodecs: true})
}
