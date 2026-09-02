package contractstate

import (
	"errors"
	"testing"
)

func TestReferencePersistenceCompareAndPublish(t *testing.T) {
	layout, first, second := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 0 || len(snapshot.Entries) != 0 {
		t.Fatalf("initial snapshot = %+v", snapshot)
	}

	receipt, err := store.CompareAndPublish(layout.Hash(), snapshot.Version, []PersistentWrite{
		{Key: second, Value: []byte("second")},
		{Key: first, Value: []byte("first")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Version != 1 || receipt.Writes != 2 || receipt.LayoutHash != layout.Hash() {
		t.Fatalf("receipt = %+v", receipt)
	}
	after, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != 1 || len(after.Entries) != 2 {
		t.Fatalf("published snapshot = %+v", after)
	}
	want := map[Key]string{first: "first", second: "second"}
	for _, entry := range after.Entries {
		if string(entry.Value) != want[entry.Key] {
			t.Fatalf("published entry = %+v, want value %q", entry, want[entry.Key])
		}
		delete(want, entry.Key)
	}
	if len(want) != 0 {
		t.Fatalf("published snapshot omitted keys: %+v", want)
	}
}

func TestReferencePersistenceRejectsStaleAndWrongLayoutWithoutPartialWrites(t *testing.T) {
	layout, first, second := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: []byte("base")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: []byte("stale")}, {Key: second, Value: []byte("leak")}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale error = %v", err)
	}
	var wrong LayoutHash
	wrong[0] = 1
	if _, err := store.CompareAndPublish(wrong, 1, []PersistentWrite{{Key: second, Value: []byte("wrong-layout")}}); !errors.Is(err, ErrLayoutMismatch) {
		t.Fatalf("layout error = %v", err)
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 1 || len(snapshot.Entries) != 1 || string(snapshot.Entries[0].Value) != "base" {
		t.Fatalf("failed publish changed state: %+v", snapshot)
	}
}

func TestReferencePersistenceConcurrentPublishHasOneWinner(t *testing.T) {
	layout, first, second := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, write := range []PersistentWrite{{Key: first, Value: []byte("first")}, {Key: second, Value: []byte("second")}} {
		write := write
		go func() {
			<-start
			_, publishErr := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{write})
			results <- publishErr
		}()
	}
	close(start)
	successes := 0
	conflicts := 0
	for range 2 {
		switch publishErr := <-results; {
		case publishErr == nil:
			successes++
		case errors.Is(publishErr, ErrConflict):
			conflicts++
		default:
			t.Fatalf("concurrent publish error = %v", publishErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent results = %d success, %d conflict", successes, conflicts)
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 1 || len(snapshot.Entries) != 1 {
		t.Fatalf("concurrent snapshot = %+v", snapshot)
	}
}

func TestReferencePersistenceFaultBoundariesAreAtomic(t *testing.T) {
	for _, point := range []PersistenceFaultPoint{FaultBeforeCompare, FaultBeforePublish} {
		t.Run(string(point), func(t *testing.T) {
			layout, first, second := persistenceFixtureLayout(t)
			fail := false
			store, err := NewReferencePersistence(layout, func(current PersistenceFaultPoint) error {
				if fail && current == point {
					return errors.New("injected durable failure")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: []byte("base")}}); err != nil {
				t.Fatal(err)
			}
			fail = true
			if _, err := store.CompareAndPublish(layout.Hash(), 1, []PersistentWrite{{Key: first, Value: []byte("changed")}, {Key: second, Value: []byte("partial")}}); err == nil {
				t.Fatal("injected failure succeeded")
			}
			fail = false
			snapshot, err := store.Snapshot(layout.Hash())
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.Version != 1 || len(snapshot.Entries) != 1 || string(snapshot.Entries[0].Value) != "base" {
				t.Fatalf("fault published partial state: %+v", snapshot)
			}
		})
	}
}

func TestReferencePersistenceDeleteAndDefensiveCopies(t *testing.T) {
	layout, first, _ := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte("owned")
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: value}}); err != nil {
		t.Fatal(err)
	}
	value[0] = 'X'
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot.Entries[0].Value) != "owned" {
		t.Fatal("caller mutation reached persistence")
	}
	snapshot.Entries[0].Value[0] = 'Y'
	again, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Entries[0].Value) != "owned" {
		t.Fatal("snapshot mutation reached persistence")
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 1, []PersistentWrite{{Key: first, Deleted: true}}); err != nil {
		t.Fatal(err)
	}
	after, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != 2 || len(after.Entries) != 0 {
		t.Fatalf("delete snapshot = %+v", after)
	}
}

func TestReferencePersistenceLayoutPublication(t *testing.T) {
	layout, _, _ := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := NewLayoutBuilder(layout.ContractUID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "first", U32); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "second", U32); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "third", U32); err != nil {
		t.Fatal(err)
	}
	next, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := store.PublishLayout(layout.Hash(), 0, next)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.LayoutHash != next.Hash() || receipt.Version != 1 {
		t.Fatalf("layout receipt = %+v", receipt)
	}
	if _, err := store.Snapshot(layout.Hash()); !errors.Is(err, ErrLayoutMismatch) {
		t.Fatalf("old layout snapshot error = %v", err)
	}
	if snapshot, err := store.Snapshot(next.Hash()); err != nil || snapshot.Version != 1 {
		t.Fatalf("new layout snapshot = (%+v, %v)", snapshot, err)
	}
}

func TestReferencePersistenceRejectsDuplicateAndInvalidDelete(t *testing.T) {
	layout, first, _ := persistenceFixtureLayout(t)
	store, err := NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: []byte("a")}, {Key: first, Value: []byte("b")}}); err == nil {
		t.Fatal("duplicate write succeeded")
	}
	if _, err := store.CompareAndPublish(layout.Hash(), 0, []PersistentWrite{{Key: first, Value: []byte("payload"), Deleted: true}}); err == nil {
		t.Fatal("delete with payload succeeded")
	}
	snapshot, _ := store.Snapshot(layout.Hash())
	if snapshot.Version != 0 || len(snapshot.Entries) != 0 {
		t.Fatalf("invalid request changed state: %+v", snapshot)
	}
}

func persistenceFixtureLayout(t *testing.T) (Layout, Key, Key) {
	t.Helper()
	builder, err := NewLayoutBuilder("tests.persistence")
	if err != nil {
		t.Fatal(err)
	}
	first, err := DeclareCell(builder, "first", U32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeclareCell(builder, "second", U32)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return layout, first.Field().Prefix, second.Field().Prefix
}
