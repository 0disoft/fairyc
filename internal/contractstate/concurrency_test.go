package contractstate

import (
	"sync"
	"testing"
)

func TestTransactionSupportsConcurrentDistinctMapWrites(t *testing.T) {
	t.Parallel()
	builder, err := NewLayoutBuilder("example.concurrent")
	if err != nil {
		t.Fatal(err)
	}
	values, err := DeclareMap(builder, "values", U32, U64)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	limits := Limits{MaxWrites: 64, MaxWriteBytes: 8192, MaxValueBytes: 64, MaxMapKeyBytes: 64}
	transaction, err := store.Begin(layout, limits)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 32
	errorsByWorker := make([]error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		worker := worker
		wait.Add(1)
		go func() {
			defer wait.Done()
			errorsByWorker[worker] = values.Set(transaction, uint32(worker), uint64(worker*worker))
		}()
	}
	wait.Wait()
	for worker, err := range errorsByWorker {
		if err != nil {
			t.Fatalf("worker %d Set() error = %v", worker, err)
		}
	}
	if _, err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := View(store, layout, limits, func(transaction *Transaction) error {
		for worker := 0; worker < workers; worker++ {
			value, found, err := values.Get(transaction, uint32(worker))
			if err != nil {
				return err
			}
			if !found || value != uint64(worker*worker) {
				t.Fatalf("value[%d] = %d, %v", worker, value, found)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
