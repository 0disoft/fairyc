package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

func runLinuxConsoleFailureProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	implementation, err := cstdlib.LinuxConsoleFailureProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Linux console failure implementation: %w", err)
	}
	probe, err := cstdlib.LinuxConsoleFailureProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Linux console failure probe: %w", err)
	}
	return runStandardLibraryProbe(
		temporaryRoot, toolchains, "Linux console failure", "linux-console-failure-probe", probe, implementation,
	)
}
