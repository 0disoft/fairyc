package workspacecompiler

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const inlineDependency = `module answer;@inline pub fn i32 value(){return 42;}fn i32 private_helper(){return 1;}`

func TestCrossPackageInlineCacheInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, inlineDependency)
			config := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
			if storage == StorageDisk {
				config = persistentTestConfiguration(t.TempDir(), "inline-test-build")
			}
			session, err := NewSession(config)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { session.Close() }()
			check := func(text string, want, hits int, allHit bool) {
				t.Helper()
				writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), text)
				workspace := loadCacheWorkspace(t, root)
				r, err := session.Check(workspace)
				if err != nil || !r.Compiler.Verified || r.Cache.Hits != hits || (r.Cache.Outcome == OutcomeAllHit) != allHit {
					t.Fatalf("cache: %v %+v %+v", err, r.Cache, r.Compiler.Semantic.Diagnostics)
				}
				id, err := workspace.RootPackageID()
				if err != nil {
					t.Fatal(err)
				}
				entry := hir.QualifiedModulePath(id, "main") + "::main"
				out, err := interpreter.Execute(r.Compiler.MIR, entry)
				if err != nil || out.Value != int32(want) {
					t.Fatalf("result: %+v %v", out, err)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					unit, err := cbackend.Emit(r.Compiler.MIR, entry, profile)
					if err != nil || !strings.Contains(unit.Source, "static inline int32_t") {
						if !strings.Contains(text, "@inline") {
							continue
						}
						t.Fatal("inline body lost", err)
					}
				}
			}
			check(inlineDependency, 42, 0, false)
			check(inlineDependency, 42, 2, true)
			if storage == StorageDisk {
				session.Close()
				session, err = NewSession(config)
				if err != nil {
					t.Fatal(err)
				}
			}
			check(inlineDependency, 42, 2, true)
			changed := strings.Replace(inlineDependency, "return 42;", "return 43;", 1)
			check(changed, 43, 0, false)
			check(changed, 43, 2, true)
			private := strings.Replace(changed, "return 1;", "return 2;", 1)
			check(private, 43, 1, false)
			check(private, 43, 2, true)
			formatted := strings.Replace(private, "return 43;", "/*shifted*/ return 43;", 1)
			check(formatted, 43, 1, false)
			check(formatted, 43, 2, true)
			removed := strings.Replace(formatted, "@inline ", "", 1)
			check(removed, 43, 0, false)
			check(removed, 43, 2, true)
		})
	}
}

func TestInlineCacheRejectsChangedOrMissingMarker(t *testing.T) {
	root := writeCacheWorkspace(t, inlineDependency)
	session, err := NewSession(Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	r, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || !r.Compiler.Verified {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "changed", "extra"} {
		t.Run(mode, func(t *testing.T) {
			program := mir.Clone(r.Compiler.MIR)
			for i := range program.Functions {
				fn := &program.Functions[i]
				if mode == "extra" && fn.InlineBody == "" {
					fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("1", 64)
					break
				}
				if fn.InlineBody != "" {
					if mode == "missing" {
						fn.InlineBody = ""
					} else if mode == "changed" {
						fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("2", 64)
					}
					break
				}
			}
			if inlinebody.VerifyProgram(r.Compiler.Semantic.HIR, program) == nil {
				t.Fatal("mismatched cached inline marker accepted")
			}
		})
	}
}

// Inject structurally valid stale markers through the lower seam to seed the real
// stores. The next Check must reject their all-hit reassembly, lower and heal.
func TestInlineCacheRejectsAndHealsStaleMarkers(t *testing.T) {
	for _, family := range []string{"ordinary", "generic"} {
		for _, storage := range []string{StorageMemory, StorageDisk} {
			for _, mode := range []string{"missing", "changed", "extra"} {
				t.Run(family+"/"+storage+"/"+mode, func(t *testing.T) {
					root := writeCacheWorkspace(t, inlineDependency)
					if family == "generic" {
						writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), inlineDependency+`pub fn T echo<T>(T input){return input;}`)
						writeFile(t, filepath.Join(root, "src", "main.fy"), `module main;import answer as answer;fn i32 main(){return answer::echo<i32>(answer::value());}`)
					}
					config := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
					if storage == StorageDisk {
						if family == "generic" {
							config = genericPersistentConfiguration(t, t.TempDir())
						} else {
							config = persistentTestConfiguration(t.TempDir(), "inline-marker-test-build")
						}
					}
					calls := 0
					mutate := func(r compiler.Result) compiler.Result {
						calls++
						if calls != 1 {
							return r
						}
						for i := range r.MIR.Functions {
							fn := &r.MIR.Functions[i]
							if mode == "extra" {
								if !strings.HasSuffix(fn.Name, "::private_helper") {
									continue
								}
								fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("1", 64)
								return r
							}
							if fn.InlineBody == "" {
								continue
							}
							if mode == "missing" {
								fn.InlineBody = ""
							} else {
								fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("2", 64)
							}
							return r
						}
						t.Fatal("no injection target")
						return r
					}
					var check func(workspaceloader.Workspace) (Result, error)
					var closeSession func() error
					open := func() {
						if family == "ordinary" {
							session, err := NewSession(config)
							if err != nil {
								t.Fatal(err)
							}
							lower := session.lower
							session.lower = func(ctx context.Context, a compiler.Analysis) (compiler.Result, error) {
								r, err := lower(ctx, a)
								if err == nil && r.Verified {
									r = mutate(r)
								}
								return r, err
							}
							check = session.Check
							closeSession = session.Close
						} else {
							session, err := NewGenericSession(config, genericplan.DefaultPlanLimits())
							if err != nil {
								t.Fatal(err)
							}
							lower := session.lower
							session.lower = func(ctx context.Context, a compiler.Analysis, c admission.Capabilities) (compiler.Result, error) {
								r, err := lower(ctx, a, c)
								if err == nil && r.Verified {
									r = mutate(r)
								}
								return r, err
							}
							check = session.Check
							closeSession = session.Close
						}
					}
					open()
					defer func() { closeSession() }()
					workspace := loadCacheWorkspace(t, root)
					cold, err := check(workspace)
					if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || calls != 1 {
						t.Fatalf("seed stale cache: %+v %v", cold.Cache, err)
					}
					if storage == StorageDisk {
						if err := closeSession(); err != nil {
							t.Fatal(err)
						}
						open()
					}
					healed, err := check(workspace)
					if err != nil || !healed.Compiler.Verified || healed.Cache.Outcome != OutcomeFallback || !healed.Cache.ReassemblyFallback || !healed.Cache.Lowered || healed.Cache.Stored != 2 || calls != 2 {
						t.Fatalf("stale marker escaped: %+v %v", healed.Cache, err)
					}
					assertWorkspaceResult(t, workspace, healed.Compiler, 42)
					warm, err := check(workspace)
					if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered || calls != 2 {
						t.Fatalf("healed cache did not restore: %+v %v", warm.Cache, err)
					}
					assertWorkspaceResult(t, workspace, warm.Compiler, 42)
				})
			}
		}
	}
}
