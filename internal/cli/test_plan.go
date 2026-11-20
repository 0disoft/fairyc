package cli

import (
	"encoding/json"
	"io"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type testPlanReport struct {
	Schema        string                  `json:"schema"`
	Package       string                  `json:"package_uid"`
	Symbol        string                  `json:"symbol"`
	SymbolPackage string                  `json:"symbol_package_uid,omitempty"`
	Recommend     string                  `json:"recommendation"`
	Tests         []atlasmap.TestPlanPath `json:"tests"`
	Reasons       []string                `json:"reasons"`
}

func runTestPlan(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	var symbol, uid, subjectUID string
	_, err := parseCommandFlags(args, flagSchema{
		Command: "test-plan", Experimental: true, Required: []string{"--symbol"},
		Flags: []flagDefinition{{Name: "--symbol", Kind: flagScalar}, {Name: "--package", Kind: flagScalar}, {Name: "--symbol-package", Kind: flagScalar}},
	}, func(flag, value string) error {
		if flag == "--symbol" {
			symbol = value
			return nil
		}
		if err := validateAtlasPackageUID("test-plan", value); err != nil {
			return err
		}
		if flag == "--package" {
			uid = value
		} else {
			subjectUID = value
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
	var subjectPath string
	if subjectUID != "" {
		subject, err := selectWorkspaceTestPackage(workspace, subjectUID)
		if err != nil {
			return invocationError(stderr, err.Error())
		}
		subjectPath = subject.Path
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
		Schema: "fairyc.test-plan/v1", Package: selected.Manifest.Package.UID, Symbol: symbol, SymbolPackage: subjectUID,
		Tests: []atlasmap.TestPlanPath{}, Reasons: []string{},
	}
	input, code := buildTestPlanIndex(workspace, selected.Path, subjectPath, stderr)
	if code != ExitSuccess {
		return code
	}
	identities := make([]string, len(tests))
	for i, test := range tests {
		identities[i] = test.display
	}
	switch {
	case input.subjectInstance != "":
		report.Tests, report.Reasons = atlasmap.WorkspaceTestPlan(input.partitions, input.expectations, input.testInstance, input.subjectInstance, symbol, identities, input.result, input.packageIDs)
	case input.index != nil:
		report.Tests, report.Reasons = input.index.TestPlan(symbol, identities, input.result, input.packageID)
	default:
		report.Reasons = append(report.Reasons, "atlas-unavailable")
	}
	return writeTestPlan(report, stdout, stderr)
}

// Compile the test-augmented snapshot once. Atlas identities are independently
// checked against that snapshot, with no persisted maps or lock updates.
type testPlanInput struct {
	index                         *atlasmap.QueryIndex
	result                        compiler.Result
	packageID                     hir.PackageID
	partitions                    []atlasmap.Partition
	expectations                  map[string]atlasmap.LoadExpectation
	packageIDs                    map[string]hir.PackageID
	testInstance, subjectInstance string
}

func buildTestPlanIndex(workspace workspaceloader.Workspace, packagePath, subjectPath string, stderr io.Writer) (testPlanInput, int) {
	result, err := workspacecompiler.CheckGeneric(workspace)
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return testPlanInput{}, ExitInternalFailure
		}
		return testPlanInput{}, ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeStableUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return testPlanInput{}, ExitInternalFailure
		}
		return testPlanInput{}, ExitSourceDiagnostics
	}
	if err != nil || !result.Verified {
		return testPlanInput{}, ExitSuccess
	}
	packageID, err := workspace.PackageID(packagePath)
	if err != nil {
		return testPlanInput{}, ExitSuccess
	}
	instances, err := workspace.PackageIdentities()
	if err != nil {
		return testPlanInput{}, ExitSuccess
	}
	identity, err := buildidentity.Current()
	if err != nil {
		return testPlanInput{}, ExitSuccess
	}
	_, partitions, err := atlasmap.PrepareWorkspaceV3(workspace, result, identity)
	if err != nil {
		return testPlanInput{}, ExitSuccess
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectationsV3(workspace, result, identity)
	if err != nil {
		return testPlanInput{}, ExitSuccess
	}
	instance := instances[packageID].String()
	if subjectPath != "" {
		subjectID, err := workspace.PackageID(subjectPath)
		if err != nil {
			return testPlanInput{}, ExitSuccess
		}
		packageIDs := make(map[string]hir.PackageID, len(instances))
		for id, instance := range instances {
			packageIDs[instance.String()] = id
		}
		return testPlanInput{
			partitions: partitions, expectations: expectations, result: result, packageIDs: packageIDs,
			testInstance: instance, subjectInstance: instances[subjectID].String(),
		}, ExitSuccess
	}
	for _, partition := range partitions {
		if partition.Header().PackageInstance == instance {
			if index, err := atlasmap.NewQueryIndexV3(partition, expectations[instance]); err == nil {
				return testPlanInput{index: index, result: result, packageID: packageID}, ExitSuccess
			}
			break
		}
	}
	return testPlanInput{}, ExitSuccess
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
