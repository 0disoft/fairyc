package workspacecompiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func stateGraphCacheWorkspace(t *testing.T, fixture string) (string, string) {
	t.Helper()
	data, err := os.ReadFile("../../examples/contract_state_graph/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fairy.toml"), fmt.Sprintf(`schema = "fairyc.manifest/v1"
[package]
uid = "test.state-graph-cache"
name = "state-graph-cache"
version = "1.0.0"
language = %q
source = "src"
`, sdkstdlib.LanguageRevision))
	writeFile(t, filepath.Join(root, "src", "transaction.fy"), string(data))
	return root, string(data)
}

func graphCacheConfiguration(t *testing.T, storage string) Configuration {
	t.Helper()
	if storage == StorageDisk {
		return persistentTestConfiguration(t.TempDir(), "test-state-graph-build-a")
	}
	return Configuration{Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}}
}

func currentGraphTransport(t *testing.T, workspace workspaceloader.Workspace, compiled compiler.Result) contractstatenative.GraphTransport {
	t.Helper()
	if !compiled.Verified || len(compiled.StateTransactions) != 1 {
		t.Fatal("cache did not publish one verified graph")
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	owner := compiled.StateTransactions[0]
	handoff, err := contractstatenative.PrepareCompiled(compiled, owner.Package, owner.Function, coordinates[owner.Package])
	if err != nil {
		t.Fatal("cached ordinary/state owners did not align", err)
	}
	prepared, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func TestStateGraphSessionMemoryAndPersistent(t *testing.T) {
	for _, fixture := range []string{"transaction.fy", "helpers/transaction.fy", "borrows/transaction.fy", "dynamic/transaction.fy"} {
		for _, storage := range []string{StorageMemory, StorageDisk} {
			t.Run(fixture+"/"+storage, func(t *testing.T) {
				root, _ := stateGraphCacheWorkspace(t, fixture)
				configuration := graphCacheConfiguration(t, storage)
				session, err := NewExperimentalStateGraphSession(configuration)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = session.Close() })
				workspace := loadCacheWorkspace(t, root)
				cold, err := session.Check(workspace)
				if err != nil || !cold.Compiler.Verified || cold.Cache.Outcome != OutcomeFallback || !cold.Cache.Lowered || cold.Cache.Misses != 1 || cold.Cache.Stored != 1 {
					t.Fatalf("state cold %+v: %v", cold.Cache, err)
				}
				coldTransport := currentGraphTransport(t, workspace, cold.Compiler)
				coldC, err := coldTransport.Source()
				if err != nil {
					t.Fatal(err)
				}
				if storage == StorageDisk {
					if err := session.Close(); err != nil {
						t.Fatal(err)
					}
					session, err = NewExperimentalStateGraphSession(configuration)
					if err != nil {
						t.Fatal(err)
					}
				}
				session.lower = func(context.Context, compiler.Analysis) (compiler.Result, error) {
					t.Fatal("complete state cache hit lowered ordinary MIR")
					return compiler.Result{}, nil
				}
				workspace = loadCacheWorkspace(t, root)
				warm, err := session.Check(workspace)
				if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered || warm.Cache.Hits != 1 || warm.Cache.Stored != 0 {
					t.Fatalf("state warm %+v: %v", warm.Cache, err)
				}
				warmTransport := currentGraphTransport(t, workspace, warm.Compiler)
				warmC, err := warmTransport.Source()
				if err != nil || coldC != warmC || coldTransport.BodyHash() != warmTransport.BodyHash() || coldTransport.ProgramHash() != warmTransport.ProgramHash() {
					t.Fatal("restoration changed exact current-source graph C/identities", err)
				}
			})
		}
	}
}

func TestStateGraphSessionSourceInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root, original := stateGraphCacheWorkspace(t, "dynamic/transaction.fy")
			configuration := graphCacheConfiguration(t, storage)
			session, err := NewExperimentalStateGraphSession(configuration)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = session.Close() })
			workspace := loadCacheWorkspace(t, root)
			first, err := session.Check(workspace)
			if err != nil || !first.Compiler.Verified {
				t.Fatal(err)
			}
			firstIdentity := currentGraphTransport(t, workspace, first.Compiler).BodyHash()
			for _, edit := range []struct {
				name, source string
				bodyChanged  bool
			}{
				{"comment", original + "\n// exact source change\n", false},
				{"helper-body", strings.Replace(original, `b"\x00\xffAB\x1a\r\n"`, `b"\x00\xffCD\x1a\r\n"`, 1), true},
			} {
				t.Run(edit.name, func(t *testing.T) {
					writeFile(t, filepath.Join(root, "src", "transaction.fy"), edit.source)
					workspace = loadCacheWorkspace(t, root)
					got, err := session.Check(workspace)
					if err != nil || !got.Compiler.Verified || got.Cache.Misses != 1 || !got.Cache.Lowered || got.Cache.Stored != 1 {
						t.Fatalf("edited source reused stale executable %+v: %v", got.Cache, err)
					}
					identity := currentGraphTransport(t, workspace, got.Compiler).BodyHash()
					if (identity != firstIdentity) != edit.bodyChanged {
						t.Fatal("source bytes and state body identities were conflated")
					}
				})
			}
			stats := func() any {
				if storage == StorageDisk {
					value, err := session.PersistentStats()
					if err != nil {
						t.Fatal(err)
					}
					return value
				}
				return session.Stats()
			}
			before := stats()
			writeFile(t, filepath.Join(root, "src", "transaction.fy"), strings.Replace(original, "@effects(storage_write)", "@effects(storage_read)", 1))
			invalid, err := session.Check(loadCacheWorkspace(t, root))
			if err == nil && invalid.Compiler.Verified || len(invalid.Compiler.MIR.Functions)+len(invalid.Compiler.StateTransactions) != 0 || invalid.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
				t.Fatal("failed current-source admission used/published an artifact", err)
			}
		})
	}
}

func TestStateGraphSessionAdmissionAndCancellation(t *testing.T) {
	root, _ := stateGraphCacheWorkspace(t, "dynamic/transaction.fy")
	workspace := loadCacheWorkspace(t, root)
	configuration := graphCacheConfiguration(t, StorageDisk)
	graph, err := NewExperimentalStateGraphSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = graph.Close() })
	ordinary, err := NewSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ordinary.Close() })
	if graph.compilerBuild == ordinary.compilerBuild || !strings.HasSuffix(graph.compilerBuild, ";"+stateGraphSourceAdmission) {
		t.Fatal("state source admission missing from artifact key")
	}
	if got, err := graph.Check(workspace); err != nil || !got.Compiler.Verified {
		t.Fatal("state cold", err)
	}
	if got, err := ordinary.Check(workspace); err == nil && got.Compiler.Verified || got.Cache.Hits != 0 || len(got.Compiler.StateTransactions) != 0 {
		t.Fatal("ordinary constructor accepted experimental state source", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before, err := graph.PersistentStats()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := graph.CheckContext(ctx, workspace); !errors.Is(err, context.Canceled) || got.Compiler.Verified || got.Cache.Hits != 0 {
		t.Fatal("canceled request reused state artifact", err)
	}
	after, err := graph.PersistentStats()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("cancellation touched persistent cache", err)
	}
	writeFile(t, filepath.Join(root, "src", "extra.fy"), "module extra; fn void extra() {}\n")
	if got, err := graph.Check(loadCacheWorkspace(t, root)); err == nil || got.Compiler.Verified || got.Cache.Hits != 0 {
		t.Fatal("unadmitted multi-file state source reused an artifact", err)
	}
}

func TestStateGraphSessionRejectedRebindFallsBack(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root, _ := stateGraphCacheWorkspace(t, "helpers/transaction.fy")
			session, err := NewExperimentalStateGraphSession(graphCacheConfiguration(t, storage))
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			workspace := loadCacheWorkspace(t, root)
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified {
				t.Fatal(err)
			}
			identity := currentGraphTransport(t, workspace, cold.Compiler).ProgramHash()
			restore := session.restore
			session.restore = func(compiled compiler.Result) (compiler.Result, error) {
				compiled.MIR.Functions[0].Name = "absent::owner"
				return restore(compiled)
			}
			repaired, err := session.Check(workspace)
			if err != nil || !repaired.Compiler.Verified || !repaired.Cache.ReassemblyFallback || !repaired.Cache.Lowered || repaired.Cache.Outcome != OutcomeFallback || repaired.Cache.Stored != 1 || currentGraphTransport(t, workspace, repaired.Compiler).ProgramHash() != identity {
				t.Fatalf("rejected state rebind did not heal by cold source lowering %+v: %v", repaired.Cache, err)
			}
			session.restore = restore
			warm, err := session.Check(workspace)
			if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered || currentGraphTransport(t, workspace, warm.Compiler).ProgramHash() != identity {
				t.Fatal("repaired state artifact did not restore", err)
			}
		})
	}
}
