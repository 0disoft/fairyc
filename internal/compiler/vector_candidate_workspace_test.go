package compiler_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPrivateVectorSDKWorkspaceCache(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "disk"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVectorCacheFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.vector-candidate.client"
name = "client"
version = "1.0.0"
language = "fairyc-language/14"
source = "src"
`)
			client := vectorCandidateProgram()
			writeVectorCacheFile(t, root, "src/main.fy", client)
			load := func() workspaceloader.Workspace {
				t.Helper()
				workspace, err := workspaceloader.LoadVectorCandidate(root)
				if err != nil {
					t.Fatal(err)
				}
				return workspace
			}
			workspace := load()
			sdk, exists := workspace.Package(sdkstdlib.PackagePath)
			if !exists || sdk.SDKIdentity != sdkstdlib.VectorCandidateIdentity || len(workspace.Graph.Nodes) != 2 {
				t.Fatal("candidate graph is not the root and exact SDK")
			}
			configuration := vectorCacheConfiguration(8)
			if persistent {
				configuration = vectorPersistentConfiguration(t, t.TempDir())
			}
			session, err := workspacecompiler.NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || !cold.Cache.Lowered || cold.Cache.Stored != 2 {
				t.Fatalf("cold candidate: %v %+v %+v", err, cold.Cache, cold.Compiler.Semantic.Diagnostics)
			}
			warm, err := session.Check(load())
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
				t.Fatalf("warm candidate: %v %+v", err, warm.Cache)
			}
			rootID, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			runVectorStorageMIRNativeLane(t, warm.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", vectorCandidateProbe, true)
			ordinary, err := workspacecompiler.NewSourceSession(vectorCacheConfiguration(8))
			if err != nil {
				t.Fatal(err)
			}
			defer ordinary.Close()
			denied, err := ordinary.Check(workspace)
			if err == nil && denied.Compiler.Verified {
				t.Fatal("ordinary workspace session admitted private SDK")
			}
			changed := strings.Replace(client, "return 41;", "return 40 + 1;", 1)
			writeVectorCacheFile(t, root, "src/main.fy", changed)
			updated, err := session.Check(load())
			if err != nil || !updated.Compiler.Verified || !updated.Cache.Lowered || updated.Cache.Misses == 0 {
				t.Fatalf("changed candidate: %v %+v", err, updated.Cache)
			}
			stats := func() any {
				t.Helper()
				if !persistent {
					return session.Stats()
				}
				value, err := session.PersistentStats()
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			before := stats()
			writeVectorCacheFile(t, root, "src/main.fy", strings.Replace(changed, "return 40 + 1;", "return true;", 1))
			invalid, err := session.Check(load())
			if invalid.Compiler.Verified || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
				t.Fatalf("invalid source reached candidate cache: %v %+v", err, invalid.Cache)
			}
		})
	}
}
