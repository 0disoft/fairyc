package cli

import (
	"errors"
	"fmt"
	"io"
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

func runWorkspaceBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseWorkspaceTargetOptions("build-workspace", arguments, true)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, lockState, code := compileWindowsWorkspace(sourceRoot, options, stderr)
	if code != ExitSuccess {
		return code
	}
	if err := writeBuildArtifact(sourceRoot, options.output, artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built workspace: %s (%s, msvc %s, %s, lock %s)\n",
		options.output,
		options.targetID,
		artifact.Toolchain.Version,
		artifact.Profile.Name,
		lockState,
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
	artifact, lockState, code := compileWindowsWorkspace(sourceRoot, options, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
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
		artifact.Toolchain.Version,
		artifact.Profile.Name,
		lockState,
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

func compileWindowsWorkspace(sourceRoot string, options workspaceTargetOptions, stderr io.Writer) (cdriver.WindowsHostArtifact, string, int) {
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		return cdriver.WindowsHostArtifact{}, "", packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return cdriver.WindowsHostArtifact{}, "", packageFailure(stderr, err)
	}
	checked, pipelineErr := compiler.CheckPackages(input)
	if len(checked.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Semantic.Diagnostics); err != nil {
			return cdriver.WindowsHostArtifact{}, "", ExitInternalFailure
		}
		return cdriver.WindowsHostArtifact{}, "", ExitSourceDiagnostics
	}
	if len(checked.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, checked.Semantic.Unsupported); err != nil {
			return cdriver.WindowsHostArtifact{}, "", ExitInternalFailure
		}
		return cdriver.WindowsHostArtifact{}, "", ExitSourceDiagnostics
	}
	if pipelineErr != nil {
		var failure *compiler.Error
		if errors.As(pipelineErr, &failure) && failure.Stage == compiler.StageMIRLower {
			return cdriver.WindowsHostArtifact{}, "", subsetCapabilityError(stderr, "Windows workspace C-emission", failure.Cause)
		}
		return cdriver.WindowsHostArtifact{}, "", internalError(stderr)
	}
	if !checked.Verified {
		return cdriver.WindowsHostArtifact{}, "", internalError(stderr)
	}
	entryName, err := workspaceEntryName(workspace)
	if err != nil {
		return cdriver.WindowsHostArtifact{}, "", packageFailure(stderr, err)
	}
	entry, found := programEntry(checked.MIR, entryName)
	if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeI32 {
		return cdriver.WindowsHostArtifact{}, "", subsetCapabilityError(stderr, "Windows workspace C-emission", fmt.Errorf("root entry %s must have type fn() -> i32", entryName))
	}
	artifact, code := compileWindowsProgram(checked.MIR, entryName, options.targetID, options.profile, stderr)
	if code != ExitSuccess {
		return cdriver.WindowsHostArtifact{}, "", code
	}
	return artifact, lockState, ExitSuccess
}

func workspaceEntryName(workspace workspaceloader.Workspace) (string, error) {
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return "", err
	}
	return hir.QualifiedModulePath(rootPackage, "main") + "::main", nil
}
