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
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type workspaceTargetOptions struct {
	targetID     target.ID
	profile      cdriver.Profile
	output       string
	locked       bool
	compilerName string
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

type loweredWorkspace struct {
	program         mir.Program
	entryName       string
	lockState       string
	rootPackageName string
	sdkStorage      bool
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

func runStableRun(arguments []string, sourceRoot string, stdin io.Reader, stdout, stderr io.Writer) int {
	compilerArguments, programArguments, err := splitStableRunArguments(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	options, err := parseStableNativeOptions("run", compilerArguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	profile := cdriver.LowProfile
	if options.release {
		profile = cdriver.OptimizedProfile
	}
	compiled, code := compileWindowsWorkspace(sourceRoot, workspaceTargetOptions{
		targetID: target.X8664WindowsMSVC,
		profile:  profile,
		locked:   options.locked,
	}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(compiled.artifact, stdin, programArguments)
	if err != nil {
		return buildFailure(stderr, err)
	}
	programStderr, category, err := classifyStableRunOutcome(outcome)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if category != "" {
		programStderr += fmt.Sprintf("trap: %s\n", category)
	}
	if code := forwardProgramOutput(outcome.Stdout, programStderr, stdout, stderr); code != ExitSuccess {
		return code
	}
	if category != "" {
		return ExitProgramTrap
	}
	return int(uint32(outcome.ExitCode))
}

// splitStableRunArguments separates the first "--" and everything after it as program arguments.
// The delimiter itself is not passed to the program. Program arguments are part of the stable run
// command line, so an embedded NUL byte is rejected before compilation.
func splitStableRunArguments(arguments []string) (compilerArguments, programArguments []string, err error) {
	compilerArguments, programArguments, _, err = splitStableProgramArguments("run", arguments)
	return compilerArguments, programArguments, err
}

func splitStableProgramArguments(command string, arguments []string) (compilerArguments, programArguments []string, found bool, err error) {
	for index, argument := range arguments {
		if argument != "--" {
			continue
		}
		programArguments = arguments[index+1:]
		for _, programArgument := range programArguments {
			if strings.ContainsRune(programArgument, '\x00') {
				return nil, nil, false, fmt.Errorf(`command %q rejects program arguments containing a NUL byte`, command)
			}
		}
		return arguments[:index], programArguments, true, nil
	}
	return arguments, nil, false, nil
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
	if options.targetID == target.X8664LinuxGNU {
		artifact, lockState, code := compileLinuxWorkspace(sourceRoot, options, stderr)
		if code != ExitSuccess {
			return code
		}
		if err := writeBuildArtifact(sourceRoot, options.output, artifact.Bytes); err != nil {
			return buildFailure(stderr, err)
		}
		if _, err := fmt.Fprintf(stdout, "built workspace: %s (%s, %s %s, %s, lock %s)\n",
			options.output, options.targetID, artifact.Toolchain.Name, artifact.Toolchain.Version,
			artifact.Profile.Name, lockState); err != nil {
			return outputError(stderr)
		}
		return ExitSuccess
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
		if flag != "--target" && flag != "--profile" && !(requireOutput && (flag == "--output" || flag == "--cc")) {
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
			if target.ID(value) != target.X8664WindowsMSVC && !(command == "build-workspace" && target.ID(value) == target.X8664LinuxGNU) {
				return workspaceTargetOptions{}, fmt.Errorf("experimental %s currently requires --target %s", command, target.X8664WindowsMSVC)
			}
			options.targetID = target.ID(value)
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
			options.output = normalized
		case "--cc":
			if value != "gcc" && value != "clang" {
				return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace requires --cc gcc or --cc clang")
			}
			options.compilerName = value
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
	if options.targetID == target.X8664LinuxGNU {
		if !seen["--cc"] {
			return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace with Linux target requires --cc gcc or --cc clang")
		}
		if strings.HasSuffix(strings.ToLower(options.output), ".exe") {
			return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace Linux output must not end with .exe")
		}
	} else if requireOutput {
		if seen["--cc"] {
			return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace Windows target does not accept --cc")
		}
		if !strings.HasSuffix(strings.ToLower(options.output), ".exe") {
			return workspaceTargetOptions{}, fmt.Errorf("experimental build-workspace Windows output must end with .exe")
		}
	}
	return options, nil
}

func compileWindowsWorkspace(sourceRoot string, options workspaceTargetOptions, policy workspaceCompilePolicy, stderr io.Writer) (compiledWorkspace, int) {
	lowered, code := lowerWorkspaceExecutable(sourceRoot, options.locked, policy, "Windows workspace C-emission", stderr)
	if code != ExitSuccess {
		return compiledWorkspace{}, code
	}
	artifact, code := compileWindowsProgramPolicy(lowered.program, lowered.entryName, options.targetID, options.profile, policy.runTrapReceipt, lowered.sdkStorage, stderr)
	if code != ExitSuccess {
		return compiledWorkspace{}, code
	}
	return compiledWorkspace{
		artifact: artifact, lockState: lowered.lockState, rootPackageName: lowered.rootPackageName,
	}, ExitSuccess
}

func lowerWorkspaceExecutable(sourceRoot string, locked bool, policy workspaceCompilePolicy, subset string, stderr io.Writer) (loweredWorkspace, int) {
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, locked)
	if err != nil {
		if policy.stableDiagnostics {
			if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
				return loweredWorkspace{}, code
			}
		}
		return loweredWorkspace{}, packageFailure(stderr, err)
	}
	return lowerLoadedWorkspaceExecutable(workspace, lockState, policy, subset, stderr)
}

func lowerLoadedWorkspaceExecutable(workspace workspaceloader.Workspace, lockState string, policy workspaceCompilePolicy, subset string, stderr io.Writer) (loweredWorkspace, int) {
	checked, pipelineErr := workspacecompiler.Check(workspace)
	if len(checked.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, checked.Semantic.Diagnostics); err != nil {
			return loweredWorkspace{}, ExitInternalFailure
		}
		return loweredWorkspace{}, ExitSourceDiagnostics
	}
	if len(checked.Semantic.Unsupported) != 0 {
		writeUnsupported := writeUnsupportedSemantics
		if policy.stableDiagnostics {
			writeUnsupported = writeStableUnsupportedSemantics
		}
		if err := writeUnsupported(stderr, checked.Semantic.Unsupported); err != nil {
			return loweredWorkspace{}, ExitInternalFailure
		}
		return loweredWorkspace{}, ExitSourceDiagnostics
	}
	if pipelineErr != nil {
		var failure *compiler.Error
		if errors.As(pipelineErr, &failure) && failure.Stage == compiler.StageMIRLower {
			return loweredWorkspace{}, workspaceSubsetCapabilityError(policy, stderr, subset, failure.Cause)
		}
		return loweredWorkspace{}, internalError(stderr)
	}
	if !checked.Verified {
		return loweredWorkspace{}, internalError(stderr)
	}
	sdkStorage, err := workspaceSDKStorage(workspace)
	if err != nil {
		return loweredWorkspace{}, workspaceSubsetCapabilityError(policy, stderr, subset, err)
	}
	entryName, err := workspaceEntryName(workspace)
	if err != nil {
		return loweredWorkspace{}, packageFailure(stderr, err)
	}
	entry, found := programEntry(checked.MIR, entryName)
	if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeI32 {
		return loweredWorkspace{}, workspaceSubsetCapabilityError(policy, stderr, subset, fmt.Errorf("root entry %s must have type fn() -> i32", entryName))
	}
	rootPackage, found := workspace.Package(".")
	if !found {
		return loweredWorkspace{}, packageFailure(stderr, fmt.Errorf("workspace root package is missing"))
	}
	return loweredWorkspace{
		program: checked.MIR, entryName: entryName, lockState: lockState,
		rootPackageName: rootPackage.Manifest.Package.Name,
		sdkStorage:      sdkStorage,
	}, ExitSuccess
}

func workspaceSubsetCapabilityError(policy workspaceCompilePolicy, stderr io.Writer, subset string, err error) int {
	if !policy.stableDiagnostics {
		return subsetCapabilityError(stderr, subset, err)
	}
	message := fmt.Sprintf("requested workspace is outside the %s subset", subset)
	if err != nil {
		message += ": " + strings.TrimSpace(err.Error())
	}
	if _, writeErr := fmt.Fprintf(stderr, "fairyc: %s\n", message); writeErr != nil {
		return ExitInternalFailure
	}
	return ExitInvalidInvocation
}

func classifyStableRunOutcome(outcome cdriver.WindowsHostOutcome) (string, string, error) {
	category, trapped := windowsHostTrapCategory(outcome.ExitCode)
	if !trapped {
		if outcome.TrapControl != "" {
			return "", "", fmt.Errorf("Windows workspace executable produced unexpected trap control data")
		}
		return outcome.Stderr, "", nil
	}
	trapCode := int(outcome.ExitCode) - productTrapExitBase
	receipt := string([]byte{byte(trapCode)})
	if outcome.TrapControl == "" {
		return outcome.Stderr, "", nil
	}
	if outcome.TrapControl != receipt {
		return "", "", fmt.Errorf("Windows workspace executable produced invalid trap control data")
	}
	return outcome.Stderr, category, nil
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
	expected := string([]byte{byte(trapCode)})
	if outcome.Stderr != "" || outcome.TrapControl != expected {
		return 0, "", fmt.Errorf("Windows workspace executable produced invalid trap control data")
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
