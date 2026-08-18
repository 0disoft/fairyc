package contractstate

import (
	"fmt"
	"math"
	"slices"
	"sync"
)

type transactionState uint8

const (
	transactionOpen transactionState = iota + 1
	transactionCommitted
	transactionRolledBack
)

type pendingWrite struct {
	value   []byte
	deleted bool
}

// CommitReceipt is the exact successful write-overlay publication result.
type CommitReceipt struct {
	LayoutHash LayoutHash
	Version    uint64
	Writes     int
	WriteBytes uint64
}

// TransactionStats exposes bounded execution facts without exposing values.
type TransactionStats struct {
	Reads      uint64
	Writes     int
	WriteBytes uint64
	ReadOnly   bool
}

// Transaction owns one immutable committed snapshot and one mutable bounded
// overlay. It is safe for concurrent method calls, although deterministic
// contract execution should ordinarily use one goroutine.
type Transaction struct {
	mu          sync.Mutex
	store       *MemoryStore
	layout      Layout
	baseVersion uint64
	limits      Limits
	readOnly    bool
	state       transactionState
	snapshot    map[Key][]byte
	writes      map[Key]pendingWrite
	reads       uint64
	writeBytes  uint64
}

func (store *MemoryStore) Begin(layout Layout, limits Limits) (*Transaction, error) {
	return store.begin(layout, limits, false)
}

func (store *MemoryStore) BeginReadOnly(layout Layout, limits Limits) (*Transaction, error) {
	return store.begin(layout, limits, true)
}

func (store *MemoryStore) begin(layout Layout, limits Limits, readOnly bool) (*Transaction, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}
	baseVersion, snapshot, err := store.snapshot(layout)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		store:       store,
		layout:      cloneLayout(layout),
		baseVersion: baseVersion,
		limits:      limits,
		readOnly:    readOnly,
		state:       transactionOpen,
		snapshot:    snapshot,
		writes:      make(map[Key]pendingWrite),
	}, nil
}

// Execute commits only after body returns nil. A returned error rolls back. A
// panic also rolls back and is re-panicked so a future trap path cannot be
// accidentally converted into ordinary success.
func Execute(store *MemoryStore, layout Layout, limits Limits, body func(*Transaction) error) (receipt CommitReceipt, err error) {
	if body == nil {
		return CommitReceipt{}, fmt.Errorf("contract state transaction body is nil")
	}
	transaction, err := store.Begin(layout, limits)
	if err != nil {
		return CommitReceipt{}, err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			transaction.Rollback()
			panic(recovered)
		}
	}()
	if err := body(transaction); err != nil {
		transaction.Rollback()
		return CommitReceipt{}, err
	}
	return transaction.Commit()
}

// View executes against one immutable snapshot and always closes by rollback.
func View(store *MemoryStore, layout Layout, limits Limits, body func(*Transaction) error) error {
	if body == nil {
		return fmt.Errorf("contract state view body is nil")
	}
	transaction, err := store.BeginReadOnly(layout, limits)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	return body(transaction)
}

func (transaction *Transaction) Stats() TransactionStats {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	return TransactionStats{
		Reads:      transaction.reads,
		Writes:     len(transaction.writes),
		WriteBytes: transaction.writeBytes,
		ReadOnly:   transaction.readOnly,
	}
}

func (transaction *Transaction) Commit() (CommitReceipt, error) {
	if transaction == nil {
		return CommitReceipt{}, fmt.Errorf("contract state transaction is nil")
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.ensureOpenLocked(); err != nil {
		return CommitReceipt{}, err
	}
	version, err := transaction.store.commit(transaction.layout.hash, transaction.baseVersion, transaction.writes)
	if err != nil {
		transaction.closeLocked(transactionRolledBack)
		return CommitReceipt{}, err
	}
	receipt := CommitReceipt{
		LayoutHash: transaction.layout.hash,
		Version:    version,
		Writes:     len(transaction.writes),
		WriteBytes: transaction.writeBytes,
	}
	transaction.closeLocked(transactionCommitted)
	return receipt, nil
}

// Rollback discards the overlay. It reports whether this call closed an open
// transaction.
func (transaction *Transaction) Rollback() bool {
	if transaction == nil {
		return false
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if transaction.state != transactionOpen {
		return false
	}
	transaction.closeLocked(transactionRolledBack)
	return true
}

func (transaction *Transaction) readCell(field Field) ([]byte, bool, error) {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldCell); err != nil {
		return nil, false, err
	}
	return transaction.readKeyLocked(field.Prefix)
}

func (transaction *Transaction) writeCell(field Field, value []byte) error {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldCell); err != nil {
		return err
	}
	return transaction.writeKeyLocked(field.Prefix, value, false)
}

func (transaction *Transaction) deleteCell(field Field) error {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldCell); err != nil {
		return err
	}
	return transaction.writeKeyLocked(field.Prefix, nil, true)
}

func (transaction *Transaction) readMap(field Field, encodedKey []byte) ([]byte, bool, error) {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldMap); err != nil {
		return nil, false, err
	}
	key, err := transaction.mapEntryKeyLocked(field, encodedKey)
	if err != nil {
		return nil, false, err
	}
	return transaction.readKeyLocked(key)
}

func (transaction *Transaction) writeMap(field Field, encodedKey, value []byte) error {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldMap); err != nil {
		return err
	}
	key, err := transaction.mapEntryKeyLocked(field, encodedKey)
	if err != nil {
		return err
	}
	return transaction.writeKeyLocked(key, value, false)
}

func (transaction *Transaction) deleteMap(field Field, encodedKey []byte) error {
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if err := transaction.validateFieldLocked(field, FieldMap); err != nil {
		return err
	}
	key, err := transaction.mapEntryKeyLocked(field, encodedKey)
	if err != nil {
		return err
	}
	return transaction.writeKeyLocked(key, nil, true)
}

func (transaction *Transaction) validateFieldLocked(field Field, expectedKind FieldKind) error {
	if err := transaction.ensureOpenLocked(); err != nil {
		return err
	}
	if field.Kind != expectedKind {
		return fmt.Errorf("contract state field %q is %s, expected %s", field.Name, field.Kind, expectedKind)
	}
	index := int(field.Index)
	if index < 0 || index >= len(transaction.layout.fields) || transaction.layout.fields[index] != field {
		return fmt.Errorf("%w: field %q is absent or changed", ErrLayoutMismatch, field.Name)
	}
	return nil
}

func (transaction *Transaction) mapEntryKeyLocked(field Field, encodedKey []byte) (Key, error) {
	if len(encodedKey) > transaction.limits.MaxMapKeyBytes {
		return Key{}, fmt.Errorf("%w: map %q encoded %d bytes, limit %d", ErrMapKeyTooLarge, field.Name, len(encodedKey), transaction.limits.MaxMapKeyBytes)
	}
	return deriveMapEntryKey(field.Prefix, encodedKey), nil
}

func (transaction *Transaction) readKeyLocked(key Key) ([]byte, bool, error) {
	if err := transaction.ensureOpenLocked(); err != nil {
		return nil, false, err
	}
	transaction.reads++
	if write, found := transaction.writes[key]; found {
		if write.deleted {
			return nil, false, nil
		}
		return slices.Clone(write.value), true, nil
	}
	value, found := transaction.snapshot[key]
	if !found {
		return nil, false, nil
	}
	return slices.Clone(value), true, nil
}

func (transaction *Transaction) writeKeyLocked(key Key, value []byte, deleted bool) error {
	if err := transaction.ensureOpenLocked(); err != nil {
		return err
	}
	if transaction.readOnly {
		return ErrReadOnly
	}
	if !deleted && len(value) > transaction.limits.MaxValueBytes {
		return fmt.Errorf("%w: encoded %d bytes, limit %d", ErrValueTooLarge, len(value), transaction.limits.MaxValueBytes)
	}
	oldWrite, replacing := transaction.writes[key]
	writeCount := len(transaction.writes)
	if !replacing {
		writeCount++
	}
	if writeCount > transaction.limits.MaxWrites {
		return fmt.Errorf("%w: %d keys, limit %d", ErrWriteLimit, writeCount, transaction.limits.MaxWrites)
	}
	newWrite := pendingWrite{value: slices.Clone(value), deleted: deleted}
	oldCost := uint64(0)
	if replacing {
		oldCost = pendingWriteCost(oldWrite)
	}
	newCost := pendingWriteCost(newWrite)
	if transaction.writeBytes < oldCost {
		return fmt.Errorf("contract state internal write-byte accounting underflow")
	}
	nextBytes := transaction.writeBytes - oldCost
	if math.MaxUint64-nextBytes < newCost {
		return ErrWriteBytesLimit
	}
	nextBytes += newCost
	if nextBytes > transaction.limits.MaxWriteBytes {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrWriteBytesLimit, nextBytes, transaction.limits.MaxWriteBytes)
	}
	transaction.writes[key] = newWrite
	transaction.writeBytes = nextBytes
	return nil
}

func pendingWriteCost(write pendingWrite) uint64 {
	// Fixed key plus one operation byte keeps deletion and empty-value writes
	// distinct in accounting.
	return uint64(len(Key{})) + 1 + uint64(len(write.value))
}

func (transaction *Transaction) ensureOpenLocked() error {
	if transaction.state != transactionOpen {
		return ErrClosed
	}
	return nil
}

func (transaction *Transaction) closeLocked(state transactionState) {
	transaction.state = state
	transaction.snapshot = nil
	transaction.writes = nil
	transaction.writeBytes = 0
}
