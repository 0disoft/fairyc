package workspacecompiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
)

const aggregateCacheLibrary = `module answer;
pub struct Cell<T> { T value; }
pub resource Box<T> { Cell<T> cell; cleanup { let T observed = finish<T>(this.cell.value); } }
fn T finish<T>(T value) { return value; }
pub fn Box<T> make<T>(T value) { return Box<T> { .cell = Cell<T> { .value = value } }; }
pub fn T read<T>(ref Box<T> value) { return value.cell.value; }`

const aggregateCacheMain = `module main; import answer as a;
fn i32 main() { let a::Box<i32> value = a::make<i32>(42); return a::read<i32>(ref value); }`

func assertAggregateCacheOrigins(t *testing.T, result compiler.Result) {
	t.Helper()
	declarations := result.GenericProvenance.Declarations()
	origins := result.GenericProvenance.Specializations()
	if len(declarations) != 5 || len(origins) != 5 {
		t.Fatalf("aggregate provenance: %d/%d", len(declarations), len(origins))
	}
	byID := map[genericplan.DeclarationID]compiler.GenericDeclaration{}
	for _, declaration := range declarations {
		byID[declaration.ID] = declaration
	}
	for _, origin := range origins {
		declaration, found := byID[origin.Declaration]
		if !found {
			t.Fatal("origin has no current declaration")
		}
		qualified := hir.QualifiedModulePath(declaration.Package, declaration.Module) + "::" + origin.GeneratedName
		matches := 0
		switch declaration.Kind {
		case genericsource.OriginFunction:
			for _, function := range result.MIR.Functions {
				if function.Name == qualified {
					matches++
				}
			}
		case genericsource.OriginStruct:
			for _, value := range result.MIR.StructTypes {
				if value.Type == hir.ParseType(qualified) {
					matches++
				}
			}
		case genericsource.OriginResource:
			for _, value := range result.MIR.ResourceTypes {
				if value.Type == hir.ParseType(qualified) {
					matches++
				}
			}
		}
		if matches != 1 {
			t.Fatalf("cached %s origin did not join its current MIR: %s", declaration.Kind, qualified)
		}
	}
}

func TestGenericAggregateCacheColdWarmAndInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			session, root := genericWorkspaceSession(t, 32<<20)
			defer session.Close()
			if storage == StorageDisk {
				session = newGenericPersistentFixture(t, genericPersistentConfiguration(t, t.TempDir()))
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), aggregateCacheLibrary)
			writeFile(t, filepath.Join(root, "src", "main.fy"), aggregateCacheMain)
			workspace := loadCacheWorkspace(t, root)
			lower := session.lower
			calls := 0
			session.lower = func(a compiler.Analysis) (compiler.Result, error) { calls++; return lower(a) }
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || cold.Cache.Misses != 2 || calls != 1 {
				t.Fatalf("cold: %v %+v", err, cold.Cache)
			}
			warm, err := session.Check(workspace)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered || calls != 1 {
				t.Fatalf("warm: %v %+v", err, warm.Cache)
			}
			assertWorkspaceResult(t, workspace, warm.Compiler, 42)
			assertAggregateCacheOrigins(t, cold.Compiler)
			assertAggregateCacheOrigins(t, warm.Compiler)
			if !reflect.DeepEqual(cold.Compiler.GenericProvenance.Declarations(), warm.Compiler.GenericProvenance.Declarations()) {
				t.Fatal("warm cache lost current declarations")
			}
			assertGenericWorkspaceNative(t, workspace, warm)
			writeFile(t, filepath.Join(root, "src", "main.fy"), strings.Replace(aggregateCacheMain, "let a::Box<i32>", "let a::Box<bool> extra = a::make<bool>(true); let a::Box<i32>", 1))
			changed, err := session.Check(loadCacheWorkspace(t, root))
			if err != nil || !changed.Compiler.Verified || changed.Cache.Misses != 2 || !changed.Cache.Lowered || calls != 2 {
				t.Fatalf("new type argument reused stale artifact: %v %+v", err, changed.Cache)
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), strings.Replace(aggregateCacheLibrary, "return value;", "let T copy = value; return copy;", 1))
			changed, err = session.Check(loadCacheWorkspace(t, root))
			if err != nil || !changed.Compiler.Verified || changed.Cache.Misses != 2 || !changed.Cache.Lowered || calls != 3 {
				t.Fatalf("cleanup helper edit reused stale artifact: %v %+v", err, changed.Cache)
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), strings.Replace(aggregateCacheLibrary, "return value;", "return false;", 1))
			invalid, err := session.Check(loadCacheWorkspace(t, root))
			if err != nil || invalid.Compiler.Verified || invalid.Cache.Outcome != OutcomeNotEligible || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || len(invalid.Compiler.GenericProvenance.Declarations()) != 0 {
				t.Fatalf("invalid source reused cache: %v %+v", err, invalid.Cache)
			}
		})
	}
}

func TestGenericAggregatePersistentCrossProcess(t *testing.T) {
	root := writeCacheWorkspace(t, aggregateCacheLibrary)
	writeFile(t, filepath.Join(root, "src", "main.fy"), aggregateCacheMain)
	cacheRoot := t.TempDir()
	session := newGenericPersistentFixture(t, genericPersistentConfiguration(t, cacheRoot))
	cold, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || cold.Cache.Stored != 2 {
		t.Fatalf("cold: %v %+v", err, cold.Cache)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGenericAggregatePersistentChild$", "-test.count=1")
	command.Env = append(os.Environ(), "FAIRYC_AGGREGATE_WORKSPACE_CHILD="+root, "FAIRYC_AGGREGATE_WORKSPACE_CACHE="+cacheRoot)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "aggregate-workspace-disk-42") {
		t.Fatalf("fresh process: %v %s", err, output)
	}
}

func TestGenericAggregatePersistentChild(t *testing.T) {
	root := os.Getenv("FAIRYC_AGGREGATE_WORKSPACE_CHILD")
	if root == "" {
		t.Skip("subprocess consumer only")
	}
	session := newGenericPersistentFixture(t, genericPersistentConfiguration(t, os.Getenv("FAIRYC_AGGREGATE_WORKSPACE_CACHE")))
	session.lower = func(compiler.Analysis) (compiler.Result, error) {
		t.Fatal("warm aggregate child invoked lowering")
		return compiler.Result{}, nil
	}
	workspace := loadCacheWorkspace(t, root)
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 2 || result.Cache.Lowered {
		t.Fatalf("warm: %v %+v", err, result.Cache)
	}
	assertWorkspaceResult(t, workspace, result.Compiler, 42)
	assertAggregateCacheOrigins(t, result.Compiler)
	assertGenericWorkspaceNative(t, workspace, result)
	fmt.Println("aggregate-workspace-disk-42")
}
