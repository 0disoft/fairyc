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

type sourceCacheSession interface {
	Check(workspaceloader.Workspace) (Result, error)
	CheckContext(context.Context, workspaceloader.Workspace) (Result, error)
	Stats() packageartifact.CacheStats
	PersistentStats() (packageartifact.StableCacheStats, error)
	Close() error
}

type sourceMode uint8

const (
	sourceOrdinary sourceMode = iota
	sourceGeneric
	sourceSDKStorage
)

// SourceSession owns one active cache family. A source-mode transition releases
// the old session before constructing the next, rather than keeping two LRUs.
// Disk bounds and statistics apply to the selected format namespace, not every
// historical format directory below the configured root.
type SourceSession struct {
	configuration Configuration
	active        sourceCacheSession
	mode          sourceMode
	closed        bool
}

func NewSourceSession(configuration Configuration) (*SourceSession, error) {
	if configuration.Persistent != nil {
		persistent := *configuration.Persistent
		configuration.Persistent = &persistent
	}
	active, err := NewSession(configuration)
	if err != nil {
		return nil, err
	}
	return &SourceSession{configuration: configuration, active: active}, nil
}

func (session *SourceSession) Check(workspace workspaceloader.Workspace) (Result, error) {
	return session.CheckContext(context.Background(), workspace)
}

func (session *SourceSession) CheckContext(ctx context.Context, workspace workspaceloader.Workspace) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: canceled}
		}
	}()
	if session == nil || session.closed {
		return Result{}, fmt.Errorf("source workspace session is closed")
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return Result{}, err
	}
	owners, authorityErr := workspace.SDKStorageOwners()
	if authorityErr != nil {
		return Result{}, authorityErr
	}
	sdkStorage := len(owners) != 0
	if hasGenericAggregates(input) && !sdkStorage {
		compiled, err := compiler.CheckPackagesContext(ctx, input)
		storage := StorageMemory
		if session.configuration.Persistent != nil {
			storage = StorageDisk
		}
		return Result{Compiler: compiled, Cache: Report{Outcome: OutcomeNotEligible, Storage: storage}}, err
	}
	mode := sourceOrdinary
	if sdkStorage {
		mode = sourceSDKStorage
	} else if hasGenericDeclarations(input) {
		mode = sourceGeneric
	}
	if session.active == nil || session.mode != mode {
		if session.active != nil {
			old := session.active
			session.active = nil
			if err := old.Close(); err != nil {
				return Result{}, err
			}
		}
		if mode == sourceOrdinary {
			session.active, err = NewSession(session.configuration)
		} else {
			capabilities := admission.Capabilities{}
			if mode == sourceSDKStorage {
				capabilities = sdkStorageCapabilities()
			}
			session.active, err = NewGenericSessionWithCapabilities(session.configuration, genericplan.DefaultPlanLimits(), capabilities)
		}
		if err != nil {
			session.active = nil
			return Result{}, err
		}
		session.mode = mode
	}
	return session.active.CheckContext(ctx, workspace)
}

func (session *SourceSession) Stats() packageartifact.CacheStats {
	if session == nil || session.active == nil {
		return packageartifact.CacheStats{}
	}
	return session.active.Stats()
}

func (session *SourceSession) PersistentStats() (packageartifact.StableCacheStats, error) {
	if session == nil || session.active == nil {
		return packageartifact.StableCacheStats{}, fmt.Errorf("source workspace session is inactive")
	}
	return session.active.PersistentStats()
}

func (session *SourceSession) Close() error {
	if session == nil || session.closed {
		return nil
	}
	session.closed = true
	active := session.active
	session.active = nil
	if active == nil {
		return nil
	}
	return active.Close()
}
