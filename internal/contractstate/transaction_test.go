package contractstate

import (
	"errors"
	"testing"
)

func TestTypedCellAndMapCommitAndView(t *testing.T) {
	t.Parallel()
	layout, supply, balances := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	var alice, bob Bytes20
	alice[19] = 1
	bob[19] = 2
	receipt, err := Execute(store, layout, testLimits(), func(transaction *Transaction) error {
		if err := supply.Set(transaction, 100); err != nil {
			return err
		}
		if err := balances.Set(transaction, alice, 40); err != nil {
			return err
		}
		if err := balances.Set(transaction, bob, 60); err != nil {
			return err
		}
		observed, found, err := balances.Get(transaction, alice)
		if err != nil {
			return err
		}
		if !found || observed != 40 {
			t.Fatalf("read-your-writes balance = %d, %v", observed, found)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.LayoutHash != layout.Hash() || receipt.Version != 1 || receipt.Writes != 3 {
		t.Fatalf("receipt = %+v", receipt)
	}
	if entries := store.Dump(); len(entries) != 3 {
		t.Fatalf("Dump() length = %d", len(entries))
	}

	if err := View(store, layout, testLimits(), func(transaction *Transaction) error {
		gotSupply, found, err := supply.Get(transaction)
		if err != nil {
			return err
		}
		if !found || gotSupply != 100 {
			t.Fatalf("supply = %d, %v", gotSupply, found)
		}
		gotAlice, found, err := balances.Get(transaction, alice)
		if err != nil {
			return err
		}
		if !found || gotAlice != 40 {
			t.Fatalf("alice = %d, %v", gotAlice, found)
		}
		if err := balances.Set(transaction, alice, 0); !errors.Is(err, ErrReadOnly) {
			t.Fatalf("read-only Set() error = %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteRollsBackErrorAndPanic(t *testing.T) {
	t.Parallel()
	layout, supply, _ := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	limits := testLimits()
	if _, err := Execute(store, layout, limits, func(transaction *Transaction) error {
		return supply.Set(transaction, 7)
	}); err != nil {
		t.Fatal(err)
	}
	baseVersion := store.Version()

	rejected := errors.New("contract rejected")
	if _, err := Execute(store, layout, limits, func(transaction *Transaction) error {
		if err := supply.Set(transaction, 99); err != nil {
			return err
		}
		return rejected
	}); !errors.Is(err, rejected) {
		t.Fatalf("Execute(error) = %v", err)
	}
	if store.Version() != baseVersion {
		t.Fatalf("error rollback changed version to %d", store.Version())
	}
	assertSupply(t, store, layout, supply, 7)

	func() {
		defer func() {
			if recovered := recover(); recovered != "trap" {
				t.Fatalf("panic = %v", recovered)
			}
		}()
		_, _ = Execute(store, layout, limits, func(transaction *Transaction) error {
			if err := supply.Set(transaction, 123); err != nil {
				return err
			}
			panic("trap")
		})
	}()
	if store.Version() != baseVersion {
		t.Fatalf("panic rollback changed version to %d", store.Version())
	}
	assertSupply(t, store, layout, supply, 7)
}

func TestExecuteOwnsManagedTransactionLifecycle(t *testing.T) {
	t.Parallel()
	layout, supply, _ := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(store, layout, testLimits(), func(transaction *Transaction) error {
		if err := supply.Set(transaction, 99); err != nil {
			return err
		}
		if transaction.Rollback() {
			t.Fatal("managed callback closed its transaction")
		}
		_, err := transaction.Commit()
		return err
	}); !errors.Is(err, ErrManagedLifecycle) {
		t.Fatalf("managed Commit() error = %v", err)
	}
	if store.Version() != 0 || len(store.Dump()) != 0 {
		t.Fatalf("managed lifecycle escape published state: version=%d dump=%v", store.Version(), store.Dump())
	}
}

func TestOverlayDeleteAndRollback(t *testing.T) {
	t.Parallel()
	layout, supply, _ := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := store.Begin(layout, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := supply.Set(transaction, 9); err != nil {
		t.Fatal(err)
	}
	if value, found, err := supply.Get(transaction); err != nil || !found || value != 9 {
		t.Fatalf("Get(after Set) = %d, %v, %v", value, found, err)
	}
	if err := supply.Delete(transaction); err != nil {
		t.Fatal(err)
	}
	if _, found, err := supply.Get(transaction); err != nil || found {
		t.Fatalf("Get(after Delete) found = %v, error = %v", found, err)
	}
	if !transaction.Rollback() || transaction.Rollback() {
		t.Fatal("Rollback() close behavior is incorrect")
	}
	if _, _, err := supply.Get(transaction); !errors.Is(err, ErrClosed) {
		t.Fatalf("Get(closed) error = %v", err)
	}
	if len(store.Dump()) != 0 || store.Version() != 0 {
		t.Fatalf("rolled-back store = %+v, version %d", store.Dump(), store.Version())
	}
}

func TestStaleTransactionCannotPartiallyCommit(t *testing.T) {
	t.Parallel()
	layout, supply, balances := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	limits := testLimits()
	first, err := store.Begin(layout, limits)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Begin(layout, limits)
	if err != nil {
		t.Fatal(err)
	}
	var alice Bytes20
	alice[0] = 1
	if err := supply.Set(first, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := supply.Set(second, 20); err != nil {
		t.Fatal(err)
	}
	if err := balances.Set(second, alice, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Commit(); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale Commit() error = %v", err)
	}
	assertSupply(t, store, layout, supply, 10)
	if err := View(store, layout, limits, func(transaction *Transaction) error {
		if _, found, err := balances.Get(transaction, alice); err != nil || found {
			t.Fatalf("stale map write found = %v, error = %v", found, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteLimitsRejectOnlyTheOffendingMutation(t *testing.T) {
	t.Parallel()
	layout, supply, balances := testBankLayout(t, false)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	limits := Limits{MaxWrites: 1, MaxWriteBytes: 64, MaxValueBytes: 8, MaxMapKeyBytes: 20}
	transaction, err := store.Begin(layout, limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := supply.Set(transaction, 5); err != nil {
		t.Fatal(err)
	}
	var alice Bytes20
	if err := balances.Set(transaction, alice, 1); !errors.Is(err, ErrWriteLimit) {
		t.Fatalf("second key Set() error = %v", err)
	}
	if value, found, err := supply.Get(transaction); err != nil || !found || value != 5 {
		t.Fatalf("first pending write after rejection = %d, %v, %v", value, found, err)
	}
	if _, err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	assertSupply(t, store, layout, supply, 5)

	largeKeyLayoutBuilder, err := NewLayoutBuilder("example.large-key")
	if err != nil {
		t.Fatal(err)
	}
	largeMap, err := DeclareMap(largeKeyLayoutBuilder, "items", Bytes, U64)
	if err != nil {
		t.Fatal(err)
	}
	largeKeyLayout, err := largeKeyLayoutBuilder.Build()
	if err != nil {
		t.Fatal(err)
	}
	largeKeyStore, err := NewMemoryStore(largeKeyLayout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(largeKeyStore, largeKeyLayout, limits, func(transaction *Transaction) error {
		return largeMap.Set(transaction, make([]byte, 21), 1)
	}); !errors.Is(err, ErrMapKeyTooLarge) {
		t.Fatalf("oversize map key error = %v", err)
	}
}

func TestLayoutUpgradeInvalidatesOpenTransaction(t *testing.T) {
	t.Parallel()
	base, supply, _ := testBankLayout(t, false)
	extended, _, _ := testBankLayout(t, true)
	store, err := NewMemoryStore(base)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := store.Begin(base, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := supply.Set(transaction, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.Upgrade(extended); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Commit(); !errors.Is(err, ErrLayoutMismatch) {
		t.Fatalf("Commit(after upgrade) error = %v", err)
	}
	if len(store.Dump()) != 0 {
		t.Fatalf("stale transaction wrote entries: %+v", store.Dump())
	}
}

func assertSupply(t *testing.T, store *MemoryStore, layout Layout, supply Cell[uint64], expected uint64) {
	t.Helper()
	if err := View(store, layout, testLimits(), func(transaction *Transaction) error {
		value, found, err := supply.Get(transaction)
		if err != nil {
			return err
		}
		if !found || value != expected {
			t.Fatalf("supply = %d, %v; expected %d", value, found, expected)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
