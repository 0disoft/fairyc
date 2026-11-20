package contractstate

import "github.com/0disoft/fairyc/internal/durablefile"

const localBackupName = "state-backup-v1.bin"

// BackupTo captures one validated committed record under the store lock.
// Publication to an existing trusted directory is exclusive and occurs after
// releasing the store lock.
func (store *LocalPersistence) BackupTo(hash LayoutHash, directory string) error {
	var record []byte
	if err := store.withLock(func() error {
		current, encoded, err := store.readRecord()
		if err != nil {
			return err
		}
		if current.layout.Hash() != hash {
			return ErrLayoutMismatch
		}
		record = encoded
		return nil
	}); err != nil {
		return err
	}
	root, err := openLocalRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	temporary, err := stageLocalRecord(root, record)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	return root.Link(temporary, localBackupName)
}

// RestoreLocalPersistence validates a backup in an existing trusted directory
// before creating a new store. It preserves the exact record and refuses to
// replace either authority file.
func RestoreLocalPersistence(directory, backupDirectory string, expected Layout, limits LocalPersistenceLimits) (*LocalPersistence, error) {
	if err := validateLayout(expected); err != nil {
		return nil, err
	}
	if err := limits.validate(); err != nil {
		return nil, err
	}
	root, err := openLocalRoot(backupDirectory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := durablefile.OpenRegular(root, localBackupName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	record, err := readLocalStateBytes(file, limits)
	if err != nil {
		return nil, err
	}
	state, err := decodeLocalState(record, limits)
	if err != nil {
		return nil, err
	}
	if state.layout.Hash() != expected.Hash() {
		return nil, ErrLayoutMismatch
	}
	return openLocalPersistence(directory, expected, limits, record)
}
