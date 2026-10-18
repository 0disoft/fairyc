package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/nativesupport"
)

type nativeCompilation struct {
	artifact  cdriver.NativeArtifact
	toolchain cdriver.NativeToolchain
	target    cdriver.NativeTarget
}

func compileNativeProgram(program mir.Program, entry string, options workspaceTargetOptions, policy workspaceCompilePolicy, sdkStorage bool, stderr io.Writer) (nativeCompilation, int) {
	native, found := cdriver.LookupNativeTarget(options.targetID)
	if !found {
		return nativeCompilation{}, invocationError(stderr, fmt.Sprintf("unsupported native target %q", options.targetID))
	}
	emit := cbackend.Emit
	if sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(program, entry, cbackend.Profile{Target: options.targetID, Dialect: cbackend.DialectC11})
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return nativeCompilation{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" C-emission", capability)
		}
		return nativeCompilation{}, internalError(stderr)
	}
	if unit.Target != string(options.targetID) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return nativeCompilation{}, internalError(stderr)
	}
	support, err := nativesupport.Source(options.targetID, unit.ForeignSymbols, nativesupport.Options{SDKStorage: sdkStorage, TrapReceipt: policy.runTrapReceipt})
	if err != nil {
		return nativeCompilation{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" support linkage", err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(options.targetID, options.compilerName)
	if err != nil {
		return nativeCompilation{}, buildFailure(stderr, err)
	}
	artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: options.profile, UnitSource: unit.Source, RuntimeSource: support, TrapControl: policy.runTrapReceipt})
	if err != nil {
		return nativeCompilation{}, buildFailure(stderr, err)
	}
	return nativeCompilation{artifact: artifact, toolchain: toolchain, target: native}, ExitSuccess
}
