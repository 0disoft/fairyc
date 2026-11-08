package cdriver

import (
	"fmt"
	"time"

	"github.com/0disoft/fairyc/internal/windowsbuild"
)

func applicationSubsystem(p *windowsbuild.Project) uint16 {
	if p != nil && p.Subsystem == "windows" {
		return 2
	}
	return 3
}

// BuildWindowsApplicationParts consumes application native inputs only when
// explicitly selected by the workspace caller. Compiler-host builds stay closed.
func BuildWindowsApplicationParts(tc WindowsHostToolchain, profile Profile, header, literals string, units []string, support string, timeout time.Duration, application *windowsbuild.Project) (WindowsHostArtifact, error) {
	if application == nil {
		return WindowsHostArtifact{}, fmt.Errorf("Windows application settings required")
	}
	if len(application.RCInputs) != 0 {
		return WindowsHostArtifact{}, fmt.Errorf("resource linking not implemented in this stage")
	}
	return buildWindowsMSVCTargetParts(tc, profile, header, literals, units, support, timeout, 0, false, false, application)
}
