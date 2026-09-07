package atlasmap

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMapStoreV2PersistsGenericFactsAndIsolatesV1(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadGenericPlanWorkspace(t, used)
		_, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectations, err := CurrentWorkspaceExpectationsV2(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		plan := plans[0]
		expectation := expectations[plan.Header().PackageInstance]
		root := t.TempDir()
		v1, err := NewMapStore(root, testMapStoreLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer v1.Close()
		v2, err := NewMapStoreV2(root, testMapStoreLimits())
		if err != nil {
			t.Fatal(err)
		}
		checksum, err := v2.Publish(plan)
		if err != nil {
			t.Fatal(err)
		}
		if again, err := v2.Publish(plan); err != nil || again != checksum {
			t.Fatalf("repeated publication: %v", err)
		}
		if _, err := v1.Publish(plan); err == nil {
			t.Fatal("v1 published a v2 partition")
		}
		if _, found, err := v1.LoadCurrent(expectation); err != nil || found {
			t.Fatalf("v1 saw v2 storage: %v %v", found, err)
		}
		if err := v2.Close(); err != nil {
			t.Fatal(err)
		}
		boundary, err := os.OpenRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		v2, err = NewMapStoreV2InRoot(boundary, ".", testMapStoreLimits())
		boundary.Close()
		if err != nil {
			t.Fatal(err)
		}
		defer v2.Close()
		loaded, found, err := v2.LoadCurrent(expectation)
		if err != nil || !found || !reflect.DeepEqual(documentFromPartition(loaded), documentFromPartition(plan)) {
			t.Fatalf("reopened generic map: %v %v", found, err)
		}
		wrong := expectation
		wrong.GenericInterface[0] ^= 1
		if _, found, err := v2.LoadCurrent(wrong); found || !errors.Is(err, ErrIncompatibleMap) {
			t.Fatalf("wrong interface: %v %v", found, err)
		}
		wrong = expectation
		wrong.Current.Source[0] ^= 1
		if _, found, err := v2.LoadCurrent(wrong); found || !errors.Is(err, ErrStaleMap) {
			t.Fatalf("stale source: %v %v", found, err)
		}
		path := filepath.Join(root, mapStoreLayoutDirectoryV2, plan.Header().PackageInstance, mapGenerationName(checksum))
		if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, found, err := v2.LoadCurrent(expectation); found || !errors.Is(err, ErrStoredMapCorrupt) {
			t.Fatalf("corruption: %v %v", found, err)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("corrupt generation retained: %v", err)
		}
		if _, err := v2.Publish(plan); err != nil {
			t.Fatal(err)
		}
		if _, found, err := v2.LoadCurrent(expectation); err != nil || !found {
			t.Fatalf("repair: %v %v", found, err)
		}
	}
}

func TestMapStoreV2RetentionDoesNotTouchV1(t *testing.T) {
	workspace, result := loadQuickstart(t)
	_, legacy, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	_, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectationsV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	v1, err := NewMapStore(root, testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer v1.Close()
	checksum, err := v1.Publish(legacy[0])
	if err != nil {
		t.Fatal(err)
	}
	limits := testMapStoreLimits()
	limits.MaxGenerations = 1
	v2, err := NewMapStoreV2(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer v2.Close()
	if _, err := v2.Publish(legacy[0]); err == nil {
		t.Fatal("v2 published v1 data")
	}
	for _, plan := range plans {
		if _, err := v2.Publish(plan); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := v2.Stats()
	if err != nil || stats.Generations != 1 || stats.Evictions == 0 {
		t.Fatalf("v2 retention: %+v %v", stats, err)
	}
	legacyPath := filepath.Join(root, mapStoreLayoutDirectory, legacy[0].Header().PackageInstance, mapGenerationName(checksum))
	bytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal("v2 retention removed v1 map")
	}
	contaminated := filepath.Join(root, mapStoreLayoutDirectoryV2, legacy[0].Header().PackageInstance)
	if err := os.MkdirAll(contaminated, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contaminated, mapGenerationName(checksum)), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _ = v2.LoadCurrent(expectations[legacy[0].Header().PackageInstance])
	if _, err := os.Stat(filepath.Join(contaminated, mapGenerationName(checksum))); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("v2 retained copied v1 generation")
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatal("v2 cleanup touched original v1 map")
	}
	limits.MaxMapBytes = 1
	tiny, err := NewMapStoreV2(t.TempDir(), limits)
	if err != nil {
		t.Fatal(err)
	}
	defer tiny.Close()
	if _, err := tiny.Publish(plans[0]); !errors.Is(err, ErrMapTooLarge) {
		t.Fatalf("v2 byte bound: %v", err)
	}
}

func TestMapStoreV2RejectsTraversalAndVersionAlias(t *testing.T) {
	root := t.TempDir()
	boundary, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	if store, err := NewMapStoreV2InRoot(boundary, "../escape", testMapStoreLimits()); err == nil {
		store.Close()
		t.Fatal("parent traversal accepted")
	}
	if err := os.Mkdir(filepath.Join(root, mapStoreLayoutDirectory), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, mapStoreLayoutDirectory), filepath.Join(root, mapStoreLayoutDirectoryV2)); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	if store, err := NewMapStoreV2InRoot(boundary, ".", testMapStoreLimits()); err == nil {
		store.Close()
		t.Fatal("version directory alias accepted")
	}
}
