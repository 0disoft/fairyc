package contractstate

import (
	"errors"
	"fmt"
	"testing"
)

type preparedConcurrentCommit struct {
	name        string
	marker      uint64
	transaction *Transaction
}

type concurrentCommitResult struct {
	name    string
	marker  uint64
	receipt CommitReceipt
	err     error
}

func TestConcurrentCellTransactionsPublishExactlyOneCompleteOverlay(t *testing.T) {
	t.Parallel()

	builder, err := NewLayoutBuilder("example.concurrent-cells")
	if err != nil {
		t.Fatal(err)
	}
	left, err := DeclareCell(builder, "left", U64)
	if err != nil {
		t.Fatal(err)
	}
	right, err := DeclareCell(builder, "right", U64)
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

	first := beginCellPairTransaction(t, store, layout, left, right, 11)
	second := beginCellPairTransaction(t, store, layout, left, right, 22)
	winner := assertOneConcurrentCommit(t, first, second)

	if store.Version() != 1 {
		t.Fatalf("store version = %d, want 1", store.Version())
	}
	if err := View(store, layout, concurrentTestLimits(), func(transaction *Transaction) error {
		leftValue, leftFound, err := left.Get(transaction)
		if err != nil {
			return err
		}
		rightValue, rightFound, err := right.Get(transaction)
		if err != nil {
			return err
		}
		if !leftFound || !rightFound {
			return fmt.Errorf("committed pair is incomplete: left=%v right=%v", leftFound, rightFound)
		}
		if leftValue != winner || rightValue != winner {
			return fmt.Errorf("committed pair is mixed: left=%d right=%d winner=%d", leftValue, rightValue, winner)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentMapTransactionsKeepStoreWideConflictRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		slug       string
		firstKeys  []uint32
		secondKeys []uint32
	}{
		{name: "same keys", slug: "same-keys", firstKeys: []uint32{1, 2}, secondKeys: []uint32{1, 2}},
		{name: "distinct keys", slug: "distinct-keys", firstKeys: []uint32{1, 2}, secondKeys: []uint32{3, 4}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			builder, err := NewLayoutBuilder("example.concurrent-map-" + test.slug)
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

			first := beginMapTransaction(t, store, layout, values, "first", 11, test.firstKeys)
			second := beginMapTransaction(t, store, layout, values, "second", 22, test.secondKeys)
			winner := assertOneConcurrentCommit(t, first, second)

			winnerKeys := test.firstKeys
			loserKeys := test.secondKeys
			if winner == 22 {
				winnerKeys, loserKeys = loserKeys, winnerKeys
			}
			if err := View(store, layout, concurrentTestLimits(), func(transaction *Transaction) error {
				for _, key := range winnerKeys {
					value, found, err := values.Get(transaction, key)
					if err != nil {
						return err
					}
					if !found || value != winner {
						return fmt.Errorf("winner key %d = %d, %v; want %d", key, value, found, winner)
					}
				}
				if test.name == "distinct keys" {
					for _, key := range loserKeys {
						if _, found, err := values.Get(transaction, key); err != nil {
							return err
						} else if found {
							return fmt.Errorf("loser key %d was partially published", key)
						}
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConcurrentReadersObserveOnlyCompleteCommits(t *testing.T) {
	t.Parallel()

	builder, err := NewLayoutBuilder("example.concurrent-readers")
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

	const keyCount = 128
	if _, err := Execute(store, layout, concurrentTestLimits(), func(transaction *Transaction) error {
		for key := uint32(0); key < keyCount; key++ {
			if err := values.Set(transaction, key, 0); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	writer, err := store.Begin(layout, concurrentTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	for key := uint32(0); key < keyCount; key++ {
		if err := values.Set(writer, key, 1); err != nil {
			t.Fatal(err)
		}
	}

	const readerCount = 32
	start := make(chan struct{})
	writerResult := make(chan error, 1)
	readerResults := make(chan error, readerCount)

	go func() {
		<-start
		_, err := writer.Commit()
		writerResult <- err
	}()
	for reader := 0; reader < readerCount; reader++ {
		go func() {
			<-start
			readerResults <- View(store, layout, concurrentTestLimits(), func(transaction *Transaction) error {
				var generation uint64
				for key := uint32(0); key < keyCount; key++ {
					value, found, err := values.Get(transaction, key)
					if err != nil {
						return err
					}
					if !found {
						return fmt.Errorf("key %d is missing from a committed snapshot", key)
					}
					if key == 0 {
						generation = value
					}
					if value != generation {
						return fmt.Errorf("mixed snapshot at key %d: first=%d current=%d", key, generation, value)
					}
				}
				if generation != 0 && generation != 1 {
					return fmt.Errorf("unexpected committed generation %d", generation)
				}
				return nil
			})
		}()
	}

	close(start)
	if err := <-writerResult; err != nil {
		t.Fatalf("writer Commit() error = %v", err)
	}
	for reader := 0; reader < readerCount; reader++ {
		if err := <-readerResults; err != nil {
			t.Fatalf("reader %d: %v", reader, err)
		}
	}
}

func beginCellPairTransaction(
	t *testing.T,
	store *MemoryStore,
	layout Layout,
	left Cell[uint64],
	right Cell[uint64],
	value uint64,
) preparedConcurrentCommit {
	t.Helper()
	transaction, err := store.Begin(layout, concurrentTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := left.Set(transaction, value); err != nil {
		t.Fatal(err)
	}
	if err := right.Set(transaction, value); err != nil {
		t.Fatal(err)
	}
	return preparedConcurrentCommit{name: fmt.Sprintf("value-%d", value), marker: value, transaction: transaction}
}

func beginMapTransaction(
	t *testing.T,
	store *MemoryStore,
	layout Layout,
	values Map[uint32, uint64],
	name string,
	value uint64,
	keys []uint32,
) preparedConcurrentCommit {
	t.Helper()
	transaction, err := store.Begin(layout, concurrentTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if err := values.Set(transaction, key, value); err != nil {
			t.Fatal(err)
		}
	}
	return preparedConcurrentCommit{name: name, marker: value, transaction: transaction}
}

func assertOneConcurrentCommit(t *testing.T, first, second preparedConcurrentCommit) uint64 {
	t.Helper()
	start := make(chan struct{})
	results := make(chan concurrentCommitResult, 2)
	for _, prepared := range []preparedConcurrentCommit{first, second} {
		prepared := prepared
		go func() {
			<-start
			receipt, err := prepared.transaction.Commit()
			results <- concurrentCommitResult{
				name: prepared.name, marker: prepared.marker, receipt: receipt, err: err,
			}
		}()
	}
	close(start)

	outcomes := []concurrentCommitResult{<-results, <-results}
	var winner uint64
	successes := 0
	conflicts := 0
	for _, outcome := range outcomes {
		switch {
		case outcome.err == nil:
			if outcome.receipt.Version != 1 || outcome.receipt.Writes != 2 {
				t.Fatalf("%s receipt = %+v, want version 1 with 2 writes", outcome.name, outcome.receipt)
			}
			successes++
			winner = outcome.marker
		case errors.Is(outcome.err, ErrConflict):
			conflicts++
		case outcome.err != nil:
			t.Fatalf("%s Commit() error = %v", outcome.name, outcome.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("commit outcomes = %+v; want one success and one conflict", outcomes)
	}
	return winner
}

func concurrentTestLimits() Limits {
	return Limits{MaxWrites: 256, MaxWriteBytes: 64 * 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}
