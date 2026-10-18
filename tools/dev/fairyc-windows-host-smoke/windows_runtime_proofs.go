package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/windowsabiprobe"
	"github.com/0disoft/fairyc/internal/windowslayoutprobe"
	"github.com/0disoft/fairyc/internal/windowsunwindprobe"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func runWindowsUnwindProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit := windowsunwindprobe.Generate()
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCUnwindProbe(toolchain, profile, unit.CatcherSource, unit.FrameSource, unit.RaiserSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 3 || artifact.Objects[0].Name != "catcher.obj" || artifact.Objects[1].Name != "frame.obj" || artifact.Objects[2].Name != "raiser.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe PE executable image with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64Unwind(artifact.Bytes)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe unwind data with %s: %w", profile.Name, err)
		}
		if report.HandlerInfos == 0 {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s has no handler-bearing unwind records", profile.Name)
		}
		handlerSymbol, ok := unit.HandlerSymbolForProfile(profile.Name)
		if !ok {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe has no handler identity for profile %q", profile.Name)
		}
		handlerReport, err := pecoff.InspectAMD64HandlerLink(artifact.Objects[1].Bytes, artifact.Bytes, artifact.LinkMap, pecoff.AMD64HandlerLinkSpec{
			SourceSymbol: unit.FrameSymbol, SourceObject: unit.FrameObject, HandlerSymbol: handlerSymbol,
		})
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind handler relocation with %s: %w", profile.Name, err)
		}
		if handlerReport.LinkedHandlerRVA != handlerReport.ExpectedHandlerRVA || handlerReport.HandlerMapOwner == "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind handler relocation with %s returned an incomplete receipt", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCUnwindProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

// generatedFairyProbeImplementation builds the test-only standard-library closure for a
// generated-Fairy probe: the existing probe-only SystemHeap allocator adapter, the exact
// standard-library components selected for the emitted C foreign symbols, and the canonical
// scalar runtime for emitted trap calls. No product source, symbol contract, or ordinary
// product linkage is changed.
func generatedFairyProbeImplementation(repositoryRoot string, symbols []string) (string, error) {
	if len(symbols) == 0 {
		return "", fmt.Errorf("generated-Fairy probe closure requires at least one referenced symbol")
	}
	implementation, err := cstdlib.WindowsMSVCSourceForSymbols(symbols)
	if err != nil {
		return "", fmt.Errorf("select generated-Fairy probe standard-library closure: %w", err)
	}
	adapterPath := filepath.Join(repositoryRoot, filepath.FromSlash("internal/cstdlib/stage0_system_heap_probe_allocator.c"))
	adapterBytes, err := os.ReadFile(adapterPath)
	if err != nil {
		return "", fmt.Errorf("read generated-Fairy probe SystemHeap adapter: %w", err)
	}
	adapter := string(adapterBytes)
	runtimeSource, err := cruntime.ScalarSource(generatedFairyProbeTrapExitBase)
	if err != nil {
		return "", fmt.Errorf("render generated-Fairy probe scalar runtime: %w", err)
	}
	return generatedFairyProbeClosureSource(adapter, implementation, runtimeSource)
}

// generatedFairyProbeClosureSource joins the three canonical C sources and rejects incomplete or
// duplicated closures before they reach the native toolchain.
func generatedFairyProbeClosureSource(adapter, implementation, runtimeSource string) (string, error) {
	if adapter == "" || strings.Contains(adapter, "\r") || !strings.HasSuffix(adapter, "\n") ||
		implementation == "" || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") ||
		runtimeSource == "" || strings.Contains(runtimeSource, "\r") || !strings.HasSuffix(runtimeSource, "\n") {
		return "", fmt.Errorf("generated-Fairy probe closure source is empty or non-canonical")
	}
	if !strings.Contains(adapter, "fy_std_system_heap_probe_fail_after_allocations_v0") ||
		!strings.Contains(adapter, "fy_std_system_heap_probe_live_allocations_v0") {
		return "", fmt.Errorf("generated-Fairy probe SystemHeap adapter lost its failure controls")
	}
	if strings.Contains(adapter+implementation, "void fy_runtime_trap(") {
		return "", fmt.Errorf("generated-Fairy probe closure already defines fy_runtime_trap")
	}
	if !strings.Contains(runtimeSource, "void fy_runtime_trap(") {
		return "", fmt.Errorf("generated-Fairy probe scalar runtime lost its trap definition")
	}
	return adapter + "\n" + implementation + "\n" + runtimeSource, nil
}
func runGeneratedFairyStringFromBufferProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextFromBufferManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Buffer-to-String conversion manifest: %w", err)
	}
	checkedInMain, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextFromBufferMainPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Buffer-to-String conversion entry: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-string-from-buffer-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy Buffer-to-String conversion workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": checkedInMain,
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy Buffer-to-String conversion path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy Buffer-to-String conversion path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy Buffer-to-String conversion workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy Buffer-to-String conversion semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy Buffer-to-String conversion source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy Buffer-to-String conversion source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy Buffer-to-String conversion root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy Buffer-to-String conversion C: %w", err)
	}
	probeSource, err := generatedFairyStringFromBufferProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy Buffer-to-String conversion", probeSource, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}
func generatedFairyStringFromBufferProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy Buffer-to-String conversion C must contain a main wrapper")
	}
	for _, required := range []string{
		"fy_std_string_from_buffer_storage_v0(",
		"fy_std_string_into_buffer_storage_v0(",
		"fy_std_string_drop_storage_v0(",
		"fy_std_buffer_drop_v0(",
	} {
		if strings.Count(generated, required) < 2 {
			return "", fmt.Errorf("generated-Fairy Buffer-to-String conversion C lost a required call to %s (expected prototype and body call)", strings.TrimSuffix(required, "("))
		}
	}
	return "#define main fy_generated_string_from_buffer_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(2);
    if (fy_generated_string_from_buffer_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}
func runFairyStructResultProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	probe := foreignstructprobe.Generate()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(probe.FairySource)), source.DefaultLimits())
	if err != nil {
		return 0, fmt.Errorf("parse generated-Fairy struct-result source: %w", err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return 0, fmt.Errorf("generated-Fairy struct-result source is not parser-valid: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return 0, fmt.Errorf("compile generated-Fairy struct-result source: %w", err)
	}
	semanticResult := checked.Semantic
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("generated-Fairy struct-result source did not complete Flow: phase=%q failed=%q diagnostics=%+v unsupported=%+v", semanticResult.CompletedThrough, semanticResult.FailedPhase, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	if !checked.Verified {
		return 0, fmt.Errorf("generated-Fairy struct-result source did not produce verified MIR")
	}
	caller, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, fmt.Errorf("emit generated-Fairy Windows struct-result C: %w", err)
	}
	for _, required := range probe.RequiredCFragments {
		if !strings.Contains(caller.Source, required) {
			return 0, fmt.Errorf("generated-Fairy Windows struct-result C lost required fragment %q", required)
		}
	}
	executions := 0
	for _, consumerMode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		count, runErr := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy struct-result", caller.Source, probe.CompanionC, probe.ExpectedExit, consumerMode)
		if runErr != nil {
			return executions, runErr
		}
		executions += count
	}
	return executions, nil
}
func runWindowsSeparateTranslationUnitProbe(
	toolchain cdriver.WindowsHostToolchain,
	name string,
	probe string,
	implementation string,
	expectedExit int,
	consumerMode cdriver.ConsumerMode,
) (int, error) {
	if name == "" || probe == "" || implementation == "" {
		return 0, fmt.Errorf("Windows separate-translation-unit probe identity and sources must be non-empty")
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(toolchain, profile, consumerMode, probe, implementation)
		if err != nil {
			return executions, fmt.Errorf("build %s probe with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if artifact.ConsumerMode != consumerMode {
			return executions, fmt.Errorf("%s probe with %s/%s lost its consumer-mode identity", name, consumerMode, profile.Name)
		}
		if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
			return executions, fmt.Errorf("%s probe with %s/%s returned unexpected COFF object receipts", name, consumerMode, profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect %s probe PE with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect %s probe unwind data with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute %s probe with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if outcome.ExitCode != int32(expectedExit) || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("%s probe with %s/%s returned %d, stdout %q, and stderr %q", name, consumerMode, profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}
func runWindowsLayoutProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowslayoutprobe.Generate(filepath.FromSlash(windowsLayoutManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC layout probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, unit.Source)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 1 || artifact.Objects[0].Name != "probe.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}
func runWindowsProcessArgsDecodeProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	implementation, err := cstdlib.WindowsProcessArgsSource()
	if err != nil {
		return 0, 0, fmt.Errorf("load Windows process args implementation: %w", err)
	}
	probeSource := implementation + "\n" + windowsProcessArgsDecodeProbeMain
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, probeSource)
		if err != nil {
			return executions, 0, fmt.Errorf("build Windows process args decode probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 1 || artifact.Objects[0].Name != "probe.obj" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, 0, fmt.Errorf("inspect Windows process args decode probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, 0, fmt.Errorf("inspect Windows process args decode probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return executions, 0, fmt.Errorf("execute Windows process args decode probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with %s returned %d, stdout %q, stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
		outcomeWithArgs, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, strings.NewReader(""), processArgsDecodeRoundTripArguments)
		if err != nil {
			return executions, 0, fmt.Errorf("execute Windows process args decode probe with arguments with %s: %w", profile.Name, err)
		}
		if outcomeWithArgs.ExitCode != 0 || outcomeWithArgs.Stdout != "" || outcomeWithArgs.Stderr != "" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with arguments with %s returned %d, stdout %q, stderr %q", profile.Name, outcomeWithArgs.ExitCode, outcomeWithArgs.Stdout, outcomeWithArgs.Stderr)
		}
		executions++
	}
	return executions, executions, nil
}
func runWindowsABIProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowsabiprobe.Generate(filepath.FromSlash(windowsABIManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC ABI probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.CallerSource, unit.CalleeSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}
