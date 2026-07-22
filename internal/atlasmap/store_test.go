package atlasmap

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestMapStorePublishesConvergesAndLoadsCurrentWorkspace(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	current, err := CurrentWorkspacePackages(workspace)
	if err != nil {
		t.Fatal(err)
	}
	derivedExpectations, err := CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil || len(derivedExpectations) != len(partitions) {
		t.Fatalf("CurrentWorkspaceExpectations() = %d, %v", len(derivedExpectations), err)
	}
	root := t.TempDir()
	store, err := NewMapStore(root, testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, partition := range partitions {
		header := partition.Header()
		first, err := store.Publish(partition)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
		second, err := store.Publish(partition)
		if err != nil || second != first {
			t.Fatalf("repeated Publish() = %s, %v", second, err)
		}
		expectation := expectationForPartition(t, partition, current, compilerBuild)
		if !reflect.DeepEqual(derivedExpectations[header.PackageInstance], expectation) {
			t.Fatalf("derived expectation differs for %s", header.PackageInstance)
		}
		loaded, found, err := store.LoadCurrent(expectation)
		if err != nil || !found {
			t.Fatalf("LoadCurrent() = %v, %v", found, err)
		}
		if !reflect.DeepEqual(documentFromPartition(loaded), documentFromPartition(partition)) {
			t.Fatalf("loaded partition differs")
		}
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Generations != len(partitions) || stats.Hits != uint64(len(partitions)) || stats.Rejected != 0 {
		t.Fatalf("store stats = %+v", stats)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewMapStore(root, testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, partition := range partitions {
		if _, found, err := reopened.LoadCurrent(expectationForPartition(t, partition, current, compilerBuild)); err != nil || !found {
			t.Fatalf("reopened LoadCurrent() = %v, %v", found, err)
		}
	}
}

func TestCurrentWorkspaceExpectationsRequireVerifiedExactCompilerInput(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	if _, err := CurrentWorkspaceExpectations(workspace, result, buildidentity.Identity{}); err == nil {
		t.Fatal("zero compiler identity succeeded")
	}
	result.Verified = false
	if _, err := CurrentWorkspaceExpectations(workspace, result, testCompilerBuild()); err == nil {
		t.Fatal("unverified compiler result succeeded")
	}
}

func TestMapStoreRejectsStaleIncompatibleCorruptAndConflictingGenerations(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	current, err := CurrentWorkspacePackages(workspace)
	if err != nil {
		t.Fatal(err)
	}
	partition := partitionWithFunction(t, partitions)
	store, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	checksum, err := store.Publish(partition)
	if err != nil {
		t.Fatal(err)
	}
	expectation := expectationForPartition(t, partition, current, compilerBuild)

	stale := expectation
	stale.Current.Files = append([]CurrentFile(nil), expectation.Current.Files...)
	stale.Current.Files[0].Digest[0] ^= 0xff
	if _, found, err := store.LoadCurrent(stale); found || !errors.Is(err, ErrStaleMap) {
		t.Fatalf("stale LoadCurrent() = %v, %v", found, err)
	}
	incompatible := expectation
	incompatible.CompilerBuild[0] ^= 0xff
	if _, found, err := store.LoadCurrent(incompatible); found || !errors.Is(err, ErrIncompatibleMap) {
		t.Fatalf("incompatible LoadCurrent() = %v, %v", found, err)
	}

	conflicting := partition
	conflicting.data.functions = cloneFunctionSummaries(partition.data.functions)
	conflicting.data.functions[0].TrapCategories = append(conflicting.data.functions[0].TrapCategories, "zz-store-conflict")
	conflicting.data.header.Counts.Relations++
	if _, err := store.Publish(conflicting); err != nil {
		t.Fatalf("publish conflicting generation: %v", err)
	}
	if _, found, err := store.LoadCurrent(expectation); found || !errors.Is(err, ErrMapConflict) {
		t.Fatalf("conflicting LoadCurrent() = %v, %v", found, err)
	}

	conflictChecksum, err := computeTransportChecksum(documentFromPartition(conflicting))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.root.Remove(filepath.Join(partition.Header().PackageInstance, mapGenerationName(conflictChecksum))); err != nil {
		t.Fatal(err)
	}
	corruptPath := filepath.Join(partition.Header().PackageInstance, mapGenerationName(checksum))
	if err := store.root.WriteFile(corruptPath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.LoadCurrent(expectation); found || !errors.Is(err, ErrStoredMapCorrupt) {
		t.Fatalf("corrupt LoadCurrent() = %v, %v", found, err)
	}
	if _, err := store.root.Lstat(corruptPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupt generation was not quarantined: %v", err)
	}
}

func TestMapStoreSweepsBoundsAndExpiredTemporaries(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	limits := testMapStoreLimits()
	limits.MaxGenerations = 1
	limits.TempMaxAge = time.Minute
	store, err := NewMapStore(t.TempDir(), limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, partition := range partitions {
		if _, err := store.Publish(partition); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Generations != 1 || stats.Evictions != 1 {
		t.Fatalf("bounded stats = %+v", stats)
	}
	remaining := partitions[len(partitions)-1]
	directory := remaining.Header().PackageInstance
	if err := store.root.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
	temporary := filepath.Join(directory, ".tmp-expired")
	if err := store.root.WriteFile(temporary, []byte("abandoned"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * limits.TempMaxAge)
	if err := os.Chtimes(filepath.Join(store.rootName, temporary), old, old); err != nil {
		t.Fatal(err)
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.root.Lstat(temporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired temporary remains: %v", err)
	}
}

func TestMapStoreRejectsInvalidLimitsAndEscapingOpenedRoot(t *testing.T) {
	t.Parallel()
	if _, err := NewMapStore("", testMapStoreLimits()); err == nil {
		t.Fatal("empty root succeeded")
	}
	for _, limits := range []MapStoreLimits{
		{},
		{MaxGenerations: 1, MaxBytes: 1, MaxMapBytes: 2, TempMaxAge: time.Second},
	} {
		if _, err := NewMapStore(t.TempDir(), limits); err == nil {
			t.Fatalf("invalid limits succeeded: %+v", limits)
		}
	}
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := NewMapStoreInRoot(root, "../escape", testMapStoreLimits()); err == nil || !strings.Contains(err.Error(), "relative") {
		t.Fatalf("escaping root error = %v", err)
	}
	outside := t.TempDir()
	workspace := root.Name()
	if err := os.Symlink(outside, filepath.Join(workspace, "maps")); err != nil {
		if os.IsPermission(err) {
			t.Skipf("directory symlink creation is unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if store, err := NewMapStoreInRoot(root, filepath.Join("maps", "atlas"), testMapStoreLimits()); err == nil {
		_ = store.Close()
		t.Fatal("external directory symlink escaped opened map boundary")
	}
	if _, err := os.Stat(filepath.Join(outside, "atlas")); !os.IsNotExist(err) {
		t.Fatalf("outside map path was touched: %v", err)
	}
}

func TestCurrentWorkspacePackagesRejectsPostValidationMutation(t *testing.T) {
	t.Parallel()
	workspace, _ := loadQuickstart(t)
	if len(workspace.Packages) == 0 || len(workspace.Packages[0].Sources) == 0 || len(workspace.Packages[0].Sources[0].Bytes) == 0 {
		t.Fatal("workspace lacks source mutation fixture")
	}
	workspace.Packages[0].Sources[0].Bytes[0] ^= 0x20
	if _, err := CurrentWorkspacePackages(workspace); err == nil || !strings.Contains(err.Error(), "source bytes changed after validation") {
		t.Fatalf("CurrentWorkspacePackages() error = %v", err)
	}
}

func TestValidateCurrentRejectsEveryStoredSourceIdentityAxis(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	current, err := CurrentWorkspacePackages(workspace)
	if err != nil {
		t.Fatal(err)
	}
	partition := partitions[0]
	base := current[partition.Header().PackageInstance]
	if err := ValidateCurrent(partition, base); err != nil {
		t.Fatalf("current partition rejected: %v", err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*CurrentPackage)
	}{
		{name: "package source", mutate: func(value *CurrentPackage) { value.Source[0] ^= 0xff }},
		{name: "file digest", mutate: func(value *CurrentPackage) { value.Files[0].Digest[0] ^= 0xff }},
		{name: "line map", mutate: func(value *CurrentPackage) { value.Files[0].LineMap[0] ^= 0xff }},
		{name: "file kind", mutate: func(value *CurrentPackage) { value.Files[0].Kind = semanticsnapshot.FileKindTest }},
		{name: "file path", mutate: func(value *CurrentPackage) { value.Files[0].Path = "zz_stale.fy" }},
		{name: "features", mutate: func(value *CurrentPackage) { value.Features = append(value.Features, "zz-stale") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneCurrentPackage(base)
			test.mutate(&candidate)
			if err := ValidateCurrent(partition, candidate); !errors.Is(err, ErrStaleMap) {
				t.Fatalf("ValidateCurrent() error = %v", err)
			}
		})
	}
}

func expectationForPartition(
	t *testing.T,
	partition Partition,
	current map[string]CurrentPackage,
	compilerBuild buildidentity.Identity,
) LoadExpectation {
	t.Helper()
	header := partition.Header()
	packageValue, found := current[header.PackageInstance]
	if !found {
		t.Fatalf("current package %q missing", header.PackageInstance)
	}
	interfaceHash, err := moduleinterface.ParseHash(header.InterfaceHash)
	if err != nil {
		t.Fatal(err)
	}
	return LoadExpectation{CompilerBuild: compilerBuild, PublicInterface: interfaceHash, Current: packageValue}
}

func partitionWithFunction(t *testing.T, values []Partition) Partition {
	t.Helper()
	for _, value := range values {
		if len(value.data.functions) != 0 {
			return value
		}
	}
	t.Fatal("no partition has a function summary")
	return Partition{}
}

func testMapStoreLimits() MapStoreLimits {
	return MapStoreLimits{MaxGenerations: 8, MaxBytes: 32 << 20, MaxMapBytes: testMapLimit, TempMaxAge: time.Hour}
}

func cloneCurrentPackage(value CurrentPackage) CurrentPackage {
	value.Features = append([]string(nil), value.Features...)
	value.Dependencies = append([]string(nil), value.Dependencies...)
	value.PackageReferences = append([]PackageReference(nil), value.PackageReferences...)
	value.Files = append([]CurrentFile(nil), value.Files...)
	return value
}
