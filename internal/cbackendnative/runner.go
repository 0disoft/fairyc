// Package cbackendnative compiles and executes constrained C fixtures on the declared Linux target.
package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cabifixture"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/target"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v37"
	trapExitBase        = 192
	maxProcessOutput    = 64 * 1024
	executionTimeout    = 10 * time.Second
)

var nativeProfiles = []cdriver.Profile{
	cdriver.LowProfile,
	cdriver.OptimizedProfile,
}

// Run compiles every emission fixture with Clang and GCC at low and optimized profiles, compares
// the Linux process outcome and source-linked drop trace with the MIR-interpreter oracle, executes
// every production unit under the closed address/undefined sanitizer mode, and exercises the
// product build CLI.
func Run(manifestPath string) (Report, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Report{}, fmt.Errorf("native C differential requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	fixtureReport, err := cbackendfixture.Run(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if len(fixtureReport.Failures) != 0 || len(fixtureReport.Emitted) != fixtureReport.CasesRun {
		return Report{}, fmt.Errorf("C emission fixtures are not clean: %s", summarizeFailures(fixtureReport.Failures))
	}
	traceRuntimeSource, err := cruntime.DropTraceSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind scalar runtime: %w", err)
	}
	scalarRuntimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind uninstrumented scalar runtime: %w", err)
	}
	toolchains, err := resolveToolchains()
	if err != nil {
		return Report{}, err
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-native-c-")
	if err != nil {
		return Report{}, fmt.Errorf("create native fixture temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	report := Report{
		SchemaVersion: reportSchemaVersion,
		Target:        string(target.X8664LinuxGNU),
		Libc:          toolchains[0].Libc,
		Sanitizer:     cdriver.AddressUndefinedSanitizerMode,
	}
	for _, toolchain := range toolchains {
		if toolchain.Libc != report.Libc {
			return Report{}, fmt.Errorf("C toolchains disagree on libc: %q and %q", report.Libc, toolchain.Libc)
		}
		report.Compilers = append(report.Compilers, Compiler{Name: toolchain.Name, Version: toolchain.Version})
	}
	for _, profile := range nativeProfiles {
		report.Profiles = append(report.Profiles, profile.Name)
	}
	layoutManifest := filepath.Clean(filepath.Join(filepath.Dir(absoluteManifest), "..", "layout", "manifest-v0.json"))
	layoutTarget, layoutCases, err := layoutfixture.NativeProbeCases(layoutManifest)
	if err != nil {
		return Report{}, fmt.Errorf("load native layout probe cases: %w", err)
	}
	if layoutTarget != target.X8664LinuxGNU {
		return Report{}, fmt.Errorf("native layout probe target is %q, want %q", layoutTarget, target.X8664LinuxGNU)
	}
	layoutProbe, err := generateLayoutProbe(layoutTarget, layoutCases)
	if err != nil {
		return Report{}, fmt.Errorf("generate native layout probe: %w", err)
	}
	report.LayoutCases = slices.Clone(layoutProbe.Cases)
	layoutExecutions, err := runLayoutProbe(temporaryRoot, toolchains, layoutProbe)
	if err != nil {
		return Report{}, err
	}
	report.LayoutProbeExecutions = layoutExecutions
	abiManifest := filepath.Clean(filepath.Join(filepath.Dir(absoluteManifest), "..", "c-abi", "manifest-v0.json"))
	abiTarget, abiCases, err := cabifixture.NativeProbeCases(abiManifest)
	if err != nil {
		return Report{}, fmt.Errorf("load native C ABI probe cases: %w", err)
	}
	if abiTarget != target.X8664LinuxGNU {
		return Report{}, fmt.Errorf("native C ABI probe target is %q, want %q", abiTarget, target.X8664LinuxGNU)
	}
	abiProbes, err := generateABIProbes(abiTarget, abiCases)
	if err != nil {
		return Report{}, fmt.Errorf("generate native C ABI probes: %w", err)
	}
	for _, probe := range abiProbes {
		report.ABICases = append(report.ABICases, probe.ID)
	}
	report.ABIDirections = []string{"clang-caller-gcc-callee", "gcc-caller-clang-callee"}
	abiExecutions, err := runABIProbes(temporaryRoot, toolchains, abiProbes)
	if err != nil {
		return Report{}, err
	}
	report.ABIProbeExecutions = abiExecutions
	systemHeapSource, err := cstdlib.SystemHeapSource()
	if err != nil {
		return Report{}, fmt.Errorf("load SystemHeap implementation: %w", err)
	}
	systemHeapProbe, err := cstdlib.SystemHeapProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load SystemHeap probe: %w", err)
	}
	systemHeapExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "SystemHeap", "system-heap-probe", systemHeapProbe, systemHeapSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.SystemHeapExecutions = systemHeapExecutions
	systemHeapSourceCallExecutions, err := runSystemHeapSourceCallProbe(temporaryRoot, toolchains, systemHeapSource)
	if err != nil {
		return Report{}, err
	}
	report.SystemHeapSourceCallExecutions = systemHeapSourceCallExecutions
	storageSource, err := cstdlib.StorageSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Buffer storage implementation: %w", err)
	}
	bufferProbe, err := cstdlib.BufferProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Buffer probe: %w", err)
	}
	bufferProbeImplementation, err := cstdlib.BufferProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented Buffer implementation: %w", err)
	}
	bufferExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Buffer", "buffer-probe", bufferProbe, bufferProbeImplementation,
	)
	if err != nil {
		return Report{}, err
	}
	report.BufferExecutions = bufferExecutions
	bufferSourceCallExecutions, err := runBufferSourceCallProbe(temporaryRoot, toolchains, storageSource)
	if err != nil {
		return Report{}, err
	}
	report.BufferSourceCallExecutions = bufferSourceCallExecutions
	arenaSource, err := cstdlib.ArenaSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Arena implementation: %w", err)
	}
	arenaProbe, err := cstdlib.ArenaProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Arena probe: %w", err)
	}
	arenaProbeImplementation, err := cstdlib.ArenaProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented Arena implementation: %w", err)
	}
	arenaExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Arena", "arena-probe", arenaProbe, arenaProbeImplementation,
	)
	if err != nil {
		return Report{}, err
	}
	report.ArenaExecutions = arenaExecutions
	arenaSourceCallExecutions, err := runArenaSourceCallProbe(temporaryRoot, toolchains, arenaSource)
	if err != nil {
		return Report{}, err
	}
	report.ArenaSourceCallExecutions = arenaSourceCallExecutions
	textViewSource, err := cstdlib.TextViewSource()
	if err != nil {
		return Report{}, fmt.Errorf("load text view implementation: %w", err)
	}
	textViewProbe, err := cstdlib.TextViewProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load text view probe: %w", err)
	}
	textViewExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "text view", "text-view-probe", textViewProbe, textViewSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.TextViewExecutions = textViewExecutions
	ownedTextSource, err := cstdlib.OwnedTextSource()
	if err != nil {
		return Report{}, fmt.Errorf("load owned text implementation: %w", err)
	}
	ownedTextProbe, err := cstdlib.OwnedTextProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load owned text probe: %w", err)
	}
	ownedTextProbeImplementation, err := cstdlib.OwnedTextProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented owned text implementation: %w", err)
	}
	ownedTextExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "owned text", "owned-text-probe", ownedTextProbe, ownedTextProbeImplementation,
	)
	if err != nil {
		return Report{}, err
	}
	report.OwnedTextExecutions = ownedTextExecutions
	cStrValidationSourceCallExecutions, err := runCStrValidationSourceCallProbe(temporaryRoot, toolchains, ownedTextSource)
	if err != nil {
		return Report{}, err
	}
	report.CStrValidationSourceCallExecutions = cStrValidationSourceCallExecutions
	ownedTextSourceCallExecutions, err := runOwnedTextSourceCallProbe(temporaryRoot, toolchains, ownedTextSource)
	if err != nil {
		return Report{}, err
	}
	report.OwnedTextSourceCallExecutions = ownedTextSourceCallExecutions
	linuxConsoleSource, err := cstdlib.LinuxConsoleSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux console implementation: %w", err)
	}
	linuxConsoleProbe, err := cstdlib.LinuxConsoleProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux console probe: %w", err)
	}
	linuxConsoleExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Linux console", "linux-console-probe", linuxConsoleProbe, linuxConsoleSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.LinuxConsoleExecutions = linuxConsoleExecutions
	linuxConsoleFailureExecutions, err := runLinuxConsoleFailureProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.LinuxConsoleFailureExecutions = linuxConsoleFailureExecutions
	linuxFileSource, err := cstdlib.LinuxFileSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux File implementation: %w", err)
	}
	linuxFileProbe, err := cstdlib.LinuxFileProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux File probe: %w", err)
	}
	linuxFileExecutions, err := runStandardLibraryProbe(temporaryRoot, toolchains, "Linux File", "linux-file-probe", linuxFileProbe, linuxFileSource)
	if err != nil {
		return Report{}, err
	}
	report.LinuxFileExecutions = linuxFileExecutions
	linuxFileSourceCallExecutions, err := runLinuxFileSourceCallProbe(temporaryRoot, toolchains, linuxFileSource)
	if err != nil {
		return Report{}, err
	}
	report.LinuxFileSourceCallExecutions = linuxFileSourceCallExecutions
	foreignCallExecutions, err := runForeignCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCallExecutions = foreignCallExecutions
	foreignCIntCallExecutions, err := runForeignCIntCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCIntCallExecutions = foreignCIntCallExecutions
	foreignCLongCallExecutions, err := runForeignCLongCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCLongCallExecutions = foreignCLongCallExecutions
	foreignCScalarFamilyCallExecutions, err := runForeignCScalarFamilyCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCScalarFamilyCallExecutions = foreignCScalarFamilyCallExecutions
	foreignCSizeTCallExecutions, err := runForeignCSizeTCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCSizeTCallExecutions = foreignCSizeTCallExecutions
	foreignCSSizeTCallExecutions, err := runForeignCSSizeTCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCSSizeTCallExecutions = foreignCSSizeTCallExecutions
	foreignPointerCallExecutions, err := runForeignPointerCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignPointerCallExecutions = foreignPointerCallExecutions
	foreignCStructResultCallExecutions, err := runForeignCStructResultCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCStructResultCallExecutions = foreignCStructResultCallExecutions
	textValidationSourceCallExecutions, err := runTextValidationSourceCallProbe(temporaryRoot, toolchains, textViewSource)
	if err != nil {
		return Report{}, err
	}
	report.TextValidationSourceCallExecutions = textValidationSourceCallExecutions
	textViewOutputSourceCallExecutions, err := runTextViewOutputSourceCallProbe(temporaryRoot, toolchains, textViewSource)
	if err != nil {
		return Report{}, err
	}
	report.TextViewOutputSourceCallExecutions = textViewOutputSourceCallExecutions
	byteStringLiteralSourceCallExecutions, err := runByteStringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ByteStringSourceCallExecutions = byteStringLiteralSourceCallExecutions
	utf8StringLiteralSourceCallExecutions, err := runUTF8StringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.UTF8StringSourceCallExecutions = utf8StringLiteralSourceCallExecutions
	cStringLiteralSourceCallExecutions, err := runCStringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.CStringLiteralSourceCallExecutions = cStringLiteralSourceCallExecutions
	for caseIndex, fixture := range fixtureReport.Emitted {
		if fixture.Unit.Target != string(target.X8664LinuxGNU) || fixture.Unit.Dialect != cbackend.DialectC11 {
			return Report{}, fmt.Errorf("case %s emitted unexpected profile %s/%s", fixture.ID, fixture.Unit.Target, fixture.Unit.Dialect)
		}
		report.Cases = append(report.Cases, fixture.ID)
		expectedExit, err := expectedExitCode(fixture)
		if err != nil {
			return Report{}, err
		}
		expectedDrops, err := expectedDropSiteIDs(fixture)
		if err != nil {
			return Report{}, err
		}
		for compilerIndex, toolchain := range toolchains {
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.Build(toolchain, profile, fixture.TraceUnit.Source, traceRuntimeSource)
				if err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("program-%02d-%d-%d", caseIndex, compilerIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				if err := execute(fixture.ID, executablePath, expectedExit, expectedDrops); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				report.Executions++
				report.DropTraceExecutions++
			}
			sanitizedArtifact, err := cdriver.BuildSanitized(toolchain, fixture.Unit.Source, scalarRuntimeSource)
			if err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			sanitizedPath := filepath.Join(temporaryRoot, fmt.Sprintf("sanitized-%02d-%d", caseIndex, compilerIndex))
			if err := os.WriteFile(sanitizedPath, sanitizedArtifact.Bytes, 0o700); err != nil {
				return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			if err := executeSanitized(fixture.ID, sanitizedPath, expectedExit); err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			report.SanitizerExecutions++
		}
	}
	buildSmoke, err := runBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.BuildSmokeExecutions = buildSmoke
	standardLibraryBuildSmoke, err := runStandardLibraryBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.StandardLibraryBuildSmokeExecutions = standardLibraryBuildSmoke
	publicFileBuildSmoke, err := runPublicFileBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.LinuxPublicFileExecutions = publicFileBuildSmoke
	byteStringLiteralBuildSmoke, err := runByteStringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.ByteStringBuildSmokeExecutions = byteStringLiteralBuildSmoke
	utf8StringLiteralBuildSmoke, err := runUTF8StringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.UTF8StringBuildSmokeExecutions = utf8StringLiteralBuildSmoke
	cStringLiteralBuildSmoke, err := runCStringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.CStringLiteralBuildSmokeExecutions = cStringLiteralBuildSmoke
	return report, nil
}
