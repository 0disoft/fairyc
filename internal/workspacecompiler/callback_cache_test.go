package workspacecompiler

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/hir"
)

// The callback target lives in another package, so cache restoration must
// preserve its import reference as well as the root's foreign declaration.
func TestBoundedCallbackCacheColdWarm(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			session, root := genericWorkspaceSession(t, 32<<20)
			defer session.Close()
			configuration := genericPersistentConfiguration(t, t.TempDir())
			if storage == StorageDisk {
				session = newGenericPersistentFixture(t, configuration)
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), `module answer; pub unsafe fn i32 handle(i32 value,ptr mut void data){return value+1;}`)
			writeFile(t, filepath.Join(root, "src", "main.fy"), `module main; import answer as a;
unsafe extern C fn i32 visit(i32 value,unsafe extern C fn i32(i32,ptr mut void) cb,ptr mut void data);
fn i32 main(){return unsafe{visit(41,a::handle,null)};}`)
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
