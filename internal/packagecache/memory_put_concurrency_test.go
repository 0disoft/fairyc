package packagecache

import (
	"sync"
	"testing"
)

func TestMemoryStoreConcurrentWritersPublishCompleteRecords(t *testing.T) {
	store, err := NewMemoryStore(Limits{MaxEntries: 1, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	records := []Record{memoryCopyRecord(1, 64<<10), memoryCopyRecord(2, 64<<10)}
	if err := store.Put(records[0]); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, record := range records {
		workers.Go(func() {
			<-start
			for range 64 {
				if err := store.Put(record); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	workers.Go(func() {
		<-start
		for range 256 {
			record, found := store.Get(records[0].Key)
			if found && !recordsEqual(record, records[0]) && !recordsEqual(record, records[1]) {
				t.Error("reader observed a partially published record")
				return
			}
		}
	})
	close(start)
	workers.Wait()
	stats := store.Stats()
	wantSize, err := validateRecord(records[0])
	if err != nil || stats.Entries != 1 || stats.Bytes != wantSize || stats.Evictions != 0 {
		t.Fatalf("replacement changed store accounting: %+v, %v", stats, err)
	}
}

func BenchmarkMemoryStoreSmallGetWithLargeWriter(b *testing.B) {
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
	stop, done, started := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		if err := store.Put(large); err != nil {
			b.Error(err)
		}
		close(started)
		for {
			select {
			case <-stop:
				return
			default:
				if err := store.Put(large); err != nil {
					b.Error(err)
					return
				}
			}
		}
	}()
	<-started
	defer func() { close(stop); <-done }()
	for b.Loop() {
		record, found := store.Get(small.Key)
		if !found || !recordsEqual(record, small) {
			b.Fatal("small cache hit changed")
		}
	}
}
