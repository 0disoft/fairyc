package atlasmap

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	mapCrashHelperEnv = "FAIRYC_TEST_FYMAP_CRASH_HELPER"
	mapCrashRootEnv   = "FAIRYC_TEST_FYMAP_CRASH_ROOT"
	mapCrashMarkerEnv = "FAIRYC_TEST_FYMAP_CRASH_MARKER"
)

func TestMapStoreProcessInterruptionLeavesOnlyMissOrVerifiedHit(t *testing.T) {
	for _, stage := range []string{"temporary-closed", "published"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(t.TempDir(), "marker")
			command := exec.Command(os.Args[0], "-test.run=^TestMapStoreCrashHelper$", "-test.v=false")
			command.Env = append(os.Environ(),
				mapCrashHelperEnv+"="+stage,
				mapCrashRootEnv+"="+root,
				mapCrashMarkerEnv+"="+marker,
			)
			var output bytes.Buffer
			command.Stdout = &output
			command.Stderr = &output
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- command.Wait() }()
			deadline := time.NewTimer(15 * time.Second)
			ticker := time.NewTicker(10 * time.Millisecond)
			marked := false
			for !marked {
				select {
				case err := <-done:
					ticker.Stop()
					deadline.Stop()
					t.Fatalf("crash helper exited before marker: %v\n%s", err, output.String())
				case <-ticker.C:
					if _, err := os.Stat(marker); err == nil {
						marked = true
					}
				case <-deadline.C:
					ticker.Stop()
					_ = command.Process.Kill()
					<-done
					t.Fatalf("crash helper marker timeout\n%s", output.String())
				}
			}
			ticker.Stop()
			if !deadline.Stop() {
				select {
				case <-deadline.C:
				default:
				}
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			<-done

			workspace, result := loadQuickstart(t)
			compilerBuild := testCompilerBuild()
			_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
			if err != nil {
				t.Fatal(err)
			}
			partition := partitionWithFunction(t, partitions)
			current, err := CurrentWorkspacePackages(workspace)
			if err != nil {
				t.Fatal(err)
			}
			limits := testMapStoreLimits()
			limits.TempMaxAge = time.Nanosecond
			store, err := NewMapStore(root, limits)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			_, found, err := store.LoadCurrent(expectationForPartition(t, partition, current, compilerBuild))
			if stage == "temporary-closed" {
				if err != nil || found {
					t.Fatalf("pre-publication interruption = %v, %v", found, err)
				}
			} else if err != nil || !found {
				t.Fatalf("post-publication interruption = %v, %v", found, err)
			}
			if err := store.Sweep(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMapStoreCrashHelper(t *testing.T) {
	stage := os.Getenv(mapCrashHelperEnv)
	if stage == "" {
		t.Skip("subprocess helper")
	}
	workspace, result := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	partition := partitionWithFunction(t, partitions)
	store, err := NewMapStore(os.Getenv(mapCrashRootEnv), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	block := func(path string) {
		if err := os.WriteFile(os.Getenv(mapCrashMarkerEnv), []byte(path), 0o600); err != nil {
			panic(fmt.Sprintf("write crash marker: %v", err))
		}
		select {}
	}
	hooks := mapPublishHooks{}
	switch stage {
	case "temporary-closed":
		hooks.afterTemporaryClosed = block
	case "published":
		hooks.afterPublished = block
	default:
		t.Fatalf("unknown crash stage %q", stage)
	}
	if _, err := store.publish(partition, hooks); err != nil {
		t.Fatal(err)
	}
	t.Fatal("crash helper publication returned")
}
