package atlasmap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapStoreCountsEveryConflictAndCleansCorruptGenerations(t *testing.T) {
	workspace, result := loadQuickstart(t)
	build := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, build)
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
	directory := partition.Header().PackageInstance
	if err := store.root.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	// Model externally restored generations beyond the retention limit. Loads
	// must still inspect all of them, report the conflict, and clean corruption.
	const generations = mapDirectoryBatchSize + 3
	for i := range generations {
		variant := partition
		variant.data.functions = cloneFunctionSummaries(partition.data.functions)
		variant.data.functions[0].TrapCategories = append(variant.data.functions[0].TrapCategories, fmt.Sprintf("zz-conflict-%04d", i))
		variant.data.header.Counts.Relations++
		encoded, checksum, err := encodeVersionWithChecksum(variant, testMapLimit, TransportSchema, TransportEncoding, LogicalSchema)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.root.WriteFile(filepath.Join(directory, mapGenerationName(checksum)), encoded, 0600); err != nil {
			t.Fatal(err)
		}
	}
	corrupt := filepath.Join(directory, strings.Repeat("0", 64)+".fymap")
	if err := store.root.WriteFile(corrupt, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.LoadCurrent(expectationForPartition(t, partition, current, build)); found || !errors.Is(err, ErrMapConflict) {
		t.Fatalf("conflict = %t, %v", found, err)
	}
	stats, err := store.Stats()
	if err != nil || stats.Rejected != generations+1 || stats.Misses != 1 || stats.Hits != 0 {
		t.Fatalf("stats = %+v, %v", stats, err)
	}
	if _, err := store.root.Lstat(corrupt); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupt generation remains: %v", err)
	}
}
