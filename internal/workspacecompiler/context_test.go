package workspacecompiler

import (
	"context"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestSessionCancellationDoesNotPublishPartialArtifacts(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	workspace := loadCacheWorkspace(t, root)
	session, err := NewSession(Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lower := session.lower
	session.lower = func(ctx context.Context, analysis compiler.Analysis) (compiler.Result, error) {
		cancel()
		return lower(ctx, analysis)
	}
	result, err := session.CheckContext(ctx, workspace)
	if !errors.Is(err, context.Canceled) || result.Compiler.Verified || session.Stats().Store.Entries != 0 {
		t.Fatalf("canceled result=%+v stats=%+v err=%v", result, session.Stats(), err)
	}
	session.lower = lower
	result, err = session.CheckContext(context.Background(), workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Stored != 2 {
		t.Fatalf("retry result=%+v err=%v", result, err)
	}
	result, err = session.Check(workspace)
	if err != nil || result.Cache.Outcome != OutcomeAllHit {
		t.Fatalf("warm result=%+v err=%v", result.Cache, err)
	}
}

func TestGenericSessionCancellationDoesNotPublishPartialArtifacts(t *testing.T) {
	session, root := genericWorkspaceSession(t, 32<<20)
	defer session.Close()
	workspace := loadCacheWorkspace(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lower := session.lower
	session.lower = func(ctx context.Context, analysis compiler.Analysis, capabilities admission.Capabilities) (compiler.Result, error) {
		cancel()
		return lower(ctx, analysis, capabilities)
	}
	result, err := session.CheckContext(ctx, workspace)
	if !errors.Is(err, context.Canceled) || result.Compiler.Verified || session.Stats().Store.Entries != 0 {
		t.Fatalf("canceled generic result=%+v stats=%+v err=%v", result, session.Stats(), err)
	}
	session.lower = lower
	result, err = session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Stored != 2 {
		t.Fatalf("retry generic result=%+v err=%v", result, err)
	}
}

func TestSourceSessionAndLoaderHonorCanceledContext(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	workspace := loadCacheWorkspace(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if loaded, err := workspaceloader.LoadContext(ctx, root); !errors.Is(err, context.Canceled) || len(loaded.Packages) != 0 {
		t.Fatalf("loader: %+v %v", loaded, err)
	}
	if loaded, err := workspaceloader.LoadEditorOverlaysContext(ctx, root, nil); !errors.Is(err, context.Canceled) || len(loaded.Packages) != 0 {
		t.Fatalf("overlay: %+v %v", loaded, err)
	}
	if result, err := CheckContext(ctx, workspace); !errors.Is(err, context.Canceled) || result.Verified {
		t.Fatalf("workspace: %+v %v", result, err)
	}
	session, err := NewSourceSession(Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	before := session.Stats()
	if result, err := session.CheckContext(ctx, workspace); !errors.Is(err, context.Canceled) || result.Compiler.Verified || session.Stats() != before {
		t.Fatalf("source session: %+v %v", result, err)
	}
	if result, err := session.Check(workspace); err != nil || !result.Compiler.Verified {
		t.Fatalf("source retry: %+v %v", result, err)
	}
}
