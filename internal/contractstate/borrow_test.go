package contractstate

import (
	"errors"
	"testing"
)

func TestOwnerBoundCellBorrowInvalidatesOnReplacement(t *testing.T) {
	store, layout, cell, _ := borrowFixture(t)
	transaction, err := store.Begin(layout, borrowLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := cell.Set(transaction, []byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	borrow, found, err := cell.Borrow(transaction)
	if err != nil || !found {
		t.Fatalf("borrow = (%+v, %t, %v)", borrow.Origin(), found, err)
	}
	view, err := borrow.View(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := view.Bytes()
	if err != nil || string(bytes) != "bcd" {
		t.Fatalf("view = %q, %v", bytes, err)
	}
	bytes[0] = 'X'
	again, _ := view.Bytes()
	if string(again) != "bcd" {
		t.Fatalf("borrow exposed mutable storage: %q", again)
	}
	if err := cell.Set(transaction, []byte("next")); err != nil {
		t.Fatal(err)
	}
	if _, err := borrow.Bytes(); !errors.Is(err, ErrBorrowInvalidated) {
		t.Fatalf("replaced borrow error = %v", err)
	}
	if _, err := view.Bytes(); !errors.Is(err, ErrBorrowInvalidated) {
		t.Fatalf("nested replaced borrow error = %v", err)
	}
	replacement, found, err := cell.Borrow(transaction)
	if err != nil || !found {
		t.Fatalf("replacement borrow = (%+v, %t, %v)", replacement.Origin(), found, err)
	}
	if bytes, err := replacement.Bytes(); err != nil || string(bytes) != "next" {
		t.Fatalf("replacement view = %q, %v", bytes, err)
	}
}

func TestOwnerBoundMapBorrowUsesExactEntryIdentity(t *testing.T) {
	store, layout, _, mapped := borrowFixture(t)
	transaction, err := store.Begin(layout, borrowLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := mapped.Set(transaction, []byte("a"), []byte("alpha")); err != nil {
		t.Fatal(err)
	}
	borrow, found, err := mapped.Borrow(transaction, []byte("a"))
	if err != nil || !found {
		t.Fatalf("borrow = (%+v, %t, %v)", borrow.Origin(), found, err)
	}
	if err := mapped.Set(transaction, []byte("b"), []byte("beta")); err != nil {
		t.Fatal(err)
	}
	if bytes, err := borrow.Bytes(); err != nil || string(bytes) != "alpha" {
		t.Fatalf("unrelated-key borrow = %q, %v", bytes, err)
	}
	if err := mapped.Delete(transaction, []byte("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := borrow.Bytes(); !errors.Is(err, ErrBorrowInvalidated) {
		t.Fatalf("deleted borrow error = %v", err)
	}
}

func TestOwnerBoundBorrowCannotEscapeOrCrossRetry(t *testing.T) {
	store, layout, cell, _ := borrowFixture(t)
	first, err := store.Begin(layout, borrowLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := cell.Set(first, []byte("value")); err != nil {
		t.Fatal(err)
	}
	borrow, _, err := cell.Borrow(first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := borrow.Bytes(); !errors.Is(err, ErrClosed) {
		t.Fatalf("escaped borrow error = %v", err)
	}
	second, err := store.Begin(layout, borrowLimits())
	if err != nil {
		t.Fatal(err)
	}
	retryBorrow, found, err := cell.Borrow(second)
	if err != nil || !found {
		t.Fatalf("retry borrow = (%+v, %t, %v)", retryBorrow.Origin(), found, err)
	}
	if _, err := borrow.Bytes(); !errors.Is(err, ErrClosed) {
		t.Fatalf("cross-retry borrow error = %v", err)
	}
	if !second.Rollback() {
		t.Fatal("retry rollback did not close")
	}
	if _, err := retryBorrow.Bytes(); !errors.Is(err, ErrClosed) {
		t.Fatalf("rolled-back retry borrow error = %v", err)
	}
}

func borrowFixture(t *testing.T) (*MemoryStore, Layout, EncodedCell, EncodedMap) {
	builder, err := NewLayoutBuilder("borrow.fixture")
	if err != nil {
		t.Fatal(err)
	}
	cell, err := DeclareEncodedCell(builder, "cell", Bytes.TypeID())
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := DeclareEncodedMap(builder, "map", Bytes.TypeID(), Bytes.TypeID())
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
	return store, layout, cell, mapped
}

func borrowLimits() Limits {
	return Limits{MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}
