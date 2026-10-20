package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

// runCacheMeasurementsOnly reuses the full measurement lane's cache oracles without
// native compilation, Atlas probes, or unrelated correctness smokes.
func runCacheMeasurementsOnly(repositoryRoot string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("cache-only measurement requires windows/amd64")
	}
	root, err := os.MkdirTemp("", "fairyc-cache-only-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	executable := filepath.Join(root, "fairyc.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-trimpath", "-o", executable, "./cmd/fairyc")
	build.Dir = repositoryRoot
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build cache measurement compiler: %w", err)
	}
	fixture := filepath.Join(repositoryRoot, "examples", "workspace_quickstart")
	resolution, err := observeMonotonicClockResolution()
	if err != nil {
		return err
	}
	processRoot, err := copyWorkspaceMeasurementFixture(fixture)
	if err != nil {
		return err
	}
	defer os.RemoveAll(processRoot)
	process, err := measureWorkspaceProcessCache(processRoot, resolution)
	if err != nil {
		return err
	}
	data, err := buildmeasure.MarshalCache(process)
	if err != nil {
		return err
	}
	if _, err := fmt.Printf("Fairy C process-cache measurement: %s\n", data); err != nil {
		return err
	}
	edits, err := measureWorkspaceProcessCacheEdits(fixture, resolution)
	if err != nil {
		return err
	}
	data, err = buildmeasure.MarshalCacheEdits(edits)
	if err != nil {
		return err
	}
	if _, err := fmt.Printf("Fairy C process-cache edit measurement: %s\n", data); err != nil {
		return err
	}
	persistent, err := measureStandaloneWorkspaceCache(executable, fixture, resolution)
	if err != nil {
		return err
	}
	data, err = buildmeasure.MarshalPersistentCacheProcess(persistent)
	if err != nil {
		return err
	}
	if _, err := fmt.Printf("Fairy C persistent-cache process measurement: %s\n", data); err != nil {
		return err
	}
	growth, err := measureStandaloneWorkspaceCacheGrowth(executable, fixture)
	if err != nil {
		return err
	}
	data, err = buildmeasure.MarshalPersistentCacheGrowth(growth)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("Fairy C persistent-cache growth measurement: %s\n", data)
	return err
}
