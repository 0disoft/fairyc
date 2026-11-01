package workspacecompiler

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"path/filepath"
	"testing"
)

func TestBroaderGenericsScalarCache(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
			t.Run(storage+"/"+scalar, func(t *testing.T) {
				root := writeCacheWorkspace(t, `module answer;pub fn T id<T>(T value){return value;}`)
				writeFile(t, filepath.Join(root, "src", "main.fy"), fmt.Sprintf(`module main;import answer as a;@repr(C) struct P{%s value;}unsafe extern C fn P seed();fn i32 main(){let P p=unsafe{seed()};discard a::id<%s>(p.value);return 42;}`, scalar, scalar))
				config := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
				if storage == StorageDisk {
					config = genericPersistentConfiguration(t, t.TempDir())
				}
				s, err := NewGenericSession(config, genericplan.DefaultPlanLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer func() { s.Close() }()
				workspace := loadCacheWorkspace(t, root)
				cold, err := s.Check(workspace)
				if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 {
					t.Fatalf("cold: %v %+v", err, cold)
				}
				if storage == StorageDisk {
					s.Close()
					s, err = NewGenericSession(config, genericplan.DefaultPlanLimits())
					if err != nil {
						t.Fatal(err)
					}
				}
				warm, err := s.Check(workspace)
				if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered {
					t.Fatalf("warm: %v %+v", err, warm)
				}
				id, err := workspace.RootPackageID()
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(warm.Compiler.MIR, hir.QualifiedModulePath(id, "main")+"::main", p); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
