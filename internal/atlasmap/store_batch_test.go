package atlasmap

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestMapStoreBatchCollectsOnceAndRetainsBoundsOnPartialFailure(t *testing.T) {
	workspace, checked := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, checked, testCompilerBuild())
	if err != nil || len(partitions) < 2 {
		t.Fatalf("partitions=%d, %v", len(partitions), err)
	}
	for _, fail := range []bool{false, true} {
		limits := testMapStoreLimits()
		limits.MaxGenerations = 1
		store, err := NewMapStore(t.TempDir(), limits)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { store.Close() })
		input := append([]Partition(nil), partitions...)
		if fail {
			input = append(input, Partition{})
		}
		collections := 0
		checksums, err := store.publishBatch(input, func() error { collections++; return store.Sweep() })
		if (err != nil) != fail || len(checksums) != len(partitions) || collections != 1 {
			t.Fatalf("batch checksums=%d, err=%v", len(checksums), err)
		}
		stats, err := store.Stats()
		if err != nil || stats.Generations != 1 || stats.Evictions != uint64(len(partitions)-1) {
			t.Fatalf("batch retention=%+v, %v", stats, err)
		}
		// Rejected/empty batches do not touch already retained generations.
		if _, err := store.PublishBatch([]Partition{{}}); err == nil {
			t.Fatal("accepted invalid first partition")
		}
		if got, err := store.PublishBatch(nil); err != nil || len(got) != 0 {
			t.Fatalf("empty batch=%v, %v", got, err)
		}
		after, err := store.Stats()
		if err != nil || after != stats {
			t.Fatalf("empty/invalid batch changed store: %+v, %v", after, err)
		}
	}
}

func TestMapStoreConcurrentBatchesConverge(t *testing.T) {
	workspace, checked := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, checked, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var writers sync.WaitGroup
	for range 4 {
		writers.Go(func() {
			if _, err := store.PublishBatch(partitions); err != nil {
				t.Error(err)
			}
		})
	}
	writers.Wait()
	stats, err := store.Stats()
	if err != nil || stats.Generations != len(partitions) {
		t.Fatalf("concurrent batches=%+v, %v", stats, err)
	}
}

func TestMapStoreBatchReportsCollectionFailure(t *testing.T) {
	workspace, checked := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, checked, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMapStore(t.TempDir(), testMapStoreLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	want := errors.New("collection failed")
	for _, failPublication := range []bool{false, true} {
		input := []Partition{partitions[0]}
		if failPublication {
			input = append(input, Partition{})
		}
		checksums, err := store.publishBatch(input, func() error { return want })
		if !errors.Is(err, want) || len(checksums) != 1 {
			t.Fatalf("collection failure lost: checksums=%d err=%v", len(checksums), err)
		}
		if failPublication {
			_, rejected := store.publishGeneration(Partition{}, mapPublishHooks{})
			if !strings.Contains(err.Error(), rejected.Error()) {
				t.Fatalf("publication error lost: %v", err)
			}
		}
	}
	if _, err := store.publishBatch([]Partition{{}}, func() error { t.Fatal("collected after invalid first partition"); return nil }); err == nil {
		t.Fatal("accepted invalid first partition")
	}
	var missing *MapStore
	if _, err := missing.PublishBatch(partitions); err == nil {
		t.Fatal("nil store accepted batch")
	}
	// Closed-root errors must propagate rather than reporting successful batch.
	store.Close()
	if _, err := store.PublishBatch(partitions); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("closed-root error: %v", err)
	}
}

func BenchmarkMapStoreBatchPublication(b *testing.B) {
	partition := decodeBenchmarkPartition(b, 32)
	partition.data.header.LogicalSchema = LogicalSchemaV3
	partitions := make([]Partition, 16)
	for i := range partitions {
		p := partition
		p.data.functions = cloneFunctionSummaries(partition.data.functions)
		p.data.functions[0].TrapCategories = append(p.data.functions[0].TrapCategories, fmt.Sprintf("zz-batch-%04d", i))
		p.data.header.Counts.Relations++
		partitions[i] = p
	}
	for _, batch := range []bool{false, true} {
		name := "individual"
		if batch {
			name = "batch"
		}
		b.Run(name, func(b *testing.B) {
			limits := testMapStoreLimits()
			limits.MaxGenerations = len(partitions)
			store, err := NewMapStoreV3(b.TempDir(), limits)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			b.ReportAllocs()
			for b.Loop() {
				if batch {
					if _, err := store.PublishBatch(partitions); err != nil {
						b.Fatal(err)
					}
				} else {
					for _, p := range partitions {
						if _, err := store.Publish(p); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}
