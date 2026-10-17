package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"os"
	"path/filepath"
	"strings"
)

func runForeignSourceProbe(
	temporaryRoot string,
	toolchains []cdriver.Toolchain,
	probeID string,
	fairySource string,
	companionSource string,
	expectedExit int,
	requiredC []string,
) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("%s probe requires resolved Clang then GCC toolchains", probeID)
	}
	if probeID == "" || fairySource == "" || companionSource == "" || len(requiredC) == 0 {
		return 0, fmt.Errorf("foreign source probe identity, sources, and C expectations must be non-empty")
	}
	sourcePath := "main.fy"
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(sourcePath, []byte(fairySource)), source.DefaultLimits())
	if err != nil {
		return 0, fmt.Errorf("parse %s FY source: %w", probeID, err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return 0, fmt.Errorf("%s FY source is not parser-valid: %+v", probeID, parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: sourcePath, Module: parsed.AST}})
	if err != nil {
		return 0, fmt.Errorf("compile %s FY source: %w", probeID, err)
	}
	semanticResult := checked.Semantic
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("%s FY source did not complete Flow: phase=%q diagnostics=%+v unsupported=%+v", probeID, semanticResult.CompletedThrough, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	if !checked.Verified {
		return 0, fmt.Errorf("%s FY source did not produce verified MIR", probeID)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile())
	if err != nil {
		return 0, fmt.Errorf("emit %s C: %w", probeID, err)
	}
	for _, required := range requiredC {
		if !strings.Contains(unit.Source, required) {
			return 0, fmt.Errorf("%s C emission lost required fragment %q", probeID, required)
		}
	}
	runtimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return 0, fmt.Errorf("bind %s scalar runtime: %w", probeID, err)
	}
	directions := [][2]int{{0, 1}, {1, 0}}
	executions := 0
	for directionIndex, direction := range directions {
		caller := toolchains[direction[0]]
		callee := toolchains[direction[1]]
		for profileIndex, profile := range nativeProfiles {
			artifact, buildErr := cdriver.BuildCrossCompilerWithRuntime(caller, callee, profile, unit.Source, companionSource, runtimeSource)
			if buildErr != nil {
				return 0, fmt.Errorf("%s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, buildErr)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("%s-%d-%d", probeID, directionIndex, profileIndex))
			if writeErr := os.WriteFile(executablePath, artifact.Bytes, 0o700); writeErr != nil {
				return 0, fmt.Errorf("materialize %s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, writeErr)
			}
			if executeErr := execute(probeID, executablePath, expectedExit, nil); executeErr != nil {
				return 0, fmt.Errorf("%s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, executeErr)
			}
			executions++
		}
	}
	return executions, nil
}

func runStandardLibraryProbe(
	temporaryRoot string,
	toolchains []cdriver.Toolchain,
	name string,
	filePrefix string,
	probeSource string,
	implementationSource string,
) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("%s probe requires resolved Clang then GCC toolchains", name)
	}
	if name == "" || filePrefix == "" || probeSource == "" || implementationSource == "" {
		return 0, fmt.Errorf("standard-library probe identity and sources must be non-empty")
	}
	executions := 0
	for compilerIndex, toolchain := range toolchains {
		for profileIndex, profile := range nativeProfiles {
			artifact, err := cdriver.Build(toolchain, profile, probeSource, implementationSource)
			if err != nil {
				return 0, fmt.Errorf("%s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("%s-%d-%d", filePrefix, compilerIndex, profileIndex))
			if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
				return 0, fmt.Errorf("materialize %s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			if err := execute(name+" probe", executablePath, 0, nil); err != nil {
				return 0, fmt.Errorf("%s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}
