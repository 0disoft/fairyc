package contractstatenative

import "github.com/0disoft/fairyc/internal/contractstate"

// Both portable wire tests and Windows native probes observe publication writes.
// Keep their Go oracle independent of the native probe's host file selection.
type overlayOraclePort struct {
	contractstate.PersistencePort
	writes []contractstate.PersistentWrite
}

func (port *overlayOraclePort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.writes = writes
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}
