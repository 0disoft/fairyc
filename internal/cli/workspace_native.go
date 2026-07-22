package cli

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type workspaceTargetOptions struct {
	targetID target.ID
	profile  cdriver.Profile
	output   string
	locked   bool
}

type stableNativeOptions struct {
	locked  bool
	release bool
}

type compiledWorkspace struct {
	artifact        cdriver.WindowsHostArtifact
	lockState       string
	rootPackageName string
}

type workspaceCompilePolicy struct {
	stableDiagnostics bool
	runTrapReceipt    bool
}

func runStableBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseStableNativeOptions("build", arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	profile := cdriver.LowProfile
	profileName := "development"
	if options.release {
		profile = cdriver.OptimizedProfile
		profileName = "release"
	}
	compiled, code := compileWindowsWorkspace(sourceRoot, workspaceTargetOptions{
		targetID: target.X8664WindowsMSVC,
		profile:  profile,
		locked:   options.locked,
	}, workspaceCompilePolicy{stableDiagnostics: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	output := path.Join("build", compiled.rootPackageName+".exe")
	if err := replaceBuildArtifact(sourceRoot, output, compiled.artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built: %s (%s, %s, lock %s)\n",
		output,
		target.X8664WindowsMSVC,
		profileName,
		compiled.lockState,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func runStableRun(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseStableNativeOptions("run", arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	profile := cdriver.LowProfile
	profileName := "development"
	if options.release {
		profile = cdriver.OptimizedProfile
		profileName = "release"
	}
	compiled, code := compileWindowsWorkspace(sourceRoot, workspaceTargetOptions{
		targetID: target.X8664WindowsMSVC,
		profile:  profile,
		locked:   options.locked,
	}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	result, category, err := classifyWorkspaceRunOutcome(outcome)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if category != "" {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(
		stdout,
		"result: %d (%s, %s, lock %s)\n",
		result,
		target.X8664WindowsMSVC,
		profileName,
		compiled.lockState,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseStableNativeOptions(command string, arguments []string) (stableNativeOptions, error) {
	options := stableNativeOptions{}
	seen := make(map[string]bool, 2)
	for _, argument := range arguments {
		if argument != "--locked" && argument != "--release" {
			return stableNativeOptions{}, fmt.Errorf(`command %q accepts only optional --release and --locked`, command)
		}
		if seen[argument] {
			return stableNativeOptions{}, fmt.Errorf(`command %q received %s more than once`, command, argument)
		}
		seen[argument] = true
		switch argument {
		case "--locked":
			options.locked = true
		case "--release":
			options.release = true
		}
	}
	return options, nil
}

func runWorkspaceBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseWorkspaceTargetOptions("build-workspace", arguments, true)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	compiled, code := compileWindowsWorkspace(sourceRoot, options, workspaceCompilePolicy{}, stderr)
	if code != ExitSuccess {
		return code
	}
	if err := writeBuildArtifact(sourceRoot, options.output, compiled.artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built workspace: %s (%s, msvc %s, %s, lock %s)\n",
		options.output,
		options.targetID,
		compiled.artifact.Toolchain.Version,
		compiled.artifact.Profile.Name,
		compiled.lockState,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func runWorkspaceTarget(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseWorkspaceTargetOptions("run-workspace", arguments, false)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	compiled, code := compileWindowsWorkspace(sourceRoot, options, workspaceCompilePolicy{runTrapReceipt: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	result, category, err := classifyWorkspaceRunOutcome(outcome)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if category != "" {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(
		stdout,
		"native workspace result: %d (%s, msvc %s, %s, lock %s)\n",
		result,
		options.targetID,
		compiled.artifact.Toolchain.Version,
		compiled.artifact.Profile.Name,
		compiled.lockState,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseWorkspaceTargetOptions(command string, arguments []string, requireOutput bool) (workspaceTargetOptions, error) {
	options := workspaceTargetOptions{}
	seen := make(map[string]bool, 4)
	for index := 0; index < len(arguments); {
		flag := arguments[index]
		if flag == "--locked" {
			if seen[flag] {
				return workspaceTargetOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
			}
			seen[flag] = true
			options.locked = true
			index++
			continue
		}
		if flag != "--target" && flag != "--profile" && (!requireOutput || flag != "--output") {
			return workspaceTargetOptions{}, fmt.Errorf("experimental command %q does not accept flag %q", command, flag)
		}
		if seen[flag] {
			return workspaceTargetOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return workspaceTargetOptions{}, fmt.Errorf("experimental command %q requires a value for %s", command, flag)
		}
		seen[flag] = true
		value := arguments[index+1]
		switch flag {
		case "--target":
			if target.ID(value) != target.X8664WindowsMSVC {
				return workspaceTargetOptions{}, fmt.Errorf("experimental %s currently requires --target %s", command, target.X8664WindowsMSVC)
			}
			options.targetID = target.X8664WindowsMSVC
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return workspaceTargetOptions{}, fmt.Errorf("experimental command %q requires --profile low or --profile optimized", command)
			}
		case "--output":
			normalized, err := validateBuildOutput(value)
			if err != nil {
				return workspaceTargetOptions{}, err
			}
			if !strings.HasSuffix(strings.ToLower(normalized), ".exe") {
				return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace Windows output must end with .exe")
			}
			options.output = normalized
		}
		index += 2
	}
	for _, flag := range []string{"--target", "--profile"} {
		if !seen[flag] {
			return workspaceTargetOptions{}, fmt.Errorf("experimental command %q requires %s", command, flag)
		}
	}
	if requireOutput && !seen["--output"] {
		return workspaceTargetOptions{}, fmt.Errorf("experimental command %q requires --output", command)
	}
	return options, nil
}

func compileWindowsWorkspace(sourceRoot string, options workspaceTargetOptions, policy workspaceCompilePolicy, stderr io.Writer) (compiledWorkspace, int) {
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		if policy.stableDiagnostics {
			if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
				return compiledWorkspace{}, code
			}
		}
		return compiledWorkspace{}, packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return compiledWorkspace{}, packageFailure(stderr, err)
	}
	checked, pipelineErr := compiler.CheckPackages(input)
	if len(checked.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Semantic.Diagnostics); err != nil {
			return compiledWorkspace{}, ExitInternalFailure
		}
		return compiledWorkspace{}, ExitSourceDiagnostics
	}
	if len(checked.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, checked.Semantic.Unsupported); err != nil {
			return compiledWorkspace{}, ExitInternalFailure
		}
		return compiledWorkspace{}, ExitSourceDiagnostics
	}
	if pipelineErr != nil {
		var failure *compiler.Error
		if errors.As(pipelineErr, &failure) && failure.Stage == compiler.StageMIRLower {
			return compiledWorkspace{}, subsetCapabilityError(stderr, "Windows workspace C-emission", failure.Cause)
		}
		return compiledWorkspace{}, internalError(stderr)
	}
	if !checked.Verified {
		return compiledWorkspace{}, internalError(stderr)
	}
	entryName, err := workspaceEntryName(workspace)
	if err != nil {
		return compiledWorkspace{}, packageFailure(stderr, err)
	}
	entry, found := programEntry(checked.MIR, entryName)
	if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeI32 {
		return compiledWorkspace{}, subsetCapabilityError(stderr, "Windows workspace C-emission", fmt.Errorf("root entry %s must have type fn() -> i32", entryName))
	}
	artifact, code := compileWindowsProgramWithRunReceipt(checked.MIR, entryName, options.targetID, options.profile, policy.runTrapReceipt, stderr)
	if code != ExitSuccess {
		return compiledWorkspace{}, code
	}
	rootPackage, found := workspace.Package(".")
	if !found {
		return compiledWorkspace{}, packageFailure(stderr, fmt.Errorf("workspace root package is missing"))
	}
	return compiledWorkspace{
		artifact: artifact, lockState: lockState, rootPackageName: rootPackage.Manifest.Package.Name,
	}, ExitSuccess
}

func classifyWorkspaceRunOutcome(outcome cdriver.WindowsHostOutcome) (int32, string, error) {
	if outcome.Stdout != "" {
		return 0, "", fmt.Errorf("Windows workspace executable produced unexpected output")
	}
	if outcome.Stderr == "" {
		return outcome.ExitCode, "", nil
	}
	category, trapped := windowsHostTrapCategory(outcome.ExitCode)
	if !trapped {
		return 0, "", fmt.Errorf("Windows workspace executable produced unexpected output")
	}
	trapCode := int(outcome.ExitCode) - productTrapExitBase
	expected := fmt.Sprintf("%s%d", cruntime.RunTrapReceiptPrefix, trapCode)
	if outcome.Stderr != expected {
		return 0, "", fmt.Errorf("Windows workspace executable produced an invalid trap receipt")
	}
	return 0, category, nil
}

func workspaceEntryName(workspace workspaceloader.Workspace) (string, error) {
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return "", err
	}
	return hir.QualifiedModulePath(rootPackage, "main") + "::main", nil
}
