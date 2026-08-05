package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const testAttribute = "test"

type stableTestOptions struct {
	locked  bool
	native  bool
	release bool
}

type workspaceTest struct {
	display string
	entry   string
}

type workspaceTestOutcome struct {
	display      string
	trapCategory string
}

type testSourceDiagnostics struct {
	path        string
	contents    []byte
	diagnostics []diagnostic.Diagnostic
}

func runStableTest(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseStableTestOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
			return code
		}
		return packageFailure(stderr, err)
	}
	workspace, err = workspaceloader.WithRootTests(workspace)
	if err != nil {
		if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
			return code
		}
		return packageFailure(stderr, err)
	}
	tests, code := prepareWorkspaceTests(&workspace, stderr)
	if code != ExitSuccess {
		return code
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return packageFailure(stderr, err)
	}
	checked, pipelineErr := compiler.CheckPackages(input)
	if len(checked.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(checked.Semantic.Unsupported) != 0 {
		if err := writeStableUnsupportedSemantics(stderr, checked.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if pipelineErr != nil {
		var failure *compiler.Error
		if errors.As(pipelineErr, &failure) && failure.Stage == compiler.StageMIRLower {
			return workspaceSubsetCapabilityError(
				workspaceCompilePolicy{stableDiagnostics: true}, stderr, "semantic test", failure.Cause,
			)
		}
		return internalError(stderr)
	}
	if !checked.Verified {
		return internalError(stderr)
	}

	var outcomes []workspaceTestOutcome
	if options.native {
		profile := cdriver.LowProfile
		if options.release {
			profile = cdriver.OptimizedProfile
		}
		outcomes, code = runNativeWorkspaceTests(checked.MIR, tests, profile, stderr)
	} else {
		outcomes, code = runInterpreterWorkspaceTests(checked.MIR, tests, stderr)
	}
	if code != ExitSuccess {
		return code
	}
	return writeStableTestReport(outcomes, lockState, stdout, stderr)
}

func runInterpreterWorkspaceTests(program mir.Program, tests []workspaceTest, stderr io.Writer) ([]workspaceTestOutcome, int) {
	outcomes := make([]workspaceTestOutcome, 0, len(tests))
	for _, test := range tests {
		entry, found := programEntry(program, test.entry)
		if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeVoid {
			return nil, internalError(stderr)
		}
		outcome, err := interpreter.ExecuteVoid(program, test.entry)
		if err != nil {
			var capability interpreter.CapabilityError
			if errors.As(err, &capability) {
				return nil, workspaceSubsetCapabilityError(
					workspaceCompilePolicy{stableDiagnostics: true}, stderr, "semantic test interpreter",
					fmt.Errorf("%s: %w", test.display, err),
				)
			}
			return nil, internalError(stderr)
		}
		switch outcome.Kind {
		case interpreter.OutcomeReturn:
			outcomes = append(outcomes, workspaceTestOutcome{display: test.display})
		case interpreter.OutcomeTrap:
			outcomes = append(outcomes, workspaceTestOutcome{display: test.display, trapCategory: outcome.TrapCategory})
		default:
			return nil, internalError(stderr)
		}
	}
	return outcomes, ExitSuccess
}

func writeStableTestReport(outcomes []workspaceTestOutcome, lockState string, stdout, stderr io.Writer) int {
	var report strings.Builder
	passed := 0
	failed := 0
	for _, outcome := range outcomes {
		if outcome.trapCategory == "" {
			passed++
			fmt.Fprintf(&report, "test %s ... ok\n", outcome.display)
			continue
		}
		failed++
		fmt.Fprintf(&report, "test %s ... FAILED (%s)\n", outcome.display, outcome.trapCategory)
	}
	status := "ok"
	if failed != 0 {
		status = "FAILED"
	}
	fmt.Fprintf(&report, "test result: %s. %d passed; %d failed; lock %s\n", status, passed, failed, lockState)
	if _, err := io.WriteString(stdout, report.String()); err != nil {
		return outputError(stderr)
	}
	if failed != 0 {
		return ExitProgramTrap
	}
	return ExitSuccess
}

func parseStableTestOptions(arguments []string) (stableTestOptions, error) {
	options := stableTestOptions{}
	seen := make(map[string]bool, 3)
	for _, argument := range arguments {
		if argument != "--locked" && argument != "--native" && argument != "--release" {
			return stableTestOptions{}, fmt.Errorf(`command "test" accepts only optional --native, --release, and --locked`)
		}
		if seen[argument] {
			return stableTestOptions{}, fmt.Errorf(`command "test" received %s more than once`, argument)
		}
		seen[argument] = true
		switch argument {
		case "--locked":
			options.locked = true
		case "--native":
			options.native = true
		case "--release":
			options.release = true
		}
	}
	if options.release && !options.native {
		return stableTestOptions{}, fmt.Errorf(`command "test" requires --native when --release is supplied`)
	}
	return options, nil
}

func prepareWorkspaceTests(workspace *workspaceloader.Workspace, stderr io.Writer) ([]workspaceTest, int) {
	rootPackageID, err := workspace.RootPackageID()
	if err != nil {
		return nil, packageFailure(stderr, err)
	}
	rootPackage, found := workspace.Package(".")
	if !found {
		return nil, packageFailure(stderr, fmt.Errorf("workspace root package is missing"))
	}
	var tests []workspaceTest
	var sourceDiagnostics []testSourceDiagnostics
	for _, file := range rootPackage.Sources {
		if !file.Test {
			continue
		}
		var diagnostics []diagnostic.Diagnostic
		collectInvalidTestAttributes(file.Module.Structs, func(value ast.Struct) []ast.Attribute { return value.Attributes }, &diagnostics)
		collectInvalidTestAttributes(file.Module.Enums, func(value ast.Enum) []ast.Attribute { return value.Attributes }, &diagnostics)
		collectInvalidTestAttributes(file.Module.Resources, func(value ast.Resource) []ast.Attribute { return value.Attributes }, &diagnostics)
		collectInvalidTestAttributes(file.Module.ForeignFunctions, func(value ast.ForeignFunction) []ast.Attribute { return value.Attributes }, &diagnostics)
		for index := range file.Module.Resources {
			cleanup := file.Module.Resources[index].Cleanup
			if cleanup != nil {
				appendInvalidTestAttributes(cleanup.Attributes, &diagnostics)
			}
		}
		for index := range file.Module.Functions {
			function := &file.Module.Functions[index]
			attributes := function.Attributes[:0]
			var declarations []ast.Attribute
			for _, attribute := range function.Attributes {
				if attribute.Name == testAttribute {
					declarations = append(declarations, attribute)
					continue
				}
				attributes = append(attributes, attribute)
			}
			function.Attributes = attributes
			if len(declarations) == 0 {
				continue
			}
			valid := len(declarations) == 1 && len(declarations[0].Arguments) == 0 &&
				!function.Public && !function.Unsafe && function.Owner == "" &&
				len(function.Parameters) == 0 && function.ResultType.Kind == ast.TypeKindNamed &&
				function.ResultType.Path == string(hir.TypeVoid)
			if !valid {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{
					Code: "E9001", Span: declarations[0].Span, Repair: diagnostic.RepairNone,
				})
				continue
			}
			display := file.Module.Path + "::" + function.Name
			tests = append(tests, workspaceTest{
				display: display,
				entry:   hir.QualifiedModulePath(rootPackageID, file.Module.Path) + "::" + function.Name,
			})
		}
		if len(diagnostics) != 0 {
			slices.SortFunc(diagnostics, diagnostic.Compare)
			sourceDiagnostics = append(sourceDiagnostics, testSourceDiagnostics{
				path: file.Path, contents: slices.Clone(file.Bytes), diagnostics: diagnostics,
			})
		}
	}
	if len(sourceDiagnostics) != 0 {
		for _, source := range sourceDiagnostics {
			if err := writeDiagnostics(stderr, source.path, source.contents, source.diagnostics); err != nil {
				return nil, ExitInternalFailure
			}
		}
		return nil, ExitSourceDiagnostics
	}
	slices.SortFunc(tests, func(left, right workspaceTest) int { return strings.Compare(left.display, right.display) })
	return tests, ExitSuccess
}

func collectInvalidTestAttributes[T any](values []T, attributes func(T) []ast.Attribute, diagnostics *[]diagnostic.Diagnostic) {
	for _, value := range values {
		appendInvalidTestAttributes(attributes(value), diagnostics)
	}
}

func appendInvalidTestAttributes(attributes []ast.Attribute, diagnostics *[]diagnostic.Diagnostic) {
	for _, attribute := range attributes {
		if attribute.Name == testAttribute {
			*diagnostics = append(*diagnostics, diagnostic.Diagnostic{
				Code: "E9001", Span: attribute.Span, Repair: diagnostic.RepairNone,
			})
		}
	}
}
