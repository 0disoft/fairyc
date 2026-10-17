package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"path/filepath"
)

func runABIProbes(temporaryRoot string, toolchains []cdriver.Toolchain, probes []abiProbeCase) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("native C ABI probes require resolved Clang then GCC toolchains")
	}
	directions := [][2]int{{0, 1}, {1, 0}}
	executions := 0
	for caseIndex, probe := range probes {
		for directionIndex, direction := range directions {
			caller := toolchains[direction[0]]
			callee := toolchains[direction[1]]
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.BuildCrossCompiler(caller, callee, profile, probe.CallerSource, probe.CalleeSource)
				if err != nil {
					return 0, fmt.Errorf("C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("abi-probe-%02d-%d-%d", caseIndex, directionIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return 0, fmt.Errorf("materialize C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				if err := execute("C ABI probe "+probe.ID, executablePath, 0, nil); err != nil {
					return 0, fmt.Errorf("C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				executions++
			}
		}
	}
	return executions, nil
}

func runLayoutProbe(temporaryRoot string, toolchains []cdriver.Toolchain, probe layoutProbeUnit) (int, error) {
	executions := 0
	for compilerIndex, toolchain := range toolchains {
		for profileIndex, profile := range nativeProfiles {
			artifact, err := cdriver.Build(toolchain, profile, probe.Source, layoutProbeRuntimeSource)
			if err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("layout-probe-%d-%d", compilerIndex, profileIndex))
			if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
				return 0, fmt.Errorf("materialize layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			if err := execute("layout-probe", executablePath, 0, nil); err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}
