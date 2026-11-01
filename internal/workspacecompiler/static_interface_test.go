package workspacecompiler

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"path/filepath"
	"strings"
	"testing"
)

const interfaceDependency = `module answer;
@interface pub fn T transform<T>(T value){}
@implements(transform,i32) fn i32 twice(i32 value){return value*2;}
pub fn T algorithm<T>(T value){return transform<T>(value);}
fn i32 unused(){return 1;}`

func TestStaticInterfaceCacheInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, interfaceDependency)
			writeFile(t, filepath.Join(root, "src", "main.fy"), `module main;import answer as a;fn i32 main(){return a::algorithm<i32>(21);}`)
			config := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
			if storage == StorageDisk {
				config = genericPersistentConfiguration(t, t.TempDir())
			}
			s, err := NewGenericSession(config, genericplan.DefaultPlanLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { s.Close() }()
			check := func(text string, want, hits int, all bool) {
				t.Helper()
				writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), text)
				w := loadCacheWorkspace(t, root)
				r, err := s.Check(w)
				if err != nil || !r.Compiler.Verified || r.Cache.Hits != hits || (r.Cache.Outcome == OutcomeAllHit) != all {
					t.Fatalf("cache: %v %+v %+v", err, r.Cache, r.Compiler.Semantic.Diagnostics)
				}
				assertWorkspaceResult(t, w, r.Compiler, int32(want))
				id, _ := w.RootPackageID()
				for _, p := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(r.Compiler.MIR, hir.QualifiedModulePath(id, "main")+"::main", p); err != nil {
						t.Fatal(err)
					}
				}
			}
			check(interfaceDependency, 42, 0, false)
			check(interfaceDependency, 42, 2, true)
			if storage == StorageDisk {
				s.Close()
				s, err = NewGenericSession(config, genericplan.DefaultPlanLimits())
				if err != nil {
					t.Fatal(err)
				}
				check(interfaceDependency, 42, 2, true)
			}
			changed := strings.Replace(interfaceDependency, "value*2", "value*3", 1)
			check(changed, 63, 0, false)
			check(changed, 63, 2, true)
			unrelated := strings.Replace(changed, "return 1;", "return 2;", 1)
			check(unrelated, 63, 1, false)
			check(unrelated, 63, 2, true)
			missing := strings.Replace(unrelated, "@implements(transform,i32)", "", 1)
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), missing)
			bad, err := s.Check(loadCacheWorkspace(t, root))
			if err != nil || bad.Compiler.Verified || bad.Cache.Outcome != OutcomeNotEligible {
				t.Fatalf("missing provider reused: %v %+v", err, bad.Cache)
			}
		})
	}
}
