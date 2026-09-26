package compiler_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
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
language = "fairyc-language/20"
source = "src"
`)
			client := vectorCandidateProgram()
			client = strings.Replace(client, "    let bytes::Buffer last =", `    let ref bytes::Buffer first = v::at<bytes::Buffer>(ref storage, 0);
    assert(valid(first));
    let bytes::Buffer last =`, 1)
			writeVectorCacheFile(t, root, "src/main.fy", client)
			load := func() workspaceloader.Workspace {
				t.Helper()
				workspace, err := workspaceloader.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				return workspace
			}
			workspace := load()
			sdk, exists := workspace.Package(sdkstdlib.PackagePath)
			if !exists || sdk.SDKIdentity != sdkstdlib.SDKIdentity || len(workspace.Graph.Nodes) != 2 {
				t.Fatal("candidate graph is not the root and exact SDK")
			}
			configuration := vectorCacheConfiguration(8)
			if persistent {
				configuration = vectorPersistentConfiguration(t, t.TempDir())
			}
			session, err := workspacecompiler.NewSourceSession(configuration)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || !cold.Cache.Lowered || cold.Cache.Stored != 2 {
				t.Fatalf("cold candidate: %v %+v %+v", err, cold.Cache, cold.Compiler.Semantic.Diagnostics)
			}
			if persistent {
				session.Close()
				session, err = workspacecompiler.NewSourceSession(configuration)
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
			}
			warm, err := session.Check(load())
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
				t.Fatalf("warm candidate: %v %+v", err, warm.Cache)
			}
			borrowCount := 0
			for _, function := range warm.Compiler.MIR.Functions {
				for _, block := range function.Blocks {
					for _, operation := range block.Operations {
						if operation.Kind == mir.OperationStorageBorrow {
							borrowCount++
						}
					}
				}
			}
			if borrowCount != 1 {
				t.Fatalf("cached element borrow missing: %d", borrowCount)
			}
			rootID, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			runVectorStorageMIRNativeLane(t, warm.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", vectorCandidateProbe, true)
			uncached, err := workspacecompiler.Check(workspace)
			if err != nil || !uncached.Verified {
				t.Fatalf("uncached candidate selection: %v", err)
			}
			if workspace.UsesVectorCandidate() {
				t.Fatal("default loader selected the private candidate identity")
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
			conflict := strings.Replace(changed, "assert(valid(first));", "let usize removed = v::clear<bytes::Buffer>(ref mut storage); assert(valid(first));", 1)
			writeVectorCacheFile(t, root, "src/main.fy", conflict)
			invalid, err = session.Check(load())
			if invalid.Compiler.Verified || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
				t.Fatalf("borrow conflict reached candidate cache: %v %+v", err, invalid.Cache)
			}
			writeVectorCacheFile(t, root, "src/main.fy", strings.Replace(changed, "return 40 + 1;", "return unsafe { storage_take<i32>(null) };", 1))
			invalid, err = session.Check(load())
			if invalid.Compiler.Verified || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
				t.Fatalf("consumer intrinsic reached candidate cache: %v %+v", err, invalid.Cache)
			}
			writeVectorCacheFile(t, root, "src/main.fy", changed)
			forged := load()
			for i := range forged.Packages {
				if forged.Packages[i].Path == sdkstdlib.PackagePath {
					forged.Packages[i].Sources[0].Module.Path = "forged"
				}
			}
			invalid, err = session.Check(forged)
			if err == nil || invalid.Compiler.Verified || !reflect.DeepEqual(before, stats()) {
				t.Fatalf("forged SDK reached candidate cache: %v %+v", err, invalid.Cache)
			}
			writeVectorCacheFile(t, root, "src/main.fy", "module main; fn i32 main() { return 42; }")
			ordinary, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			switched, err := session.Check(ordinary)
			if err != nil || !switched.Compiler.Verified || !switched.Cache.Lowered {
				t.Fatalf("candidate to ordinary transition: %v %+v", err, switched.Cache)
			}
			writeVectorCacheFile(t, root, "src/main.fy", changed)
			returned, err := session.Check(load())
			if err != nil || !returned.Compiler.Verified {
				t.Fatalf("ordinary to candidate transition: %v %+v", err, returned.Cache)
			}
			if persistent && (returned.Cache.Hits != 2 || returned.Cache.Lowered) {
				t.Fatalf("disk candidate lost exact entries: %+v", returned.Cache)
			}
			if !persistent && (!returned.Cache.Lowered || returned.Cache.Stored != 2) {
				t.Fatalf("memory candidate retained a second LRU: %+v", returned.Cache)
			}
		})
	}
}

func TestPrivateVectorSDKWorkspaceSourceAdmission(t *testing.T) {
	root := t.TempDir()
	writeVectorCacheFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.sdk.source-admission"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	const client = `module main;
import std::vec as vec;
import std::bytes as bytes;
fn T identity<T>(T value) { return value; }
fn i32 main() {
    var vec::Vec<bytes::Buffer> items = vec::empty<bytes::Buffer>();
    assert(vec::len<bytes::Buffer>(ref items) == 0);
    return identity<i32>(41);
}
`
	load := func(text string) workspaceloader.Workspace {
		t.Helper()
		writeVectorCacheFile(t, root, "src/main.fy", text)
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return workspace
	}
	for _, persistent := range []bool{false, true} {
		configuration := vectorCacheConfiguration(8)
		if persistent {
			configuration = vectorPersistentConfiguration(t, t.TempDir())
		}
		session, err := workspacecompiler.NewSourceSession(configuration)
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		baseline, err := session.Check(load(client))
		if err != nil || !baseline.Compiler.Verified {
			t.Fatalf("scalar consumer and SDK nominal template: %v %+v", err, baseline.Compiler.Semantic)
		}
		stats := func() any {
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
		for _, extra := range []string{
			"struct Local<T> { T value; }",
			"resource Local<T> { T value; cleanup {} }",
			"fn i32 use_nominal() { let bytes::Buffer value = identity<bytes::Buffer>(bytes::empty()); return 0; }",
		} {
			workspace := load(client + extra)
			checked, err := workspacecompiler.Check(workspace)
			if err == nil && checked.Verified {
				t.Fatalf("uncached consumer permission widened: %s", extra)
			}
			cached, err := session.Check(workspace)
			if cached.Compiler.Verified || cached.Cache.Hits != 0 || cached.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
				t.Fatalf("denied consumer touched cache: %s %v %+v", extra, err, cached.Cache)
			}
		}
	}
}
