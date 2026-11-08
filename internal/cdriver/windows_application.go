package cdriver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	return buildWindowsMSVCTargetParts(tc, profile, header, literals, units, support, timeout, 0, false, false, application)
}

func compileWindowsApplicationResources(tc WindowsHostToolchain, dir string, timeout time.Duration, app *windowsbuild.Project) ([]string, error) {
	if app == nil || len(app.RCInputs) == 0 {
		return nil, nil
	}
	paths := strings.Split(environmentValue(tc.Environment, "PATH"), ";")
	if len(paths) < 2 || !filepath.IsAbs(paths[1]) {
		return nil, fmt.Errorf("resolved SDK resource compiler directory missing")
	}
	rc := filepath.Join(paths[1], "rc.exe")
	if info, err := os.Stat(rc); err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Windows SDK rc.exe missing")
	}
	var names []string
	for i, input := range app.RCInputs {
		name := fmt.Sprintf("fy_resource_%d", i)
		source := filepath.Join(dir, name+".rc")
		output := filepath.Join(dir, name+".res")
		if err := os.WriteFile(source, input.Data, 0600); err != nil {
			return nil, err
		}
		if _, err := runWindowsHostCommandWithLimitAndTimeout(rc, tc.Environment, app.Root, maxProcessOutput, timeout, "/nologo", "/c65001", "/I", app.Root, "/fo", output, source); err != nil {
			return nil, fmt.Errorf("compile resource %s: %w", input.Name, err)
		}
		data, err := readArtifact(output)
		if err != nil || len(data) == 0 {
			return nil, fmt.Errorf("invalid compiled resource %s: %v", input.Name, err)
		}
		names = append(names, name+".res")
	}
	return names, nil
}
