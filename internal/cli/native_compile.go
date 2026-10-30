package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
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
	build, err := emitNativeBuildUnit(program, entry, cbackend.Profile{Target: options.targetID, Dialect: cbackend.DialectC11, PackageTrapSources: policy.packageTrapSources}, sdkStorage)
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return nativeCompilation{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" C-emission", capability)
		}
		return nativeCompilation{}, internalError(stderr)
	}
	unit := build.Unit
	if unit.Target != string(options.targetID) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return nativeCompilation{}, internalError(stderr)
	}
	support, err := nativesupport.Source(options.targetID, unit.ForeignSymbols, nativesupport.Options{SDKStorage: sdkStorage, TrapReceipt: policy.runTrapReceipt, CandidateHost: true})
	if err != nil {
		return nativeCompilation{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" support linkage", err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(options.targetID, options.compilerName)
	if err != nil {
		return nativeCompilation{}, buildFailure(stderr, err)
	}
	artifact, err := toolchain.Build(nativeBuildOptions(build, options.profile, support, policy.runTrapReceipt))
	if err != nil {
		return nativeCompilation{}, buildFailure(stderr, err)
	}
	return nativeCompilation{artifact: artifact, toolchain: toolchain, target: native}, ExitSuccess
}

func emitNativeBuildUnit(program mir.Program, entry string, profile cbackend.Profile, sdkStorage bool) (cbackend.BuildUnit, error) {
	if profile.Target == target.X8664LinuxGNU {
		return cbackend.EmitBuild(program, entry, profile, sdkStorage)
	}
	emit := cbackend.Emit
	if sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(program, entry, profile)
	return cbackend.BuildUnit{Unit: unit}, err
}

func nativeBuildOptions(build cbackend.BuildUnit, profile cdriver.Profile, support string, trap bool) cdriver.NativeBuildOptions {
	options := cdriver.NativeBuildOptions{Profile: profile, UnitSource: build.Unit.Source, RuntimeSource: support, TrapControl: trap}
	if build.Header != "" {
		options.Partitions = &cdriver.CPartitions{Header: build.Header, Sources: build.Sources}
	}
	return options
}
