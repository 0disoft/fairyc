package packagecache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

func TestDiskGetReaderLimitPreservesUnverifiedGeneration(t *testing.T) {
	t.Parallel()
	for _, sibling := range []bool{false, true} {
		for _, delta := range []int{-1, 0, 1} {
			t.Run(fmt.Sprintf("sibling=%t/limit-delta=%d", sibling, delta), func(t *testing.T) {
				limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
				root := t.TempDir()
				store, err := NewDiskStore(root, limits)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = store.Close() })
				record := testRecord(t, Key{1}, strings.Repeat("large", 64))
				if err := store.Put(record); err != nil {
					t.Fatal(err)
				}
				encoded, err := encodeDiskRecord(record, limits.MaxRecordBytes)
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256(encoded)
				filename := filepath.Join(store.keyDirectory(record.Key), hex.EncodeToString(digest[:])+".json")
				if sibling {
					if err := store.Put(testRecord(t, record.Key, "small")); err != nil {
						t.Fatal(err)
					}
				}
				limits.MaxRecordBytes = uint64(len(encoded) + delta)
				reader, err := NewDiskStore(root, limits)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = reader.Close() })
				got, hit := reader.Get(record.Key)
				wantHit := !sibling && delta >= 0
				if hit != wantHit || hit && !recordsEqual(got, record) {
					t.Errorf("Get: hit=%t want=%t; unverified sibling must never authorize a hit", hit, wantHit)
				}
				if delta < 0 || !sibling {
					data, err := os.ReadFile(filename)
					if err != nil || !bytes.Equal(data, encoded) {
						t.Errorf("reader limit destroyed an immutable generation: bytes=%d error=%v", len(data), err)
					}
				}
			})
		}
	}
}

func diskGetTestStore(t *testing.T, payloads ...string) (*DiskStore, Key, []os.DirEntry) {
	t.Helper()
	limits := DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute}
	store, err := NewDiskStore(t.TempDir(), limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	key := Key{1}
	if len(payloads) == 0 {
		payloads = []string{"first", "second"}
	}
	for _, payload := range payloads {
		if err := store.Put(testRecord(t, key, payload)); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(store.keyDirectory(key))
	if err != nil {
		t.Fatal(err)
	}
	return store, key, entries
}

func TestDiskGetReadFailuresNeverDeleteOrHideConflicts(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{os.ErrPermission, ErrRecordTooLarge, io.ErrUnexpectedEOF, io.ErrClosedPipe, context.Canceled, context.DeadlineExceeded} {
		for _, failAt := range []int{1, 2} {
			t.Run(fmt.Sprintf("%v/read=%d", failure, failAt), func(t *testing.T) {
				store, key, _ := diskGetTestStore(t)
				reads, removes := 0, 0
				_, hit := store.GetWithOptions(key, DiskGetOptions{
					ReadGeneration: func(context.Context, string, string, Key) (Record, error) {
						reads++
						if reads == failAt {
							return Record{}, fmt.Errorf("injected generation failure: %w", failure)
						}
						return testRecord(t, key, "verified"), nil
					},
					Remove: func(string) error { removes++; return nil },
				})
				if hit || reads != failAt || removes != 0 {
					t.Fatalf("failed observation: hit=%t reads=%d removes=%d", hit, reads, removes)
				}
				stats, err := store.Stats()
				if err != nil || stats.Entries != 2 || stats.Hits != 0 || stats.Misses != 1 || stats.Rejected != 0 {
					t.Fatalf("unread generations changed: stats=%+v error=%v", stats, err)
				}
			})
		}
	}
}

func TestDiskGetCancellationStopsObservationAndCleanup(t *testing.T) {
	t.Parallel()
	for _, at := range []string{"before-directory", "after-directory", "after-valid-read", "after-corrupt-read", "during-conflict-cleanup"} {
		t.Run(at, func(t *testing.T) {
			store, key, entries := diskGetTestStore(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			lists, reads, removes := 0, 0, 0
			policy := DiskGetOptions{
				Context: ctx,
				ReadDirectory: func(_ string, budget *directoryscan.Budget) ([]os.DirEntry, error) {
					lists++
					if budget.Context != ctx {
						t.Fatal("lookup cancellation was not passed to the directory budget")
					}
					if at == "after-directory" {
						cancel()
					}
					return entries, nil
				},
				ReadGeneration: func(context.Context, string, string, Key) (Record, error) {
					reads++
					if at == "after-valid-read" || at == "after-corrupt-read" {
						cancel()
					}
					if at == "after-corrupt-read" {
						return Record{}, errDiskGenerationCorrupt
					}
					return testRecord(t, key, fmt.Sprint(reads)), nil
				},
				Remove: func(string) error { removes++; cancel(); return nil },
			}
			if at == "before-directory" {
				cancel()
			}
			_, hit := store.GetWithOptions(key, policy)
			wantLists, wantReads, wantRemoves := 1, 0, 0
			switch at {
			case "before-directory":
				wantLists = 0
			case "after-valid-read", "after-corrupt-read":
				wantReads = 1
			case "during-conflict-cleanup":
				wantReads, wantRemoves = 2, 1
			}
			if hit || ctx.Err() == nil || lists != wantLists || reads != wantReads || removes != wantRemoves {
				t.Fatalf("cancellation at %s: hit=%t lists=%d reads=%d removes=%d", at, hit, lists, reads, removes)
			}
		})
	}
}

func TestDiskGetIncompleteDirectoryAndInvalidOptionsDoNoGenerationIO(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{os.ErrPermission, directoryscan.ErrLimit} {
		t.Run(failure.Error(), func(t *testing.T) {
			store, key, entries := diskGetTestStore(t)
			policy := DiskGetOptions{
				ReadDirectory: func(string, *directoryscan.Budget) ([]os.DirEntry, error) { return entries[:1], failure },
				ReadGeneration: func(context.Context, string, string, Key) (Record, error) {
					t.Fatal("incomplete directory caused a generation read")
					return Record{}, nil
				},
				Remove: func(string) error { t.Fatal("incomplete directory caused deletion"); return nil },
			}
			if _, hit := store.GetWithOptions(key, policy); hit {
				t.Fatal("partial directory produced a hit")
			}
			policy.ReadDirectory = func(string, *directoryscan.Budget) ([]os.DirEntry, error) {
				t.Fatal("invalid options caused directory I/O")
				return nil, nil
			}
			if _, hit := store.GetWithOptions(key, policy, policy); hit {
				t.Fatal("invalid options produced a hit")
			}
		})
	}
}

func TestDiskGetMissingSiblingStillAllowsVerifiedHit(t *testing.T) {
	t.Parallel()
	store, key, _ := diskGetTestStore(t)
	reads, removes := 0, 0
	want := testRecord(t, key, "remaining")
	got, hit := store.GetWithOptions(key, DiskGetOptions{
		ReadGeneration: func(context.Context, string, string, Key) (Record, error) {
			reads++
			if reads == 1 {
				return Record{}, os.ErrNotExist
			}
			return want, nil
		},
		Remove: func(string) error { removes++; return nil },
	})
	if !hit || !recordsEqual(got, want) || reads != 2 || removes != 0 {
		t.Fatalf("concurrent removal: hit=%t reads=%d removes=%d", hit, reads, removes)
	}
}

func TestDiskGetRemovesOnlyVerifiedContentCorruption(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"digest", "empty", "malformed", "noncanonical", "wrong-key"} {
		t.Run(kind, func(t *testing.T) {
			store, key, _ := diskGetTestStore(t, "valid")
			record := testRecord(t, key, "invalid sibling")
			if kind == "wrong-key" {
				record.Key = Key{2}
			}
			data, err := encodeDiskRecord(record, store.limits.MaxRecordBytes)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "empty":
				data = nil
			case "malformed":
				data = []byte("{")
			case "noncanonical":
				data = append(data, '\n')
			}
			digest := sha256.Sum256(data)
			name := hex.EncodeToString(digest[:]) + ".json"
			if kind == "digest" {
				name = strings.Repeat("f", 64) + ".json"
			}
			path := filepath.Join(store.keyDirectoryName(key), name)
			if err := store.root.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.readGeneration(path, name, key, int64(len(data))); !errors.Is(err, errDiskGenerationCorrupt) {
				t.Fatalf("content corruption was not classified: %v", err)
			}
			got, hit := store.Get(key)
			if !hit || !recordsEqual(got, testRecord(t, key, "valid")) {
				t.Fatal("verified corruption prevented a valid sibling hit")
			}
			if _, err := store.root.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("verified corrupt generation was retained: %v", err)
			}
		})
	}
}
