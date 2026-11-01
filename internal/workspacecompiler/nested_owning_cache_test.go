package workspacecompiler

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/hir"
)

// Resource definitions and recursive tag ownership must survive fresh package bindings.
func TestNestedOwningResourceCacheColdWarm(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			session, root := genericWorkspaceSession(t, 32<<20)
			defer session.Close()
			configuration := genericPersistentConfiguration(t, t.TempDir())
			if storage == StorageDisk {
				session = newGenericPersistentFixture(t, configuration)
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), `module answer;pub resource Leaf{i32 value;cleanup{}}pub fn Option<Option<Leaf>> make(){return .some(.some(Leaf{.value=42}));}`)
			writeFile(t, filepath.Join(root, "src", "main.fy"), `module main;import answer as a;fn i32 main(){let Option<Option<a::Leaf>> value=a::make();return 42;}`)
			workspace := loadCacheWorkspace(t, root)
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("cold: %v %+v %+v", err, cold.Cache, cold.Compiler.Semantic)
			}
			if storage == StorageDisk {
				session = newGenericPersistentFixture(t, configuration)
			}
			warm, err := session.Check(workspace)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
				t.Fatalf("warm: %v %+v", err, warm.Cache)
			}
			rootPackage, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = cbackend.Emit(warm.Compiler.MIR, hir.QualifiedModulePath(rootPackage, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
