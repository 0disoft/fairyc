package atlasmap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func mapLoadFixture(t *testing.T) (Partition, LoadExpectation) {
	t.Helper()
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
	return partition, expectationForPartition(t, partition, current, build)
}

func TestMapLoadPreservesGenerationAtReaderSizeBoundary(t *testing.T) {
	partition, expectation := mapLoadFixture(t)
	writer, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	checksum, err := writer.Publish(partition)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(partition.Header().PackageInstance, mapGenerationName(checksum))
	info, err := writer.root.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int64{-1, 0, 1} {
		t.Run(fmt.Sprintf("limit=encodedBytes%+d", delta), func(t *testing.T) {
			limits := testMapStoreLimits()
			limits.MaxMapBytes = uint64(info.Size() + delta)
			reader, err := NewMapStore(filepath.Dir(writer.rootName), limits)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			var load func(LoadExpectation) (Partition, bool, error) = reader.LoadCurrent
			_, found, err := load(expectation)
			if _, statErr := writer.root.Lstat(path); statErr != nil {
				t.Fatalf("reader destroyed sibling generation: %v", statErr)
			}
			if delta < 0 {
				if found || !errors.Is(err, ErrMapTooLarge) {
					t.Fatalf("reader bound: found=%v err=%v", found, err)
				}
				if reader.rejected != 0 {
					t.Fatal("reader limit counted as corruption")
				}
			} else if !found || err != nil {
				t.Fatalf("admissible map: found=%v err=%v", found, err)
			}
		})
	}
}

func TestMapLoadInjectedIOAndCancellation(t *testing.T) {
	partition, expectation := mapLoadFixture(t)
	store, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	name := strings.Repeat("a", 64) + ".fymap"
	entries, err := fs.ReadDir(fstest.MapFS{name: &fstest.MapFile{Data: []byte("unused")}}, ".")
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("injected read or close failure")
	for _, scenario := range []string{"success", "permission", "read-or-close", "size", "disappeared", "directory", "pre-cancel", "cancel-after-read", "cancel-after-visit", "corrupt", "cleanup-failure", "cancel-corrupt"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			visits, reads, removes := 0, 0, 0
			options := MapLoadOptions{Context: ctx,
				VisitDirectory: func(ctx context.Context, directory string, visit func(os.DirEntry) error) error {
					visits++
					if directory != expectation.Current.Instance.String() {
						t.Fatal("wrong directory")
					}
					if scenario == "directory" {
						return failure
					}
					err := visit(entries[0])
					if scenario == "cancel-after-visit" {
						cancel()
					}
					return err
				},
				ReadGeneration: func(ctx context.Context, path, base, directory string) (Partition, error) {
					reads++
					if base != name || path != filepath.Join(directory, name) {
						t.Fatal("wrong read path")
					}
					switch scenario {
					case "permission":
						return Partition{}, os.ErrPermission
					case "read-or-close":
						return Partition{}, failure
					case "size":
						return Partition{}, ErrMapTooLarge
					case "disappeared":
						return Partition{}, os.ErrNotExist
					case "corrupt", "cleanup-failure":
						return Partition{}, ErrStoredMapCorrupt
					case "cancel-corrupt":
						cancel()
						return Partition{}, ErrStoredMapCorrupt
					case "cancel-after-read":
						cancel()
					}
					return partition, nil
				},
				Remove: func(ctx context.Context, path string) error {
					removes++
					if scenario == "cleanup-failure" {
						return os.ErrPermission
					}
					return nil
				},
			}
			if scenario == "pre-cancel" {
				cancel()
			}
			_, found, err := store.LoadCurrentWithOptions(expectation, options)
			var want error
			switch scenario {
			case "permission":
				want = os.ErrPermission
			case "read-or-close", "directory":
				want = failure
			case "size":
				want = ErrMapTooLarge
			case "corrupt", "cleanup-failure":
				want = ErrStoredMapCorrupt
			case "pre-cancel", "cancel-after-read", "cancel-after-visit", "cancel-corrupt":
				want = context.Canceled
			}
			if !errors.Is(err, want) || found != (scenario == "success") {
				t.Fatalf("found=%v err=%v want=%v", found, err, want)
			}
			wantRemoves := 0
			if scenario == "corrupt" || scenario == "cleanup-failure" {
				wantRemoves = 1
			}
			if removes != wantRemoves {
				t.Fatalf("unsafe removals: got %d want %d", removes, wantRemoves)
			}
			if scenario == "pre-cancel" && (visits != 0 || reads != 0) {
				t.Fatal("canceled load invoked I/O")
			}
		})
	}
	if _, _, err := store.LoadCurrentWithOptions(expectation, MapLoadOptions{}, MapLoadOptions{}); err == nil {
		t.Fatal("ambiguous options admitted")
	}
}
