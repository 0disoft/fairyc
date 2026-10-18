package workspacecompiler

import (
	"context"
	"fmt"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// StorageTransferSession owns compiler-private memory or exact-build persistent
// artifacts. The raw constructor is explicit; SourceSession selects its SDK-only
// constructor for authenticated vector bundles.
// Every check analyzes current source before looking up source/interface/target/
// specialization-bound keys; only a complete verified reassembly skips lowering.
type StorageTransferSession struct {
	generic *GenericSession
}

func NewStorageTransferSession(configuration Configuration, limits genericplan.PlanLimits) (*StorageTransferSession, error) {
	session, err := NewGenericSessionWithCapabilities(configuration, limits, admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments})
	if err != nil {
		return nil, err
	}
	return &StorageTransferSession{generic: session}, nil
}

// NewSDKStorageSession selects private operations only for the exact embedded
// core-plus-vector SDK bundle. It never exposes the unrestricted test lane.
func NewSDKStorageSession(configuration Configuration, limits genericplan.PlanLimits) (*StorageTransferSession, error) {
	session, err := NewGenericSessionWithCapabilities(configuration, limits, sdkStorageCapabilities())
	if err != nil {
		return nil, err
	}
	return &StorageTransferSession{generic: session}, nil
}

func (session *StorageTransferSession) Check(workspace workspaceloader.Workspace) (Result, error) {
	return session.CheckContext(context.Background(), workspace)
}

func (session *StorageTransferSession) CheckContext(ctx context.Context, workspace workspaceloader.Workspace) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: canceled}
		}
	}()
	if session == nil || session.generic == nil {
		return Result{}, fmt.Errorf("storage transfer session is not initialized")
	}
	return session.generic.CheckContext(ctx, workspace)
}

func (session *StorageTransferSession) Stats() packageartifact.CacheStats {
	if session == nil {
		return packageartifact.CacheStats{}
	}
	return session.generic.Stats()
}

func (session *StorageTransferSession) PersistentStats() (packageartifact.StableCacheStats, error) {
	if session == nil || session.generic == nil {
		return packageartifact.StableCacheStats{}, fmt.Errorf("storage transfer session is not initialized")
	}
	return session.generic.PersistentStats()
}

func (session *StorageTransferSession) Close() error {
	if session == nil || session.generic == nil {
		return nil
	}
	return session.generic.Close()
}
