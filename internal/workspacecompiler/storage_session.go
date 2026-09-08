package workspacecompiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// StorageTransferSession is an explicit compiler-private, process-local lane.
// Stable SourceSession selection and persistent cache admission are unchanged.
// Every check analyzes current source before looking up source/interface/target/
// specialization-bound keys; only a complete verified reassembly skips lowering.
type StorageTransferSession struct {
	generic *GenericSession
}

func NewStorageTransferSession(configuration Configuration, limits genericplan.PlanLimits) (*StorageTransferSession, error) {
	session, err := newGenericSession(configuration, limits, true)
	if err != nil {
		return nil, err
	}
	return &StorageTransferSession{generic: session}, nil
}

func (session *StorageTransferSession) Check(workspace workspaceloader.Workspace) (Result, error) {
	if session == nil || session.generic == nil {
		return Result{}, fmt.Errorf("storage transfer session is not initialized")
	}
	return session.generic.Check(workspace)
}

func (session *StorageTransferSession) Stats() packageartifact.CacheStats {
	if session == nil {
		return packageartifact.CacheStats{}
	}
	return session.generic.Stats()
}
