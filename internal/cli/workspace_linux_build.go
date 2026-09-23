package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func compileLinuxWorkspace(sourceRoot string, options workspaceTargetOptions, stderr io.Writer) (cdriver.Artifact, string, int) {
	if options.targetID != target.X8664LinuxGNU {
		return cdriver.Artifact{}, "", invocationError(stderr, fmt.Sprintf("Linux workspace build requires %s", target.X8664LinuxGNU))
	}
	lowered, code := lowerWorkspaceExecutable(sourceRoot, options.locked, workspaceCompilePolicy{}, "Linux workspace C-emission", stderr)
	if code != ExitSuccess {
		return cdriver.Artifact{}, "", code
	}
	emit := cbackend.Emit
	if lowered.sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return cdriver.Artifact{}, "", subsetCapabilityError(stderr, "Linux workspace C-emission", capability)
		}
		return cdriver.Artifact{}, "", internalError(stderr)
	}
	if unit.Target != string(target.X8664LinuxGNU) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return cdriver.Artifact{}, "", internalError(stderr)
	}
	support, err := linuxVectorSupportSource(unit.ForeignSymbols)
	if err != nil {
		return cdriver.Artifact{}, "", subsetCapabilityError(stderr, "Linux workspace support linkage", err)
	}
	toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, options.compilerName)
	if err != nil {
		return cdriver.Artifact{}, "", buildFailure(stderr, err)
	}
	artifact, err := cdriver.Build(toolchain, options.profile, unit.Source, support)
	if err != nil {
		return cdriver.Artifact{}, "", buildFailure(stderr, err)
	}
	return artifact, lowered.lockState, ExitSuccess
}
