package contractstate

import (
	"errors"
	"testing"
)

func TestExecuteReturnsConflictWithoutHiddenRetry(t *testing.T) {
	t.Parallel()

	builder, err := NewLayoutBuilder("example.explicit-retry")
	if err != nil {
		t.Fatal(err)
	}
	counter, err := DeclareCell(builder, "counter", U64)
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
	limits := retryTestLimits()

	bodyCalls := 0
	injectConflict := true
	run := func() error {
		_, err := Execute(store, layout, limits, func(transaction *Transaction) error {
			bodyCalls++
			value, found, err := counter.Get(transaction)
			if err != nil {
				return err
			}
			if !found {
				value = 0
			}
			if err := counter.Set(transaction, value+1); err != nil {
				return err
			}
			if !injectConflict {
				return nil
			}
			injectConflict = false

			competing, err := store.Begin(layout, limits)
			if err != nil {
				return err
			}
			if err := counter.Set(competing, 40); err != nil {
				return err
			}
			_, err = competing.Commit()
			return err
		})
		return err
	}

	if err := run(); !errors.Is(err, ErrConflict) {
		t.Fatalf("first Execute() error = %v, want ErrConflict", err)
	}
	if bodyCalls != 1 {
		t.Fatalf("body calls after conflict = %d, want 1", bodyCalls)
	}
	assertCellValue(t, store, layout, counter, 40)

	if err := run(); err != nil {
		t.Fatalf("explicit retry Execute() error = %v", err)
	}
	if bodyCalls != 2 {
		t.Fatalf("body calls after explicit retry = %d, want 2", bodyCalls)
	}
	assertCellValue(t, store, layout, counter, 41)
	if store.Version() != 2 {
		t.Fatalf("store version = %d, want 2", store.Version())
	}
}

func TestExecuteDoesNotRetryBodyOrLimitFailures(t *testing.T) {
	t.Parallel()

	builder, err := NewLayoutBuilder("example.no-failure-retry")
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

	t.Run("returned error", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		bodyCalls := 0
		rejected := errors.New("rejected")
		_, err = Execute(store, layout, retryTestLimits(), func(transaction *Transaction) error {
			bodyCalls++
			if err := left.Set(transaction, 1); err != nil {
				return err
			}
			return rejected
		})
		if !errors.Is(err, rejected) {
			t.Fatalf("Execute() error = %v, want rejection", err)
		}
		if bodyCalls != 1 {
			t.Fatalf("body calls = %d, want 1", bodyCalls)
		}
		if store.Version() != 0 || len(store.Dump()) != 0 {
			t.Fatalf("rejected execution published state: version=%d entries=%v", store.Version(), store.Dump())
		}
	})

	t.Run("panic", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		bodyCalls := 0
		func() {
			defer func() {
				if recovered := recover(); recovered != "trap" {
					t.Fatalf("panic = %v, want trap", recovered)
				}
			}()
			_, _ = Execute(store, layout, retryTestLimits(), func(transaction *Transaction) error {
				bodyCalls++
				if err := left.Set(transaction, 1); err != nil {
					return err
				}
				panic("trap")
			})
		}()
		if bodyCalls != 1 {
			t.Fatalf("body calls = %d, want 1", bodyCalls)
		}
		if store.Version() != 0 || len(store.Dump()) != 0 {
			t.Fatalf("panic published state: version=%d entries=%v", store.Version(), store.Dump())
		}
	})

	t.Run("limit failure", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		bodyCalls := 0
		limits := Limits{MaxWrites: 1, MaxWriteBytes: 64, MaxValueBytes: 8, MaxMapKeyBytes: 8}
		_, err = Execute(store, layout, limits, func(transaction *Transaction) error {
			bodyCalls++
			if err := left.Set(transaction, 1); err != nil {
				return err
			}
			return right.Set(transaction, 2)
		})
		if !errors.Is(err, ErrWriteLimit) {
			t.Fatalf("Execute() error = %v, want ErrWriteLimit", err)
		}
		if bodyCalls != 1 {
			t.Fatalf("body calls = %d, want 1", bodyCalls)
		}
		if store.Version() != 0 || len(store.Dump()) != 0 {
			t.Fatalf("limit failure published state: version=%d entries=%v", store.Version(), store.Dump())
		}
	})
}

func assertCellValue(
	t *testing.T,
	store *MemoryStore,
	layout Layout,
	cell Cell[uint64],
	expected uint64,
) {
	t.Helper()
	if err := View(store, layout, retryTestLimits(), func(transaction *Transaction) error {
		value, found, err := cell.Get(transaction)
		if err != nil {
			return err
		}
		if !found || value != expected {
			t.Fatalf("cell = %d, %v; want %d", value, found, expected)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func retryTestLimits() Limits {
	return Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}
