package cli

import (
	"encoding/json"
	"io"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type testPlanReport struct {
	Schema    string                  `json:"schema"`
	Package   string                  `json:"package_uid"`
	Symbol    string                  `json:"symbol"`
	Recommend string                  `json:"recommendation"`
	Tests     []atlasmap.TestPlanPath `json:"tests"`
	Reasons   []string                `json:"reasons"`
}

func runTestPlan(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	var symbol, uid string
	_, err := parseCommandFlags(args, flagSchema{
		Command: "test-plan", Experimental: true, Required: []string{"--symbol"},
		Flags: []flagDefinition{{Name: "--symbol", Kind: flagScalar}, {Name: "--package", Kind: flagScalar}},
	}, func(flag, value string) error {
		if flag == "--package" {
			if err := validateAtlasPackageUID("test-plan", value); err != nil {
				return err
			}
			uid = value
		} else {
			symbol = value
		}
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, err := workspaceloader.Load(sourceRoot)
	if err != nil {
		return packageFailure(stderr, err)
	}
	selected, err := selectWorkspaceTestPackage(workspace, uid)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, err = workspaceloader.WithPackageTests(workspace, selected.Path)
	if err != nil {
		return packageFailure(stderr, err)
	}
	tests, _, code := prepareWorkspaceTests(workspace, selected.Path, "", false, stderr)
	if code != ExitSuccess {
		return code
	}
	report := testPlanReport{
		Schema: "fairyc.test-plan/v1", Package: selected.Manifest.Package.UID, Symbol: symbol,
		Tests: []atlasmap.TestPlanPath{}, Reasons: []string{},
	}
	index, code := buildTestPlanIndex(workspace, selected.Path, stderr)
	if code != ExitSuccess {
		return code
	}
	if index == nil {
		report.Reasons = append(report.Reasons, "atlas-unavailable")
	} else {
		identities := make([]string, len(tests))
		for i, test := range tests {
			identities[i] = test.display
		}
		report.Tests, report.Reasons = index.TestPlan(symbol, identities)
	}
	return writeTestPlan(report, stdout, stderr)
}

// Compile the test-augmented snapshot once. Atlas identities are independently
// checked against that snapshot, with no persisted maps or lock updates.
func buildTestPlanIndex(workspace workspaceloader.Workspace, packagePath string, stderr io.Writer) (*atlasmap.QueryIndex, int) {
	result, err := workspacecompiler.Check(workspace)
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return nil, ExitInternalFailure
		}
		return nil, ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeStableUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return nil, ExitInternalFailure
		}
		return nil, ExitSourceDiagnostics
	}
	if err != nil || !result.Verified {
		return nil, ExitSuccess
	}
	packageID, err := workspace.PackageID(packagePath)
	if err != nil {
		return nil, ExitSuccess
	}
	instances, err := workspace.PackageIdentities()
	if err != nil {
		return nil, ExitSuccess
	}
	identity, err := buildidentity.Current()
	if err != nil {
		return nil, ExitSuccess
	}
	_, partitions, err := atlasmap.PrepareWorkspaceV3(workspace, result, identity)
	if err != nil {
		return nil, ExitSuccess
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectationsV3(workspace, result, identity)
	if err != nil {
		return nil, ExitSuccess
	}
	instance := instances[packageID].String()
	for _, partition := range partitions {
		if partition.Header().PackageInstance == instance {
			if index, err := atlasmap.NewQueryIndexV3(partition, expectations[instance]); err == nil {
				return index, ExitSuccess
			}
			break
		}
	}
	return nil, ExitSuccess
}

func writeTestPlan(report testPlanReport, stdout, stderr io.Writer) int {
	report.Recommend = "selected-tests"
	if len(report.Reasons) != 0 {
		report.Recommend = "all-tests"
	}
	data, err := json.Marshal(report)
	if err != nil {
		return internalError(stderr)
	}
	if _, err := stdout.Write(append(data, '\n')); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
