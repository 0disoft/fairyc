package workspacecompiler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	testRootManifest = `schema = "fairyc.manifest/v1"

[package]
uid = "test.workspace-cache.app"
name = "workspace-cache"
version = "1.0.0"
language = "fairyc-language/11"
source = "src"

[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`
	testAnswerManifest = `schema = "fairyc.manifest/v1"

[package]
uid = "test.workspace-cache.answer"
name = "answer"
version = "1.0.0"
language = "fairyc-language/11"
source = "src"
`
	testRootSource = `module main;

import answer as answer;

fn i32 main() {
    return answer::value();
}
`
	testRootSourceWithPrivateContext = `module main;

import answer as answer;

struct Hidden { i32 value; }

fn i32 main() {
    return answer::value();
}
`
	testAnswer42 = `module answer;

pub fn i32 value() {
    return 42;
}
`
)

func TestSessionSkipsMIRLoweringOnlyAfterCompleteVerifiedHits(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	session, err := NewSession(Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20},
		MaxArtifactBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	lower := session.lower
	lowerCalls := 0
	session.lower = func(analysis compiler.Analysis) (compiler.Result, error) {
		lowerCalls++
		return lower(analysis)
	}

	workspace := loadCacheWorkspace(t, root)
	first, err := session.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Compiler.Verified || first.Cache.Outcome != OutcomeFallback || !first.Cache.Lowered ||
		first.Cache.Packages != 2 || first.Cache.Hits != 0 || first.Cache.Misses != 2 || first.Cache.Stored != 2 || lowerCalls != 1 {
		t.Fatalf("cold Check() = %+v, lower calls = %d", first, lowerCalls)
	}
	assertWorkspaceResult(t, workspace, first.Compiler, 42)

	second, err := session.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Compiler.Verified || second.Cache.Outcome != OutcomeAllHit || second.Cache.Lowered ||
		second.Cache.Hits != 2 || second.Cache.Misses != 0 || lowerCalls != 1 {
		t.Fatalf("warm Check() = %+v, lower calls = %d", second, lowerCalls)
	}
	assertWorkspaceResult(t, workspace, second.Compiler, 42)
	stats := session.Stats()
	if stats.VerifiedHits != 2 || stats.Misses != 2 || stats.Rejected != 0 {
		t.Fatalf("warm cache Stats() = %+v", stats)
	}
}

func TestSessionInvalidatesSourceAndFallsBackFromMixedArtifactContexts(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	session, err := NewSession(Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20},
		MaxArtifactBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	lower := session.lower
	lowerCalls := 0
	session.lower = func(analysis compiler.Analysis) (compiler.Result, error) {
		lowerCalls++
		return lower(analysis)
	}

	base := loadCacheWorkspace(t, root)
	if _, err := session.Check(base); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "src", "main.fy"), testRootSourceWithPrivateContext)
	changed := loadCacheWorkspace(t, root)
	changedResult, err := session.Check(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedResult.Cache.Outcome != OutcomeFallback || changedResult.Cache.Hits != 1 || changedResult.Cache.Misses != 1 ||
		!changedResult.Cache.Lowered || changedResult.Cache.Stored != 2 || lowerCalls != 2 {
		t.Fatalf("changed Check() = %+v, lower calls = %d", changedResult, lowerCalls)
	}
	assertWorkspaceResult(t, changed, changedResult.Compiler, 42)

	writeFile(t, filepath.Join(root, "src", "main.fy"), testRootSource)
	reverted := loadCacheWorkspace(t, root)
	revertedResult, err := session.Check(reverted)
	if err != nil {
		t.Fatal(err)
	}
	if revertedResult.Cache.Outcome != OutcomeFallback || revertedResult.Cache.Hits != 2 || revertedResult.Cache.Misses != 0 ||
		!revertedResult.Cache.ReassemblyFallback || !revertedResult.Cache.Lowered || lowerCalls != 3 {
		t.Fatalf("mixed-context Check() = %+v, lower calls = %d", revertedResult, lowerCalls)
	}
	assertWorkspaceResult(t, reverted, revertedResult.Compiler, 42)

	healed, err := session.Check(reverted)
	if err != nil {
		t.Fatal(err)
	}
	if healed.Cache.Outcome != OutcomeAllHit || healed.Cache.ReassemblyFallback || healed.Cache.Lowered || lowerCalls != 3 {
		t.Fatalf("healed Check() = %+v, lower calls = %d", healed, lowerCalls)
	}
	assertWorkspaceResult(t, reverted, healed.Compiler, 42)
}

func TestPersistentSessionReusesStableMIRAcrossIndependentSessions(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	diskRoot := t.TempDir()
	configuration := persistentTestConfiguration(diskRoot, "test-compiler-build-a")
	firstSession, err := NewSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	firstLower := firstSession.lower
	firstLowerCalls := 0
	firstSession.lower = func(analysis compiler.Analysis) (compiler.Result, error) {
		firstLowerCalls++
		return firstLower(analysis)
	}
	workspace := loadCacheWorkspace(t, root)
	first, err := firstSession.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if first.Cache.Storage != StorageDisk || first.Cache.Outcome != OutcomeFallback || !first.Cache.Lowered ||
		first.Cache.Misses != 2 || first.Cache.Stored != 2 || firstLowerCalls != 1 {
		t.Fatalf("persistent cold Check() = %+v, lower calls = %d", first, firstLowerCalls)
	}
	assertWorkspaceResult(t, workspace, first.Compiler, 42)

	secondSession, err := NewSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	secondLowerCalls := 0
	secondSession.lower = func(analysis compiler.Analysis) (compiler.Result, error) {
		secondLowerCalls++
		return compiler.LowerAnalyzed(analysis)
	}
	second, err := secondSession.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if second.Cache.Storage != StorageDisk || second.Cache.Outcome != OutcomeAllHit || second.Cache.Lowered ||
		second.Cache.Hits != 2 || second.Cache.Misses != 0 || secondLowerCalls != 0 {
		t.Fatalf("persistent warm Check() = %+v, lower calls = %d", second, secondLowerCalls)
	}
	assertWorkspaceResult(t, workspace, second.Compiler, 42)
	stats, err := secondSession.PersistentStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.VerifiedHits != 2 || stats.Misses != 0 || stats.Rejected != 0 || stats.Store.Entries != 2 {
		t.Fatalf("persistent Stats() = %+v", stats)
	}

	incompatibleSession, err := NewSession(persistentTestConfiguration(diskRoot, "test-compiler-build-b"))
	if err != nil {
		t.Fatal(err)
	}
	incompatibleLower := incompatibleSession.lower
	incompatibleLowerCalls := 0
	incompatibleSession.lower = func(analysis compiler.Analysis) (compiler.Result, error) {
		incompatibleLowerCalls++
		return incompatibleLower(analysis)
	}
	incompatible, err := incompatibleSession.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if incompatible.Cache.Outcome != OutcomeFallback || incompatible.Cache.Misses != 2 || !incompatible.Cache.Lowered || incompatibleLowerCalls != 1 {
		t.Fatalf("incompatible persistent Check() = %+v, lower calls = %d", incompatible, incompatibleLowerCalls)
	}
}

func TestPersistentSessionRejectsAmbiguousConfiguration(t *testing.T) {
	t.Parallel()
	configuration := persistentTestConfiguration(t.TempDir(), "build")
	configuration.Cache = packagecache.Limits{MaxEntries: 1, MaxBytes: 1024}
	if _, err := NewSession(configuration); err == nil {
		t.Fatal("combined memory and persistent cache policies were accepted")
	}
	configuration = persistentTestConfiguration(t.TempDir(), "")
	if _, err := NewSession(configuration); err == nil {
		t.Fatal("empty persistent compiler build identity was accepted")
	}
}

func persistentTestConfiguration(root, compilerBuild string) Configuration {
	return Configuration{
		Target:           target.X8664WindowsMSVC,
		MaxArtifactBytes: 8 << 20,
		Persistent: &PersistentConfiguration{
			Root: root,
			Limits: packagecache.DiskLimits{
				MaxEntries: 8, MaxBytes: 32 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute,
			},
			CompilerBuild: compilerBuild,
		},
	}
}

func writeCacheWorkspace(t *testing.T, answerSource string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fairy.toml"), testRootManifest)
	writeFile(t, filepath.Join(root, "src", "main.fy"), testRootSource)
	writeFile(t, filepath.Join(root, "libs", "answer", "fairy.toml"), testAnswerManifest)
	writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), answerSource)
	return root
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadCacheWorkspace(t *testing.T, root string) workspaceloader.Workspace {
	t.Helper()
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return workspace
}

func assertWorkspaceResult(t *testing.T, workspace workspaceloader.Workspace, result compiler.Result, value int32) {
	t.Helper()
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(result.MIR, entry)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != value {
		t.Fatalf("interpreter outcome = %+v, want %d", outcome, value)
	}
}
