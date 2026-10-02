package cli

import (
	"fmt"
	"io"
	"path"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func stableLinuxOptions(options stableNativeOptions, profile cdriver.Profile) (workspaceTargetOptions, error) {
	if runtime.GOARCH != "amd64" {
		return workspaceTargetOptions{}, fmt.Errorf("Linux native commands require amd64, got %s", runtime.GOARCH)
	}
	return workspaceTargetOptions{targetID: target.X8664LinuxGNU, profile: profile, locked: options.locked}, nil
}

func runStableLinuxBuild(root string, options stableNativeOptions, profile cdriver.Profile, profileName string, stdout, stderr io.Writer) int {
	native, err := stableLinuxOptions(options, profile)
	if err != nil {
		return buildFailure(stderr, err)
	}
	compiled, code := compileLinuxWorkspacePolicy(root, native, workspaceCompilePolicy{stableDiagnostics: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	output := path.Join("build", compiled.rootPackageName)
	if err := replaceBuildArtifact(root, output, compiled.artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(stdout, "built: %s (%s, %s, lock %s)\n", output, target.X8664LinuxGNU, profileName, compiled.lockState); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func runStableLinuxRun(root string, options stableNativeOptions, profile cdriver.Profile, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	native, err := stableLinuxOptions(options, profile)
	if err != nil {
		return buildFailure(stderr, err)
	}
	compiled, code := compileLinuxWorkspacePolicy(root, native, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteLinuxTargetWithOptions(compiled.artifact, cdriver.LinuxExecutionOptions{Input: stdin, Arguments: args, WorkingDirectory: root, Timeout: time.Minute})
	if err != nil {
		return buildFailure(stderr, err)
	}
	category, err := linuxTrapCategory(outcome)
	if err != nil {
		return buildFailure(stderr, err)
	}
	programError := outcome.Stderr
	if category != "" {
		programError += fmt.Sprintf("trap: %s\n", category)
	}
	if code := forwardProgramOutput(outcome.Stdout, programError, stdout, stderr); code != ExitSuccess {
		return code
	}
	if category != "" {
		return ExitProgramTrap
	}
	return int(outcome.ExitCode)
}

func linuxTrapCategory(outcome cdriver.LinuxHostOutcome) (string, error) {
	if outcome.TrapCategory == 0 {
		return "", nil
	}
	category, found := windowsHostTrapCategory(int32(productTrapExitBase) + int32(outcome.TrapCategory))
	if !found {
		return "", fmt.Errorf("Linux executable produced unknown trap category")
	}
	return category, nil
}
