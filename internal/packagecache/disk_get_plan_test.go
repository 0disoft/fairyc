package packagecache

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

type diskLookupEntry struct {
	name      string
	directory bool
	nameCalls *int
}

func (entry diskLookupEntry) Name() string {
	if entry.nameCalls != nil {
		*entry.nameCalls += 1
	}
	return entry.name
}
func (entry diskLookupEntry) IsDir() bool { return entry.directory }
func (entry diskLookupEntry) Type() os.FileMode {
	if entry.directory {
		return os.ModeDir
	}
	return 0
}
func (entry diskLookupEntry) Info() (os.FileInfo, error) { return nil, os.ErrInvalid }

func TestDiskGetPreservesDirectorySnapshotAndIOOrder(t *testing.T) {
	first, second := strings.Repeat("1", 64)+".json", strings.Repeat("2", 64)+".json"
	for _, mode := range []string{"hit", "conflict", "unreadable", "corrupt", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			store, err := NewDiskStore(t.TempDir(), DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			entries := []os.DirEntry{diskLookupEntry{name: second}, diskLookupEntry{name: "unowned.txt"}, diskLookupEntry{name: first}, diskLookupEntry{name: strings.Repeat("f", 64) + ".json", directory: true}}
			if mode == "hit" {
				entries = entries[1:]
			}
			before := slices.Clone(entries)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			key := Key{1}
			want := testRecord(t, key, "verified")
			got, hit := store.GetWithOptions(key, DiskGetOptions{
				Context:       ctx,
				ReadDirectory: func(string, *directoryscan.Budget) ([]os.DirEntry, error) { return entries, nil },
				ReadGeneration: func(_ context.Context, _, name string, _ Key) (Record, error) {
					events = append(events, "read:"+name)
					if mode == "canceled" {
						cancel()
						return Record{}, errDiskGenerationCorrupt
					}
					if mode == "unreadable" && name == second {
						return Record{}, os.ErrPermission
					}
					if mode == "corrupt" && name == first {
						return Record{}, errDiskGenerationCorrupt
					}
					return want, nil
				},
				Remove: func(path string) error { events = append(events, "remove:"+filepath.Base(path)); return nil },
			})
			if !slices.Equal(entries, before) {
				t.Error("lookup reordered the caller-owned directory snapshot")
			}
			wantEvents := []string{"read:" + first}
			wantHit, rejected := false, uint64(0)
			switch mode {
			case "hit":
				wantHit = true
			case "conflict":
				wantEvents = append(wantEvents, "read:"+second, "remove:"+first, "remove:"+second)
				rejected = 2
			case "unreadable":
				wantEvents = append(wantEvents, "read:"+second)
			case "corrupt":
				wantEvents = append(wantEvents, "remove:"+first, "read:"+second)
				wantHit, rejected = true, 1
			}
			if hit != wantHit || hit && !recordsEqual(got, want) || !slices.Equal(events, wantEvents) || store.rejected != rejected {
				t.Fatalf("lookup: hit=%t events=%v rejected=%d, want %t %v %d", hit, events, store.rejected, wantHit, wantEvents, rejected)
			}
		})
	}
}

func BenchmarkDiskGetDirectoryCandidates(b *testing.B) {
	for _, shuffled := range []bool{false, true} {
		b.Run(fmt.Sprintf("noise_4096/shuffled=%t", shuffled), func(b *testing.B) {
			store, err := NewDiskStore(b.TempDir(), DiskLimits{MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Minute})
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = store.Close() })
			nameCalls := 0
			entries := []os.DirEntry{diskLookupEntry{name: strings.Repeat("1", 64) + ".json", nameCalls: &nameCalls}}
			for index := range 4096 {
				entries = append(entries, diskLookupEntry{name: fmt.Sprintf("unowned-%06d.tmp", index), nameCalls: &nameCalls})
			}
			if shuffled {
				rand.New(rand.NewPCG(77, 17)).Shuffle(len(entries), func(i, j int) { entries[i], entries[j] = entries[j], entries[i] })
			}
			key := Key{1}
			policy := DiskGetOptions{
				// Give each lookup identical data, including the caller's copy cost.
				ReadDirectory:  func(string, *directoryscan.Budget) ([]os.DirEntry, error) { return slices.Clone(entries), nil },
				ReadGeneration: func(context.Context, string, string, Key) (Record, error) { return Record{Key: key}, nil },
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, hit := store.GetWithOptions(key, policy); !hit {
					b.Fatal("missing verified generation")
				}
			}
			b.ReportMetric(float64(nameCalls)/float64(b.N), "names/op")
		})
	}
}
