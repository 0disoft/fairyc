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

// A plan retains only C inputs and target facts, never frontend/MIR state.
type nativeBuildPlan struct {
	options   cdriver.NativeBuildOptions
	toolchain cdriver.NativeToolchain
	target    cdriver.NativeTarget
}

func compileNativeProgram(program mir.Program, entry string, options workspaceTargetOptions, policy workspaceCompilePolicy, sdkStorage bool, stderr io.Writer) (nativeCompilation, int) {
	plan, code := prepareNativeProgram(program, entry, options, policy, sdkStorage, stderr)
	if code != ExitSuccess {
		return nativeCompilation{}, code
	}
	return plan.compile(stderr)
}

func prepareNativeProgram(program mir.Program, entry string, options workspaceTargetOptions, policy workspaceCompilePolicy, sdkStorage bool, stderr io.Writer) (nativeBuildPlan, int) {
	native, found := cdriver.LookupNativeTarget(options.targetID)
	if !found {
		return nativeBuildPlan{}, invocationError(stderr, fmt.Sprintf("unsupported native target %q", options.targetID))
	}
	build, err := emitNativeBuildUnit(program, entry, cbackend.Profile{Target: options.targetID, Dialect: cbackend.DialectC11, PackageTrapSources: policy.packageTrapSources}, sdkStorage)
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return nativeBuildPlan{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" C-emission", capability)
		}
		return nativeBuildPlan{}, internalError(stderr)
	}
	unit := build.Unit
	if unit.Target != string(options.targetID) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return nativeBuildPlan{}, internalError(stderr)
	}
	support, err := nativesupport.Source(options.targetID, unit.ForeignSymbols, nativesupport.Options{SDKStorage: sdkStorage, TrapReceipt: policy.runTrapReceipt, CandidateHost: true})
	if err != nil {
		return nativeBuildPlan{}, workspaceSubsetCapabilityError(policy, stderr, native.EmissionLabel+" support linkage", err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(options.targetID, options.compilerName)
	if err != nil {
		return nativeBuildPlan{}, buildFailure(stderr, err)
	}
	return nativeBuildPlan{options: nativeBuildOptions(build, options.profile, support, policy.runTrapReceipt), toolchain: toolchain, target: native}, ExitSuccess
}

func (plan nativeBuildPlan) compile(stderr io.Writer) (nativeCompilation, int) {
	compiled := nativeCompilation{toolchain: plan.toolchain, target: plan.target}
	artifact, err := plan.toolchain.Build(plan.options)
	if err != nil {
		return nativeCompilation{}, buildFailure(stderr, err)
	}
	compiled.artifact = artifact
	return compiled, ExitSuccess
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
