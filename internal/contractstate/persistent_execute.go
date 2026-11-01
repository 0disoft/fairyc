package contractstate

import (
	"fmt"
	"slices"
)

type transactionStore interface {
	commit(LayoutHash, uint64, map[Key]pendingWrite) (uint64, error)
}

type persistenceCommitter struct{ port PersistencePort }

func (store persistenceCommitter) commit(layout LayoutHash, version uint64, writes map[Key]pendingWrite) (uint64, error) {
	mutations := make([]PersistentWrite, 0, len(writes))
	for key, write := range writes {
		mutations = append(mutations, PersistentWrite{Key: key, Value: slices.Clone(write.value), Deleted: write.deleted})
	}
	mutations, err := canonicalPersistentWrites(mutations)
	if err != nil {
		return 0, err
	}
	receipt, err := store.port.CompareAndPublish(layout, version, mutations)
	return receipt.Version, err
}

// ExecutePersistent performs exactly one caller-visible attempt. The existing
// bounded overlay owns lifecycle and rollback; the port alone owns publication.
// Rejection, panic and execution errors never call CompareAndPublish. Conflicts
// remain visible to the caller and are never retried here.
func ExecutePersistent(port PersistencePort, layout Layout, limits Limits, body func(*Transaction) error) (CommitReceipt, error) {
	if port == nil || body == nil {
		return CommitReceipt{}, fmt.Errorf("contract state persistence and body must be initialized")
	}
	if err := validateLayout(layout); err != nil {
		return CommitReceipt{}, err
	}
	if err := limits.validate(); err != nil {
		return CommitReceipt{}, err
	}
	snapshot, err := port.Snapshot(layout.Hash())
	if err != nil {
		return CommitReceipt{}, err
	}
	if snapshot.LayoutHash != layout.Hash() {
		return CommitReceipt{}, ErrLayoutMismatch
	}
	values := make(map[Key][]byte, len(snapshot.Entries))
	for index, entry := range snapshot.Entries {
		if index > 0 && slices.Compare(snapshot.Entries[index-1].Key[:], entry.Key[:]) >= 0 {
			return CommitReceipt{}, fmt.Errorf("persistent snapshot keys must be unique and sorted")
		}
		values[entry.Key] = slices.Clone(entry.Value)
	}
	transaction := &Transaction{
		store: persistenceCommitter{port: port}, layout: cloneLayout(layout),
		baseVersion: snapshot.Version, limits: limits, state: transactionOpen,
		snapshot: values, writes: make(map[Key]pendingWrite),
		borrowToken: &borrowToken{marker: 1}, borrowEpoch: make(map[Key]uint64),
	}
	return executeTransaction(transaction, body)
}
