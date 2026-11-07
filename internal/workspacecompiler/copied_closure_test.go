package workspacecompiler

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"path/filepath"
	"strings"
	"testing"
)

const closureDependency = `module answer;pub fn i32 value(){let i32 offset=40;let add=fn [i32 offset] i32(i32 n){return offset+n;};let i32[2] data=[1,2];var i32 result=0;for(let i32 item in data){result=add(item);}return result;}`

func TestCopiedClosureCache(t *testing.T) {
	for _, family := range []string{"ordinary", "generic"} {
		for _, storage := range []string{StorageMemory, StorageDisk} {
			t.Run(family+"/"+storage, func(t *testing.T) {
				text := closureDependency
				main := `module main;import answer as a;fn i32 main(){return a::value();}`
				if family == "generic" {
					text = strings.Replace(text, "pub fn i32 value()", "pub fn i32 value<T>()", 1)
					main = strings.Replace(main, "a::value()", "a::value<i32>()", 1)
				}
				root := writeCacheWorkspace(t, text)
				writeFile(t, filepath.Join(root, "src", "main.fy"), main)
				config := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
				if storage == StorageDisk {
					if family == "generic" {
						config = genericPersistentConfiguration(t, t.TempDir())
					} else {
						config = persistentTestConfiguration(t.TempDir(), "copied-closure-test")
					}
				}
				var check func(workspaceloader.Workspace) (Result, error)
				var closeSession func() error
				open := func() {
					if family == "ordinary" {
						s, err := NewSession(config)
						if err != nil {
							t.Fatal(err)
						}
						check = s.Check
						closeSession = s.Close
					} else {
						s, err := NewGenericSession(config, genericplan.DefaultPlanLimits())
						if err != nil {
							t.Fatal(err)
						}
						check = s.Check
						closeSession = s.Close
					}
				}
				open()
				defer func() { closeSession() }()
				verify := func(source string, want, hits int, all bool) {
					t.Helper()
					writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), source)
					w := loadCacheWorkspace(t, root)
					r, err := check(w)
					if err != nil || !r.Compiler.Verified || r.Cache.Hits != hits || (r.Cache.Outcome == OutcomeAllHit) != all {
						t.Fatalf("closure cache: %v %+v %+v", err, r.Cache, r.Compiler.Semantic)
					}
					assertWorkspaceResult(t, w, r.Compiler, int32(want))
					id, _ := w.RootPackageID()
					for _, p := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
						if _, err := cbackend.Emit(r.Compiler.MIR, hir.QualifiedModulePath(id, "main")+"::main", p); err != nil {
							t.Fatal(err)
						}
					}
				}
				verify(text, 42, 0, false)
				verify(text, 42, 2, true)
				if storage == StorageDisk {
					closeSession()
					open()
					verify(text, 42, 2, true)
				}
				changed := strings.Replace(text, "offset+n", "offset+n+1", 1)
				hits := 1
				if family == "generic" {
					hits = 0
				}
				verify(changed, 43, hits, false)
				verify(changed, 43, 2, true)
			})
		}
	}
}
