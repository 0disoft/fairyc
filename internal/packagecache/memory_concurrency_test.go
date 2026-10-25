package packagecache

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
)

func memoryCopyRecord(value byte, size int) Record {
	var record Record
	record.Key[0] = 1
	record.PublicInterface[0] = value
	record.Artifacts = []Artifact{
		{Kind: ArtifactMIR, Data: bytes.Repeat([]byte{value}, size)},
		{Kind: ArtifactObject, Data: bytes.Repeat([]byte{value}, size)},
	}
	return record
}

func TestMemoryStoreConcurrentCopiesRemainDetachedDuringReplacement(t *testing.T) {
	store, err := NewMemoryStore(Limits{MaxEntries: 1, MaxBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	first, second := memoryCopyRecord(1, 64<<10), memoryCopyRecord(2, 64<<10)
	if err := store.Put(first); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Go(func() {
		<-start
		for index := range 128 {
			if index%3 == 0 {
				store.Delete(first.Key)
			}
			record := first
			if index%2 == 0 {
				record = second
			}
			if err := store.Put(record); err != nil {
				t.Errorf("replace: %v", err)
				return
			}
			runtime.Gosched()
		}
	})
	for range 4 {
		workers.Go(func() {
			<-start
			for range 128 {
				record, found := store.Get(first.Key)
				if !found {
					continue
				}
				want := first
				if record.PublicInterface == second.PublicInterface {
					want = second
				}
				if !recordsEqual(record, want) {
					t.Error("lookup mixed generations or returned externally mutated data")
					return
				}
				record.Artifacts[0].Data[0] ^= 0xff
				record.Artifacts[1].Kind = "caller-owned"
			}
		})
	}
	close(start)
	workers.Wait()
	stats := store.Stats()
	if stats.Hits+stats.Misses != 4*128 {
		t.Fatalf("lookup count = %d, want %d", stats.Hits+stats.Misses, 4*128)
	}
}

func BenchmarkMemoryStoreSmallGet(b *testing.B) {
	for _, contended := range []bool{false, true} {
		name := "isolated"
		if contended {
			name = "with-2MiB-reader"
		}
		b.Run(name, func(b *testing.B) {
			store, err := NewMemoryStore(Limits{MaxEntries: 2, MaxBytes: 4 << 20})
			if err != nil {
				b.Fatal(err)
			}
			large, small := memoryCopyRecord(1, 1<<20), memoryCopyRecord(2, 1)
			small.Key[0] = 2
			for _, record := range []Record{large, small} {
				if err := store.Put(record); err != nil {
					b.Fatal(err)
				}
			}
			if contended {
				stop, done, started := make(chan struct{}), make(chan struct{}), make(chan struct{})
				go func() {
					defer close(done)
					store.Get(large.Key)
					close(started)
					for {
						select {
						case <-stop:
							return
						default:
							store.Get(large.Key)
						}
					}
				}()
				<-started
				defer func() { close(stop); <-done }()
			}
			for b.Loop() {
				record, found := store.Get(small.Key)
				if !found || len(record.Artifacts) != 2 || record.Artifacts[0].Data[0] != 2 {
					b.Fatal("small cache hit changed")
				}
			}
		})
	}
}
