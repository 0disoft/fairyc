package workspacecompiler

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const genericAnswer = "module answer; pub fn T value<T>(T input) { return input; }"
const genericMain = "module main; import answer as answer; fn i32 main() { return answer::value<i32>(42); }"

func genericWorkspaceSession(t *testing.T, bytes uint64) (*GenericSession, string) {
	t.Helper()
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	session, err := NewGenericSession(Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: bytes}, MaxArtifactBytes: 8 << 20}, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	return session, root
}

func TestGenericSessionSkipsLoweringOnlyOnCurrentCompleteHits(t *testing.T) {
	session, root := genericWorkspaceSession(t, 32<<20)
	lower := session.lower
	calls := 0
	session.lower = func(analysis compiler.Analysis) (compiler.Result, error) { calls++; return lower(analysis) }
	workspace := loadCacheWorkspace(t, root)
	cold, err := session.Check(workspace)
	if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || cold.Cache.Misses != 2 || calls != 1 {
		t.Fatalf("cold: %+v %v", cold.Cache, err)
	}
	warm, err := session.Check(workspace)
	if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered || calls != 1 {
		t.Fatalf("warm: %+v %v", warm.Cache, err)
	}
	assertWorkspaceResult(t, workspace, warm.Compiler, 42)
	writeFile(t, filepath.Join(root, "src", "main.fy"), "module main; import answer as answer; fn i32 main() { let bool extra = answer::value<bool>(true); return answer::value<i32>(42); }")
	workspace = loadCacheWorkspace(t, root)
	changed, err := session.Check(workspace)
	if err != nil || !changed.Compiler.Verified || !changed.Cache.Lowered || changed.Cache.Misses != 2 || calls != 2 {
		t.Fatalf("new specialization: %+v %v", changed.Cache, err)
	}
	assertWorkspaceResult(t, workspace, changed.Compiler, 42)
	warmChanged, err := session.Check(workspace)
	if err != nil || warmChanged.Cache.Outcome != OutcomeAllHit || calls != 2 {
		t.Fatalf("changed warm: %+v %v", warmChanged.Cache, err)
	}
	writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), "module answer; pub fn T value<T>(T input) { return false; }")
	invalid, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || invalid.Compiler.Verified || invalid.Cache.Outcome != OutcomeNotEligible || invalid.Cache.Hits != 0 {
		t.Fatalf("invalid source reused old artifacts: %+v %v", invalid.Cache, err)
	}
}

func TestGenericSessionWarmResultRunsMSVC(t *testing.T) {
	session, root := genericWorkspaceSession(t, 32<<20)
	workspace := loadCacheWorkspace(t, root)
	if _, err := session.Check(workspace); err != nil {
		t.Fatal(err)
	}
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Outcome != OutcomeAllHit || result.Cache.Lowered {
		t.Fatalf("warm: %+v %v", result.Cache, err)
	}
	assertGenericWorkspaceNative(t, workspace, result)
}

func assertGenericWorkspaceNative(t *testing.T, workspace workspaceloader.Workspace, result Result) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(rootPackage, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
		})
	}
}

func TestGenericSessionOversizeAndInvalidSourceDoNotPublishHits(t *testing.T) {
	session, root := genericWorkspaceSession(t, 1)
	workspace := loadCacheWorkspace(t, root)
	for i := 0; i < 2; i++ {
		result, err := session.Check(workspace)
		if err != nil || !result.Compiler.Verified || result.Cache.StoreSkipped != 2 || result.Cache.Hits != 0 {
			t.Fatalf("bounded fallback: %+v %v", result.Cache, err)
		}
	}
	writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), "module answer; pub fn T value<T>(T input) { return false; }")
	invalid, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || invalid.Compiler.Verified || invalid.Cache.Outcome != OutcomeNotEligible || session.Stats().Store.Entries != 0 {
		t.Fatalf("invalid: %+v %v", invalid.Cache, err)
	}
	if _, err := NewGenericSession(Configuration{Persistent: &PersistentConfiguration{}}, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("empty persistent configuration silently admitted")
	}
}
