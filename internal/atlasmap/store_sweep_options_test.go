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
	"time"
)

func sweepPublishedMap(t *testing.T, partition Partition) (*MapStore, string, uint64) {
	t.Helper()
	store, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	checksum, err := store.Publish(partition)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(partition.Header().PackageInstance, mapGenerationName(checksum))
	info, err := store.root.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	return store, name, uint64(info.Size())
}

func TestMapSweepPreservesSiblingAtReaderBoundary(t *testing.T) {
	partition, expectation := mapLoadFixture(t)
	for _, delta := range []int64{-1, 0, 1} {
		t.Run(fmt.Sprintf("reader-limit%+d", delta), func(t *testing.T) {
			writer, name, size := sweepPublishedMap(t, partition)
			limits := testMapStoreLimits()
			limits.MaxMapBytes = uint64(int64(size) + delta)
			reader, err := NewMapStore(filepath.Dir(writer.rootName), limits)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			var sweep func() error = reader.Sweep
			if err := sweep(); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.root.Lstat(name); err != nil {
				t.Fatalf("reader limit deleted sibling publication: %v", err)
			}
			stats, err := reader.Stats()
			if err != nil || stats.Generations != 1 || stats.Bytes != size || stats.Rejected != 0 || stats.Evictions != 0 {
				t.Fatalf("incorrect retained accounting: %+v %v", stats, err)
			}
			if _, found, err := writer.LoadCurrent(expectation); err != nil || !found {
				t.Fatalf("writer lost valid map: found=%v err=%v", found, err)
			}
		})
	}
}

func TestMapSweepTotalBytesBoundaries(t *testing.T) {
	partition, _ := mapLoadFixture(t)
	for _, delta := range []int64{-1, 0, 1} {
		t.Run(fmt.Sprintf("total-limit%+d", delta), func(t *testing.T) {
			writer, name, size := sweepPublishedMap(t, partition)
			limits := testMapStoreLimits()
			limits.MaxMapBytes = size / 2
			limits.MaxBytes = uint64(int64(size) + delta)
			reader, err := NewMapStore(filepath.Dir(writer.rootName), limits)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.SweepWithOptions(MapSweepOptions{}); err != nil {
				t.Fatal(err)
			}
			_, statErr := writer.root.Lstat(name)
			if delta < 0 && !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("over-budget map retained: %v", statErr)
			}
			if delta >= 0 && statErr != nil {
				t.Fatalf("within-budget map removed: %v", statErr)
			}
			stats, err := reader.Stats()
			if err != nil {
				t.Fatal(err)
			}
			if delta < 0 {
				if stats.Generations != 0 || stats.Bytes != 0 || stats.Evictions != 1 {
					t.Fatalf("bad eviction accounting: %+v", stats)
				}
			} else if stats.Generations != 1 || stats.Bytes != size || stats.Evictions != 0 {
				t.Fatalf("bad preservation accounting: %+v", stats)
			}
			if stats.Rejected != 0 {
				t.Fatal("reader limit classified as corruption")
			}
		})
	}
}

func TestMapSweepExpiryClockBoundaries(t *testing.T) {
	partition, _ := mapLoadFixture(t)
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		t.Run(delta.String(), func(t *testing.T) {
			store, generation, _ := sweepPublishedMap(t, partition)
			name := filepath.Join(filepath.Dir(generation), ".tmp-pending")
			if err := store.root.WriteFile(name, []byte("partial"), 0600); err != nil {
				t.Fatal(err)
			}
			info, err := store.root.Lstat(name)
			if err != nil {
				t.Fatal(err)
			}
			clockCalls := 0
			if err := store.SweepWithOptions(MapSweepOptions{Now: func() time.Time { clockCalls++; return info.ModTime().Add(store.limits.TempMaxAge + delta) }}); err != nil {
				t.Fatal(err)
			}
			if clockCalls != 1 {
				t.Fatalf("clock sampled %d times", clockCalls)
			}
			_, err = store.root.Lstat(name)
			if delta < 0 && err != nil {
				t.Fatalf("unexpired temporary removed: %v", err)
			}
			if delta >= 0 && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expired temporary retained: %v", err)
			}
		})
	}
}

func TestMapSweepInjectedFailuresAndCancellation(t *testing.T) {
	partition, _ := mapLoadFixture(t)
	packageName := partition.Header().PackageInstance
	oldName, newName := strings.Repeat("a", 64)+".fymap", strings.Repeat("b", 64)+".fymap"
	instant := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	tree := fstest.MapFS{
		packageName + "/" + oldName: &fstest.MapFile{Data: make([]byte, 129), ModTime: instant},
		packageName + "/" + newName: &fstest.MapFile{Data: make([]byte, 129), ModTime: instant.Add(time.Second)},
	}
	failure := errors.New("injected filesystem failure")
	for _, scenario := range []string{"success", "directory", "info", "nil-info", "pre-cancel", "cancel-info", "cancel-after-scan", "remove", "missing-remove", "temporary-remove", "empty-remove"} {
		t.Run(scenario, func(t *testing.T) {
			caseTree := tree
			removeName := oldName
			if scenario == "temporary-remove" || scenario == "empty-remove" {
				caseTree = fstest.MapFS{}
				for name, file := range tree {
					caseTree[name] = file
				}
				removeName = ".tmp-expired"
				if scenario == "empty-remove" {
					removeName = strings.Repeat("0", 64) + ".fymap"
				}
				caseTree[packageName+"/"+removeName] = &fstest.MapFile{ModTime: instant.Add(-time.Hour)}
			}
			limits := testMapStoreLimits()
			limits.MaxBytes = 257
			limits.MaxMapBytes = 64
			store, err := NewMapStore(t.TempDir(), limits)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			clocks, visits, infos, removes := 0, 0, 0, 0
			options := MapSweepOptions{Context: ctx, Now: func() time.Time { clocks++; return instant },
				VisitDirectory: func(ctx context.Context, name string, visit func(os.DirEntry) error) error {
					visits++
					if scenario == "directory" {
						return failure
					}
					entries, err := fs.ReadDir(caseTree, name)
					if err != nil {
						return err
					}
					for _, entry := range entries {
						if err := visit(entry); err != nil {
							return err
						}
					}
					if scenario == "cancel-after-scan" && name == "." {
						cancel()
					}
					return nil
				},
				Info: func(ctx context.Context, entry os.DirEntry) (os.FileInfo, error) {
					infos++
					if scenario == "info" {
						return nil, failure
					}
					if scenario == "nil-info" {
						return nil, nil
					}
					if scenario == "cancel-info" {
						cancel()
					}
					return entry.Info()
				},
				Remove: func(ctx context.Context, name string) error {
					removes++
					if name != filepath.Join(packageName, removeName) {
						t.Fatalf("incorrect oldest eviction: %s", name)
					}
					if scenario == "remove" || scenario == "temporary-remove" || scenario == "empty-remove" {
						return failure
					}
					if scenario == "missing-remove" {
						return os.ErrNotExist
					}
					return nil
				},
			}
			if scenario == "pre-cancel" {
				cancel()
			}
			err = store.SweepWithOptions(options)
			var want error
			switch scenario {
			case "directory", "info", "remove", "temporary-remove", "empty-remove":
				want = failure
			case "pre-cancel", "cancel-info", "cancel-after-scan":
				want = context.Canceled
			}
			if scenario == "nil-info" {
				if err == nil {
					t.Fatal("missing metadata admitted")
				}
			} else if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			wantRemoves := 0
			if scenario == "success" || scenario == "remove" || scenario == "missing-remove" || scenario == "temporary-remove" || scenario == "empty-remove" {
				wantRemoves = 1
			}
			if removes != wantRemoves {
				t.Fatalf("remove calls=%d want=%d", removes, wantRemoves)
			}
			if scenario == "pre-cancel" && (clocks != 0 || visits != 0 || infos != 0) {
				t.Fatal("pre-canceled sweep invoked dependencies")
			}
			if scenario == "success" || scenario == "missing-remove" {
				if store.evicted != 1 {
					t.Fatal("eviction not counted")
				}
			} else if store.evicted != 0 {
				t.Fatal("failed eviction counted as success")
			}
		})
	}
}
