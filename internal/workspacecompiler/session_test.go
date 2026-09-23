package workspacecompiler

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packageartifact"
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
language = "fairyc-language/19"
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
language = "fairyc-language/19"
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

func TestPackageKeysConsumeOnlyTheOwningContractStateBody(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	workspace := loadCacheWorkspace(t, root)
	session, err := NewSession(Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20},
		MaxArtifactBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	analysis, err := compiler.AnalyzePackages(input)
	if err != nil {
		t.Fatal(err)
	}
	interfaces, err := packageInterfaces(analysis.Semantic.HIR, metadata, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	owners := functionOwners(analysis.Semantic.HIR)
	bodies, err := contractstatemir.ComputeHIRStateBodyHashes(analysis.Semantic.HIR, owners, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := session.packageKeys(metadata, interfaces, bodies, owners)
	if err != nil {
		t.Fatal(err)
	}
	changedBodies := maps.Clone(bodies)
	changedOwner := owners[0]
	changed := changedBodies[changedOwner]
	changed[0]++
	changedBodies[changedOwner] = changed
	changedKeys, err := session.packageKeys(metadata, interfaces, changedBodies, owners)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range owners {
		if owner == changedOwner && changedKeys[owner] == baseline[owner] {
			t.Fatalf("package %d body change preserved its cache key", owner)
		}
		if owner != changedOwner && changedKeys[owner] != baseline[owner] {
			t.Fatalf("package %d body change invalidated unrelated owner %d", changedOwner, owner)
		}
	}
	delete(changedBodies, changedOwner)
	if _, err := session.packageKeys(metadata, interfaces, changedBodies, owners); err == nil {
		t.Fatal("package keys accepted a missing current body identity")
	}
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

func TestPersistentSessionReassemblesCachedStateWithMIRSymbols(t *testing.T) {
	root := writeCacheWorkspace(t, testAnswer42)
	workspace := loadCacheWorkspace(t, root)
	analysis := statefulCacheAnalysis(t, workspace)
	diskRoot := t.TempDir()
	configuration := persistentTestConfiguration(diskRoot, "test-stateful-compiler-build")

	writer, err := NewSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	cold := checkPersistentAnalysis(t, writer, workspace, analysis, false)
	if cold.Cache.Outcome != OutcomeFallback || !cold.Cache.Lowered || cold.Cache.Stored != 2 || len(cold.Compiler.StateTransactions) != 1 {
		t.Fatalf("cold stateful Check() = %+v", cold)
	}
	assertStateTransactionResult(t, cold, 42)

	reader, err := NewSession(configuration)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	reader.lower = func(compiler.Analysis) (compiler.Result, error) {
		t.Fatal("persistent all-hit state reassembly invoked ordinary MIR lowering")
		return compiler.Result{}, nil
	}
	warm := checkPersistentAnalysis(t, reader, workspace, analysis, true)
	if warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered || warm.Cache.Hits != 2 || len(warm.Compiler.StateTransactions) != 1 {
		t.Fatalf("warm stateful Check() = %+v", warm)
	}
	assertStateTransactionResult(t, warm, 42)
	if cold.Compiler.StateTransactions[0].Program.Capability == warm.Compiler.StateTransactions[0].Program.Capability {
		t.Fatal("persistent reassembly reused source-lowered transaction authority")
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

func TestPersistentSessionCompilerIdentityBindsCompilationFormats(t *testing.T) {
	t.Parallel()
	session, err := NewSession(persistentTestConfiguration(t.TempDir(), "test-compiler-build"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	want := compilerStableEncoding + ";test-compiler-build;" +
		packageartifact.StableCompilationVersion + ";" +
		packageartifact.StableArtifactVersion + ";" +
		contractstatemir.StableStateBodyVersion + ";" +
		packagecache.DiskRecordVersion
	if session.compilerBuild != want {
		t.Fatalf("persistent compiler identity = %q, want %q", session.compilerBuild, want)
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

func statefulCacheAnalysis(t *testing.T, workspace workspaceloader.Workspace) compiler.Analysis {
	t.Helper()
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	analysis, err := compiler.AnalyzePackages(input)
	if err != nil {
		t.Fatal(err)
	}
	var packageID hir.PackageID
	var symbol hir.SymbolID
	foundOwner := false
	for moduleIndex := range analysis.Semantic.HIR.Modules {
		module := &analysis.Semantic.HIR.Modules[moduleIndex]
		if module.Path != "answer" {
			continue
		}
		for functionIndex := range module.Functions {
			function := &module.Functions[functionIndex]
			if function.Name != "value" {
				continue
			}
			if foundOwner {
				t.Fatal("answer::value HIR owner is ambiguous")
			}
			packageID, symbol = module.Package, function.Symbol
			foundOwner = true
			function.Effects = hir.CanonicalEffects([]hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite})
		}
	}
	if !foundOwner || symbol == hir.NoSymbol {
		t.Fatal("answer::value HIR owner was not found")
	}
	encoded, err := contractstate.U64.Encode(42)
	if err != nil {
		t.Fatal(err)
	}
	analysis.Semantic.HIR.StateLayouts = []hir.StateLayout{{
		Package: packageID, ContractUID: "test.workspace-cache.answer",
		Fields: []hir.StateField{{Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()}},
	}}
	analysis.Semantic.HIR.StateTransactions = []hir.StateTransaction{{
		Package: packageID, Function: symbol, Exit: hir.StateTransactionCommit,
		Accesses: []hir.StateAccess{
			{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded}},
			{Kind: hir.StateAccessCellRead, Field: "value"},
		},
	}}
	return analysis
}

func checkPersistentAnalysis(t *testing.T, session *Session, workspace workspaceloader.Workspace, analysis compiler.Analysis, stripCurrentOperations bool) Result {
	t.Helper()
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	interfaces, err := packageInterfaces(analysis.Semantic.HIR, metadata, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	owners := functionOwners(analysis.Semantic.HIR)
	bodies, err := contractstatemir.ComputeHIRStateBodyHashes(analysis.Semantic.HIR, owners, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := session.packageKeys(metadata, interfaces, bodies, owners)
	if err != nil {
		t.Fatal(err)
	}
	if stripCurrentOperations {
		analysis.Semantic.HIR.StateTransactions = nil
	}
	result, err := session.checkPersistent(
		analysis, identities, coordinates, interfaces, bodies, owners, keys,
		Report{Outcome: OutcomeFallback, Storage: StorageDisk, Packages: len(owners)},
	)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertStateTransactionResult(t *testing.T, result Result, value uint64) {
	t.Helper()
	if len(result.Compiler.StateTransactions) != 1 {
		t.Fatalf("state transactions = %+v", result.Compiler.StateTransactions)
	}
	transaction := result.Compiler.StateTransactions[0]
	foundFunction := false
	for _, function := range result.Compiler.MIR.Functions {
		if strings.HasSuffix(function.Name, "::answer::value") {
			foundFunction = true
			if function.Symbol != transaction.Function {
				t.Fatalf("state function symbol %d does not align with MIR symbol %d", transaction.Function, function.Symbol)
			}
		}
	}
	if !foundFunction {
		t.Fatal("reassembled MIR has no answer::value function")
	}
	store, err := contractstatemir.NewMemoryStore(transaction.Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := contractstatemir.Execute(store, transaction.Program, contractstate.Limits{
		MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	want, err := contractstate.U64.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != contractstatemir.ExitCommit || len(outcome.Reads) != 1 || !outcome.Reads[0].Found ||
		outcome.Reads[0].Value.TypeID != contractstate.U64.TypeID() || !bytes.Equal(outcome.Reads[0].Value.Encoded, want) {
		t.Fatalf("state execution = %+v, want committed read %d", outcome, value)
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
