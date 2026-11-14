package packagecache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

func sweepFixture(t *testing.T) (*DiskStore, string, os.FileInfo) {
	t.Helper()
	store, err := NewDiskStore(t.TempDir(), DiskLimits{MaxEntries: 2, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	name := filepath.Join("v1", strings.Repeat("a", 64), ".tmp-pending")
	if err := store.retentionRoot.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.retentionRoot.WriteFile(name, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := store.retentionRoot.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	return store, name, info
}

func TestSweepClockExpiryBoundariesAcrossNamespaces(t *testing.T) {
	for _, delta := range []time.Duration{-2 * time.Minute, -time.Nanosecond, 0, time.Nanosecond} {
		t.Run(delta.String(), func(t *testing.T) {
			store, name, info := sweepFixture(t)
			when := info.ModTime()
			paths := []string{name}
			for _, namespace := range []string{"generic-v0", "storage-transfer-v0"} {
				path := filepath.Join(namespace, name)
				if err := store.retentionRoot.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := store.retentionRoot.WriteFile(path, []byte("partial"), 0600); err != nil {
					t.Fatal(err)
				}
				absolute := filepath.Join(store.retentionRoot.Name(), path)
				if err := os.Chtimes(absolute, when, when); err != nil {
					t.Fatal(err)
				}
				paths = append(paths, path)
			}
			calls := 0
			now := when.Add(store.limits.TempMaxAge + delta)
			if err := store.SweepWithOptions(SweepOptions{Now: func() time.Time { calls++; return now }}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("clock sampled %d times", calls)
			}
			for _, path := range paths {
				_, err := store.retentionRoot.Lstat(path)
				if delta < 0 && err != nil {
					t.Fatalf("pre-expiry file removed: %s %v", path, err)
				}
				if delta >= 0 && !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("expired file survived: %s %v", path, err)
				}
			}
		})
	}
}

func TestSweepPreservesChangedTemporaryAfterScan(t *testing.T) {
	for _, change := range []string{"refreshed", "replaced", "resized", "directory", "disappeared"} {
		t.Run(change, func(t *testing.T) {
			store, name, original := sweepFixture(t)
			now := original.ModTime().Add(2 * store.limits.TempMaxAge)
			observations, removals := 0, 0
			options := SweepOptions{Now: func() time.Time { return now }, Lstat: func(path string) (os.FileInfo, error) {
				info, err := store.retentionRoot.Lstat(path)
				if path == name && err == nil {
					observations++
					if observations == 1 {
						absolute := filepath.Join(store.retentionRoot.Name(), name)
						switch change {
						case "refreshed":
							err = os.Chtimes(absolute, now, now)
						case "replaced":
							err = store.retentionRoot.Rename(name, name+".original")
							if err == nil {
								err = store.retentionRoot.WriteFile(name, []byte("renewed"), 0600)
							}
							if err == nil {
								err = os.Chtimes(absolute, original.ModTime(), original.ModTime())
							}
						case "resized":
							err = store.retentionRoot.WriteFile(name, []byte("still writing"), 0600)
							if err == nil {
								err = os.Chtimes(absolute, original.ModTime(), original.ModTime())
							}
						case "disappeared":
							err = store.retentionRoot.Remove(name)
						case "directory":
							err = store.retentionRoot.Remove(name)
							if err == nil {
								err = store.retentionRoot.Mkdir(name, 0700)
							}
						}
					}
				}
				return info, err
			}, Remove: func(path string) error {
				if path == name {
					removals++
				}
				return store.retentionRoot.Remove(path)
			}}
			if err := store.SweepWithOptions(options); err != nil {
				t.Fatal(err)
			}
			if removals != 0 {
				t.Fatalf("changed candidate was deleted %d times", removals)
			}
			if change == "directory" {
				if info, err := store.retentionRoot.Lstat(name); err != nil || !info.IsDir() {
					t.Fatalf("replacement directory lost: %v", err)
				}
			} else if change != "disappeared" {
				data, err := store.retentionRoot.ReadFile(name)
				if err != nil || len(data) == 0 {
					t.Fatalf("writer temporary lost: %s %v", change, err)
				}
			}
		})
	}
}

func TestSweepCancellationAndIOFailures(t *testing.T) {
	for _, failure := range []string{"pre-cancel", "scan", "stat", "recheck", "cancel-recheck", "remove"} {
		t.Run(failure, func(t *testing.T) {
			store, name, info := sweepFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			injected := errors.New("injected filesystem failure")
			observations, removals, clockCalls := 0, 0, 0
			options := SweepOptions{Context: ctx, Now: func() time.Time { clockCalls++; return info.ModTime().Add(2 * store.limits.TempMaxAge) }, Lstat: func(path string) (os.FileInfo, error) {
				if path == name {
					observations++
					if failure == "stat" || failure == "recheck" && observations == 2 {
						return nil, injected
					}
					if failure == "cancel-recheck" && observations == 2 {
						cancel()
					}
				}
				return store.retentionRoot.Lstat(path)
			}, Remove: func(path string) error {
				removals++
				if path == name && failure == "remove" {
					return injected
				}
				return store.retentionRoot.Remove(path)
			}}
			if failure == "pre-cancel" {
				cancel()
			}
			if failure == "scan" {
				options.ReadDirectory = func(string, *directoryscan.Budget) ([]os.DirEntry, error) { return nil, injected }
			}
			err := store.SweepWithOptions(options)
			want := injected
			if failure == "pre-cancel" || failure == "cancel-recheck" {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("%s: %v", failure, err)
			}
			if failure != "remove" && removals != 0 {
				t.Fatal("mutation before admission/cancellation completed")
			}
			if failure == "pre-cancel" && (clockCalls != 0 || observations != 0) {
				t.Fatal("pre-canceled sweep touched dependencies")
			}
			if _, err := store.retentionRoot.Lstat(name); err != nil {
				t.Fatalf("failure erased temporary: %v", err)
			}
		})
	}
}

func TestSweepOptionalDefaultsAndPureCleanupIO(t *testing.T) {
	store, name, info := sweepFixture(t)
	// Existing zero-argument method-value shape is preserved.
	var sweep func() error = store.Sweep
	if err := sweep(); err != nil {
		t.Fatal(err)
	}
	if err := store.SweepWithOptions(SweepOptions{}, SweepOptions{}); err == nil {
		t.Fatal("ambiguous options admitted")
	}
	stats, removes := 0, 0
	policy, err := store.sweepPolicy([]SweepOptions{{Now: func() time.Time { return info.ModTime().Add(store.limits.TempMaxAge) }, Lstat: func(string) (os.FileInfo, error) { stats++; return info, nil }, Remove: func(string) error { removes++; return nil }}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.removeExpiredTemporary(diskTemporary{path: name, identity: info}, policy); err != nil {
		t.Fatal(err)
	}
	if stats != 1 || removes != 1 {
		t.Fatalf("pure I/O seam: stat=%d remove=%d", stats, removes)
	}
	if _, err := store.retentionRoot.Lstat(name); err != nil {
		t.Fatal("fake cleanup unexpectedly touched disk")
	}
}
