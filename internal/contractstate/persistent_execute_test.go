package contractstate

import (
	"errors"
	"testing"
)

type snapshotOverride struct {
	PersistencePort
	snapshot PersistentSnapshot
}

func (port snapshotOverride) Snapshot(LayoutHash) (PersistentSnapshot, error) {
	return port.snapshot, nil
}

func TestExecutePersistentRejectsMalformedSnapshotsBeforeBody(t *testing.T) {
	layout, first, _ := persistenceFixtureLayout(t)
	for _, snapshot := range []PersistentSnapshot{
		{},
		{LayoutHash: layout.Hash(), Entries: []Entry{{Key: first}, {Key: first}}},
		{LayoutHash: layout.Hash(), Entries: []Entry{{Key: Key{255}}, {Key: Key{1}}}},
	} {
		called := false
		if _, err := ExecutePersistent(snapshotOverride{snapshot: snapshot}, layout, testLimits(), func(*Transaction) error { called = true; return nil }); err == nil || called {
			t.Fatalf("malformed snapshot reached body: %+v, error %v", snapshot, err)
		}
	}
}

func TestExecutePersistentLimitsLifecycleAndBackendFailure(t *testing.T) {
	layout, cell, _ := testBankLayout(t, false)
	failure := errors.New("backend offline")
	port, err := NewReferencePersistence(layout, func(point PersistenceFaultPoint) error {
		if point == FaultBeforePublish {
			return failure
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var escaped *Transaction
	_, err = ExecutePersistent(port, layout, testLimits(), func(transaction *Transaction) error {
		escaped = transaction
		if _, err := transaction.Commit(); !errors.Is(err, ErrManagedLifecycle) {
			t.Fatalf("early commit = %v", err)
		}
		if transaction.Rollback() {
			t.Fatal("body stole lifecycle")
		}
		return cell.Set(transaction, 12)
	})
	if !errors.Is(err, failure) {
		t.Fatalf("backend error = %v", err)
	}
	if err := cell.Set(escaped, 13); !errors.Is(err, ErrClosed) {
		t.Fatalf("escaped transaction = %v", err)
	}
	limits := testLimits()
	limits.MaxValueBytes = 1
	if _, err := ExecutePersistent(port, layout, limits, func(transaction *Transaction) error { return cell.Set(transaction, 12) }); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("limit = %v", err)
	}
	snapshot, err := port.Snapshot(layout.Hash())
	if err != nil || snapshot.Version != 0 || len(snapshot.Entries) != 0 {
		t.Fatalf("failure leaked: %+v/%v", snapshot, err)
	}
}
