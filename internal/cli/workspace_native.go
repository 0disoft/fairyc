package cli

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
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

type stableBuildOptions struct {
	locked  bool
	release bool
}

type compiledWorkspace struct {
	artifact        cdriver.WindowsHostArtifact
	lockState       string
	rootPackageName string
}

func runStableBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseStableBuildOptions(arguments)
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
	}, true, stderr)
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

func parseStableBuildOptions(arguments []string) (stableBuildOptions, error) {
	options := stableBuildOptions{}
	seen := make(map[string]bool, 2)
	for _, argument := range arguments {
		if argument != "--locked" && argument != "--release" {
			return stableBuildOptions{}, fmt.Errorf(`command "build" accepts only optional --release and --locked`)
		}
		if seen[argument] {
			return stableBuildOptions{}, fmt.Errorf(`command "build" received %s more than once`, argument)
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
	compiled, code := compileWindowsWorkspace(sourceRoot, options, false, stderr)
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
	compiled, code := compileWindowsWorkspace(sourceRoot, options, false, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if outcome.Stdout != "" || outcome.Stderr != "" {
		return buildFailure(stderr, fmt.Errorf("Windows workspace executable produced unexpected output"))
	}
	if category, trapped := windowsHostTrapCategory(outcome.ExitCode); trapped {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(
		stdout,
		"native workspace result: %d (%s, msvc %s, %s, lock %s)\n",
		outcome.ExitCode,
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

func compileWindowsWorkspace(sourceRoot string, options workspaceTargetOptions, stableDiagnostics bool, stderr io.Writer) (compiledWorkspace, int) {
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		if stableDiagnostics {
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
	artifact, code := compileWindowsProgram(checked.MIR, entryName, options.targetID, options.profile, stderr)
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

func workspaceEntryName(workspace workspaceloader.Workspace) (string, error) {
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return "", err
	}
	return hir.QualifiedModulePath(rootPackage, "main") + "::main", nil
}
