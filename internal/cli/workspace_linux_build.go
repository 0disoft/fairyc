package cli

import (
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func compileLinuxWorkspace(sourceRoot string, options workspaceTargetOptions, runReceipt bool, stderr io.Writer) (cdriver.Artifact, string, int) {
	compiled, code := compileLinuxWorkspacePolicy(sourceRoot, options, workspaceCompilePolicy{runTrapReceipt: runReceipt}, stderr)
	return compiled.artifact, compiled.lockState, code
}

type compiledLinuxWorkspace struct {
	artifact                   cdriver.Artifact
	lockState, rootPackageName string
}

func compileLinuxWorkspacePolicy(sourceRoot string, options workspaceTargetOptions, policy workspaceCompilePolicy, stderr io.Writer) (compiledLinuxWorkspace, int) {
	if options.targetID != target.X8664LinuxGNU {
		return compiledLinuxWorkspace{}, invocationError(stderr, fmt.Sprintf("Linux workspace build requires %s", target.X8664LinuxGNU))
	}
	lowered, code := lowerWorkspaceExecutable(sourceRoot, options.locked, policy, "Linux workspace C-emission", stderr)
	if code != ExitSuccess {
		return compiledLinuxWorkspace{}, code
	}
	emit := cbackend.Emit
	if lowered.sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return compiledLinuxWorkspace{}, workspaceSubsetCapabilityError(policy, stderr, "Linux workspace C-emission", capability)
		}
		return compiledLinuxWorkspace{}, internalError(stderr)
	}
	if unit.Target != string(target.X8664LinuxGNU) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return compiledLinuxWorkspace{}, internalError(stderr)
	}
	supportSource := linuxVectorSupportSource
	if policy.runTrapReceipt {
		supportSource = linuxVectorRunSupportSource
	}
	support, err := supportSource(unit.ForeignSymbols)
	if err != nil {
		return compiledLinuxWorkspace{}, workspaceSubsetCapabilityError(policy, stderr, "Linux workspace support linkage", err)
	}
	compilerName := options.compilerName
	if compilerName == "" {
		for _, name := range []string{"gcc", "clang"} {
			if _, err := exec.LookPath(name); err == nil {
				compilerName = name
				break
			}
		}
		if compilerName == "" {
			return compiledLinuxWorkspace{}, buildFailure(stderr, fmt.Errorf("Linux native commands require GCC or Clang on PATH"))
		}
	}
	toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, compilerName)
	if err != nil {
		return compiledLinuxWorkspace{}, buildFailure(stderr, err)
	}
	artifact, err := cdriver.Build(toolchain, options.profile, unit.Source, support)
	if err != nil {
		return compiledLinuxWorkspace{}, buildFailure(stderr, err)
	}
	return compiledLinuxWorkspace{artifact: artifact, lockState: lowered.lockState, rootPackageName: lowered.rootPackageName}, ExitSuccess
}
