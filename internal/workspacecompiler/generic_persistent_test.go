package workspacecompiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func genericPersistentConfiguration(t *testing.T, root string) Configuration {
	t.Helper()
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	return Configuration{Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20, Persistent: &PersistentConfiguration{
		Root: root, CompilerBuild: identity.String(), Limits: packagecache.DiskLimits{MaxEntries: 8, MaxBytes: 32 << 20, MaxRecordBytes: 16 << 20, TempMaxAge: time.Minute},
	}}
}

func newGenericPersistentFixture(t *testing.T, configuration Configuration) *GenericSession {
	t.Helper()
	session, err := NewGenericSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestGenericPersistentSessionColdWarmAndInvalidation(t *testing.T) {
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	configuration := genericPersistentConfiguration(t, t.TempDir())
	session := newGenericPersistentFixture(t, configuration)
	workspace := loadCacheWorkspace(t, root)
	cold, err := session.Check(workspace)
	if err != nil || !cold.Compiler.Verified || cold.Cache.Storage != StorageDisk || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
		t.Fatalf("cold: %+v %v", cold.Cache, err)
	}
	reader := newGenericPersistentFixture(t, configuration)
	lower := reader.lower
	calls := 0
	reader.lower = func(a compiler.Analysis, capabilities admission.Capabilities) (compiler.Result, error) {
		calls++
		return lower(a, capabilities)
	}
	warm, err := reader.Check(workspace)
	if err != nil || warm.Cache.Outcome != OutcomeAllHit || calls != 0 {
		t.Fatalf("warm: %+v %v", warm.Cache, err)
	}
	assertWorkspaceResult(t, workspace, warm.Compiler, 42)
	writeFile(t, filepath.Join(root, "src", "main.fy"), "module main; import answer as answer; fn i32 main() { let bool extra = answer::value<bool>(true); return answer::value<i32>(42); }")
	workspace = loadCacheWorkspace(t, root)
	changed, err := reader.Check(workspace)
	if err != nil || !changed.Compiler.Verified || changed.Cache.Misses != 2 || calls != 1 {
		t.Fatalf("specialization change: %+v %v", changed.Cache, err)
	}
	assertWorkspaceResult(t, workspace, changed.Compiler, 42)
	changedBuild := configuration
	persistent := *configuration.Persistent
	changedBuild.Persistent = &persistent
	identity, err := buildidentity.Parse(persistent.CompilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	identity[0] ^= 1
	persistent.CompilerBuild = identity.String()
	other := newGenericPersistentFixture(t, changedBuild)
	result, err := other.Check(workspace)
	if err != nil || result.Cache.Misses != 2 || !result.Cache.Lowered {
		t.Fatalf("build invalidation: %+v %v", result.Cache, err)
	}
	writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), "module answer; pub fn T value<T>(T input) { return false; }")
	invalid, err := reader.Check(loadCacheWorkspace(t, root))
	if err != nil || invalid.Compiler.Verified || invalid.Cache.Outcome != OutcomeNotEligible || invalid.Cache.Storage != StorageDisk || invalid.Cache.Hits != 0 {
		t.Fatalf("invalid source: %+v %v", invalid.Cache, err)
	}
	stats, err := session.PersistentStats()
	if err != nil || stats.Store.Entries == 0 {
		t.Fatalf("stats: %+v %v", stats, err)
	}
}

func TestGenericPersistentSessionCrossProcess(t *testing.T) {
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	cacheRoot := t.TempDir()
	session := newGenericPersistentFixture(t, genericPersistentConfiguration(t, cacheRoot))
	cold, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || cold.Cache.Stored != 2 {
		t.Fatalf("cold: %+v %v", cold.Cache, err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGenericPersistentSessionChild$", "-test.count=1")
	command.Env = append(os.Environ(), "FAIRYC_GENERIC_WORKSPACE_CHILD="+root, "FAIRYC_GENERIC_WORKSPACE_CACHE="+cacheRoot)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "generic-workspace-disk-42") {
		t.Fatalf("fresh session process: %v %s", err, output)
	}
}

func TestGenericPersistentSessionChild(t *testing.T) {
	root := os.Getenv("FAIRYC_GENERIC_WORKSPACE_CHILD")
	if root == "" {
		t.Skip("subprocess consumer only")
	}
	session := newGenericPersistentFixture(t, genericPersistentConfiguration(t, os.Getenv("FAIRYC_GENERIC_WORKSPACE_CACHE")))
	session.lower = func(compiler.Analysis, admission.Capabilities) (compiler.Result, error) {
		t.Fatal("warm child invoked lowering")
		return compiler.Result{}, nil
	}
	workspace := loadCacheWorkspace(t, root)
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 2 || result.Cache.Lowered {
		t.Fatalf("warm: %+v %v", result.Cache, err)
	}
	assertWorkspaceResult(t, workspace, result.Compiler, 42)
	assertGenericProvenance(t, workspace, result.Compiler)
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		assertGenericWorkspaceNative(t, workspace, result)
	}
	fmt.Println("generic-workspace-disk-42")
}

func TestGenericPersistentSessionBoundsAndPolicy(t *testing.T) {
	configuration := genericPersistentConfiguration(t, t.TempDir())
	configuration.Cache = packagecache.Limits{MaxEntries: 1, MaxBytes: 1}
	if _, err := NewGenericSession(configuration, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("mixed policies accepted")
	}
	configuration.Cache = packagecache.Limits{}
	build := configuration.Persistent.CompilerBuild
	configuration.Persistent.CompilerBuild = "latest"
	if _, err := NewGenericSession(configuration, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("moving identity accepted")
	}
	configuration.Persistent.CompilerBuild = build
	configuration.MaxArtifactBytes = 1
	session := newGenericPersistentFixture(t, configuration)
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	for i := 0; i < 2; i++ {
		result, err := session.Check(loadCacheWorkspace(t, root))
		if err != nil || !result.Compiler.Verified || result.Cache.StoreSkipped != 2 || result.Cache.Hits != 0 {
			t.Fatalf("bounded fallback: %+v %v", result.Cache, err)
		}
	}
}
