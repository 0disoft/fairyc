package atlasmap

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestMapStoreV3ReopensAggregateFactsAndChecksCurrentSource(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadAggregateMapWorkspace(t, used)
		_, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectations, err := CurrentWorkspaceExpectationsV3(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		plan := plans[0]
		expectation := expectations[plan.Header().PackageInstance]
		root := t.TempDir()
		store, err := NewMapStoreV3(root, testMapStoreLimits())
		if err != nil {
			t.Fatal(err)
		}
		checksum, err := store.Publish(plan)
		if err != nil {
			t.Fatal(err)
		}
		if again, err := store.Publish(plan); err != nil || again != checksum {
			t.Fatalf("repeat publication: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		boundary, err := os.OpenRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		store, err = NewMapStoreV3InRoot(boundary, ".", testMapStoreLimits())
		boundary.Close()
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		loaded, found, err := store.LoadCurrent(expectation)
		if err != nil || !found || !reflect.DeepEqual(documentFromPartition(plan), documentFromPartition(loaded)) {
			t.Fatalf("reopen lost facts: %v %v", found, err)
		}
		wrong := expectation
		wrong.GenericInterface[0] ^= 1
		if _, found, err := store.LoadCurrent(wrong); found || !errors.Is(err, ErrIncompatibleMap) {
			t.Fatalf("interface: %v %v", found, err)
		}
		wrong = expectation
		wrong.Current.Source[0] ^= 1
		if _, found, err := store.LoadCurrent(wrong); found || !errors.Is(err, ErrStaleMap) {
			t.Fatalf("source: %v %v", found, err)
		}
		// Reload changed on-disk template source, rather than manufacture its expected hash.
		path := filepath.Join(workspace.Root, "src", "main.fy")
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(original), "return value;", "let T copied = value; return copied;", 1)
		if changed == string(original) {
			t.Fatal("template edit did not activate")
		}
		if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
			t.Fatal(err)
		}
		current, err := workspaceloader.Load(workspace.Root)
		if err != nil {
			t.Fatal(err)
		}
		input, err := current.SemanticInput()
		if err != nil {
			t.Fatal(err)
		}
		coordinates, err := current.PackageCoordinates()
		if err != nil {
			t.Fatal(err)
		}
		checked, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || !checked.Verified {
			t.Fatalf("changed source: %v", err)
		}
		currentExpectations, err := CurrentWorkspaceExpectationsV3(current, checked, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		if len(currentExpectations) != 1 {
			t.Fatal("unexpected reloaded package set")
		}
		var newExpected LoadExpectation
		for _, value := range currentExpectations {
			newExpected = value
		}
		if newExpected.Current.Source == expectation.Current.Source {
			t.Fatal("source identity did not change")
		}
		_, found, err = store.LoadCurrent(newExpected)
		if newExpected.Current.Instance == expectation.Current.Instance {
			if found || !errors.Is(err, ErrStaleMap) {
				t.Fatalf("reloaded stale source: %v %v", found, err)
			}
		} else if found || err != nil {
			t.Fatalf("changed package instance must miss: %v %v", found, err)
		}
		generation := filepath.Join(root, mapStoreLayoutDirectoryV3, plan.Header().PackageInstance, mapGenerationName(checksum))
		if err := os.WriteFile(generation, []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, found, err := store.LoadCurrent(expectation); found || !errors.Is(err, ErrStoredMapCorrupt) {
			t.Fatalf("corrupt: %v %v", found, err)
		}
		if _, err := os.Stat(generation); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("corrupt generation retained")
		}
		if _, err := store.Publish(plan); err != nil {
			t.Fatal(err)
		}
		if _, found, err := store.LoadCurrent(expectation); err != nil || !found {
			t.Fatalf("repair: %v %v", found, err)
		}
	}
}

func TestMapStoreV3RetentionAndCorruptionStayInSelectedVersion(t *testing.T) {
	workspace, result := loadQuickstart(t)
	_, v1plans, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	_, v2plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	oldFiles := map[string][]byte{}
	for i, open := range []func(string, MapStoreLimits) (*MapStore, error){NewMapStore, NewMapStoreV2} {
		store, err := open(root, testMapStoreLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		plan := [][]Partition{v1plans, v2plans}[i][0]
		checksum, err := store.Publish(plan)
		if err != nil {
			t.Fatal(err)
		}
		layout := []string{mapStoreLayoutDirectory, mapStoreLayoutDirectoryV2}[i]
		path := filepath.Join(root, layout, plan.Header().PackageInstance, mapGenerationName(checksum))
		encoded, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		oldFiles[path] = encoded
	}
	workspace, result = loadAggregateMapWorkspace(t, true)
	_, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectationsV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	plan := plans[0]
	limits := testMapStoreLimits()
	limits.MaxGenerations = 1
	store, err := NewMapStoreV3(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, old := range []Partition{v1plans[0], v2plans[0]} {
		if _, err := store.Publish(old); err == nil {
			t.Fatal("v3 published old partition")
		}
	}
	first, err := store.Publish(plan)
	if err != nil {
		t.Fatal(err)
	}
	changed := plan
	build := testCompilerBuild()
	build[0] ^= 1
	changed.data.header.CompilerBuild = build.String()
	if _, err := store.Publish(changed); err != nil {
		t.Fatal(err)
	}
	stats, err := store.Stats()
	if err != nil || stats.Generations != 1 || stats.Evictions != 1 {
		t.Fatalf("retention: %+v %v", stats, err)
	}
	firstPath := filepath.Join(root, mapStoreLayoutDirectoryV3, plan.Header().PackageInstance, mapGenerationName(first))
	if _, err := os.Stat(firstPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old v3 generation not evicted")
	}
	for path, encoded := range oldFiles {
		copyPath := filepath.Join(root, mapStoreLayoutDirectoryV3, plan.Header().PackageInstance, filepath.Base(path))
		if err := os.WriteFile(copyPath, encoded, 0600); err != nil {
			t.Fatal(err)
		}
		_, _, _ = store.LoadCurrent(expectations[plan.Header().PackageInstance])
		if _, err := os.Stat(copyPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("v3 accepted copied legacy bytes")
		}
		current, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(current, encoded) {
			t.Fatal("v3 touched legacy generation")
		}
	}
	limits.MaxMapBytes = 1
	tiny, err := NewMapStoreV3(t.TempDir(), limits)
	if err != nil {
		t.Fatal(err)
	}
	defer tiny.Close()
	if _, err := tiny.Publish(plan); !errors.Is(err, ErrMapTooLarge) {
		t.Fatalf("byte limit: %v", err)
	}
}

func TestMapStoreV3RejectsInvalidVersionTraversalAndAlias(t *testing.T) {
	root := t.TempDir()
	uncreated := filepath.Join(root, "invalid")
	if store, err := newMapStore(uncreated, testMapStoreLimits(), mapStoreVersion(99)); err == nil {
		store.Close()
		t.Fatal("unknown version accepted")
	}
	if _, err := os.Stat(uncreated); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid version performed I/O")
	}
	boundary, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	if store, err := NewMapStoreV3InRoot(boundary, "../escape", testMapStoreLimits()); err == nil {
		store.Close()
		t.Fatal("traversal accepted")
	}
	if err := os.Mkdir(filepath.Join(root, mapStoreLayoutDirectoryV2), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, mapStoreLayoutDirectoryV2), filepath.Join(root, mapStoreLayoutDirectoryV3)); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	if store, err := NewMapStoreV3InRoot(boundary, ".", testMapStoreLimits()); err == nil {
		store.Close()
		t.Fatal("version alias accepted")
	}
}
