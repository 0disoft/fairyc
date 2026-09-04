package contractstate

import (
	"fmt"
	"slices"
)

type borrowToken struct{ marker byte }

// BorrowOrigin identifies the exact transaction snapshot and storage address
// from which one owner-bound view was created. It carries no pointer authority.
type BorrowOrigin struct {
	LayoutHash      LayoutHash
	SnapshotVersion uint64
	FieldIndex      uint32
	StorageKey      Key
	TypeID          string
	Generation      uint64
}

// BorrowedValue is an opaque view owned by one open transaction. Bytes always
// returns a copy, so callers cannot derive a raw pointer into snapshot or overlay
// storage. Nested views retain the same origin and invalidation generation.
type BorrowedValue struct {
	transaction *Transaction
	token       *borrowToken
	origin      BorrowOrigin
	encoded     []byte
	offset      int
	length      int
}

func (borrow BorrowedValue) Origin() BorrowOrigin { return borrow.origin }

func (borrow BorrowedValue) Bytes() ([]byte, error) {
	if borrow.transaction == nil {
		return nil, ErrBorrowInvalidated
	}
	borrow.transaction.mu.Lock()
	defer borrow.transaction.mu.Unlock()
	if err := borrow.validateLocked(); err != nil {
		return nil, err
	}
	return slices.Clone(borrow.encoded[borrow.offset : borrow.offset+borrow.length]), nil
}

// View derives a bounded nested view without changing storage provenance.
func (borrow BorrowedValue) View(offset, length int) (BorrowedValue, error) {
	if borrow.transaction == nil {
		return BorrowedValue{}, ErrBorrowInvalidated
	}
	borrow.transaction.mu.Lock()
	defer borrow.transaction.mu.Unlock()
	if err := borrow.validateLocked(); err != nil {
		return BorrowedValue{}, err
	}
	if offset < 0 || length < 0 || offset > borrow.length || length > borrow.length-offset {
		return BorrowedValue{}, fmt.Errorf("contract state borrow view range is outside the owner view")
	}
	borrow.offset += offset
	borrow.length = length
	return borrow, nil
}

func (borrow BorrowedValue) validateLocked() error {
	transaction := borrow.transaction
	if err := transaction.ensureOpenLocked(); err != nil {
		return err
	}
	if borrow.token == nil || transaction.borrowToken != borrow.token || transaction.borrowEpoch[borrow.origin.StorageKey] != borrow.origin.Generation {
		return ErrBorrowInvalidated
	}
	if borrow.offset < 0 || borrow.length < 0 || borrow.offset > len(borrow.encoded) || borrow.length > len(borrow.encoded)-borrow.offset {
		return ErrBorrowInvalidated
	}
	return nil
}

func (transaction *Transaction) borrowCell(field Field) (BorrowedValue, bool, error) {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldCell); err != nil {
		return BorrowedValue{}, false, err
	}
	return transaction.borrowKeyLocked(field, field.Prefix)
}

func (transaction *Transaction) borrowMap(field Field, encodedKey []byte) (BorrowedValue, bool, error) {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldMap); err != nil {
		return BorrowedValue{}, false, err
	}
	key, err := transaction.mapEntryKeyLocked(field, encodedKey)
	if err != nil {
		return BorrowedValue{}, false, err
	}
	return transaction.borrowKeyLocked(field, key)
}

func (transaction *Transaction) borrowKeyLocked(field Field, key Key) (BorrowedValue, bool, error) {
	encoded, found, err := transaction.readKeyLocked(key)
	if err != nil || !found {
		return BorrowedValue{}, found, err
	}
	origin := BorrowOrigin{
		LayoutHash: transaction.layout.hash, SnapshotVersion: transaction.baseVersion,
		FieldIndex: field.Index, StorageKey: key, TypeID: field.ValueTypeID, Generation: transaction.borrowEpoch[key],
	}
	return BorrowedValue{
		transaction: transaction, token: transaction.borrowToken, origin: origin,
		encoded: encoded, length: len(encoded),
	}, true, nil
}
