package workspacecompiler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Only add is requested. Neither multiply nor the complete i64 tuple is used.
// Keep mutation vectors equal-width so sibling closure cannot pass accidentally
// because changing a sibling moved the selected declaration's AST spans.
const groupedStaticCacheLibrary = `module answer;
@interface(Arithmetic) pub fn T add<T>(T left,T right){}
@interface(Arithmetic) pub fn T multiply<T>(T left,T right){}
@implements(add,i32) fn i32 add32(i32 left,i32 right){return left+right;}
@implements(multiply,i32) fn i32 mul32(i32 left,i32 right){return left*right;}
@implements(add,i64) fn i64 add64(i64 left,i64 right){return left+right;}
@implements(multiply,i64) fn i64 mul64(i64 left,i64 right){return left*right;}
pub fn T algorithm<T>(T left,T right){return add<T>(left,right);}
fn i32 unused(){return 1;}`

const groupedStaticCacheMain = `module main;import answer as a;fn i32 main(){return a::algorithm<i32>(20,22);}`

func groupedStaticCacheRoot(t *testing.T, library, main string) string {
	t.Helper()
	root := writeCacheWorkspace(t, library)
	writeFile(t, filepath.Join(root, "src", "main.fy"), main)
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("source library-sha256=%x main-sha256=%x test-executable=%s", sha256.Sum256([]byte(library)), sha256.Sum256([]byte(main)), identity.String())
	return root
}

func groupedStaticCacheSession(t *testing.T, storage, cacheRoot string) *GenericSession {
	t.Helper()
	if storage == StorageDisk {
		return newGenericPersistentFixture(t, genericPersistentConfiguration(t, cacheRoot))
	}
	session, _ := genericWorkspaceSession(t, 32<<20)
	t.Cleanup(func() { session.Close() })
	return session
}

func forbidGroupedStaticLowering(t *testing.T, session *GenericSession) {
	t.Helper()
	session.lower = func(context.Context, compiler.Analysis, admission.Capabilities) (compiler.Result, error) {
		t.Fatal("restored grouped static cache invoked actual lowering")
		return compiler.Result{}, nil
	}
}

func checkGroupedStaticCache(t *testing.T, session *GenericSession, root string, hits int, all bool) (workspaceloader.Workspace, Result) {
	t.Helper()
	workspace := loadCacheWorkspace(t, root)
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Hits != hits || (result.Cache.Outcome == OutcomeAllHit) != all || result.Cache.Lowered == all {
		t.Fatalf("cache: err=%v report=%+v diagnostics=%+v unsupported=%+v", err, result.Cache, result.Compiler.Semantic.Diagnostics, result.Compiler.Semantic.Unsupported)
	}
	if all && (result.Cache.Misses != 0 || result.Cache.Rejected != 0 || result.Cache.Stored != 0) {
		t.Fatalf("all-hit was not a clean restoration: %+v", result.Cache)
	}
	t.Logf("cache %+v", result.Cache)
	return workspace, result
}

func groupedStaticAlgorithmID(t *testing.T, result compiler.Result) genericplan.SpecializationID {
	t.Helper()
	var declaration genericplan.DeclarationID
	for _, d := range result.GenericProvenance.Declarations() {
		if d.Name == "algorithm" {
			declaration = d.ID
		}
	}
	for _, origin := range result.GenericProvenance.Specializations() {
		if origin.Declaration == declaration && declaration != "" {
			return origin.Specialization
		}
	}
	t.Fatal("algorithm has no current specialization")
	return ""
}

func assertGroupedStaticOnlyAdd(t *testing.T, result compiler.Result) {
	t.Helper()
	names := map[genericplan.DeclarationID]string{}
	for _, d := range result.GenericProvenance.Declarations() {
		names[d.ID] = d.Name
	}
	seen := map[string]int{}
	for _, origin := range result.GenericProvenance.Specializations() {
		seen[names[origin.Declaration]]++
	}
	if !reflect.DeepEqual(seen, map[string]int{"add": 1, "algorithm": 1}) {
		t.Fatalf("consumer must select only add<i32>, got specializations %v", seen)
	}
}

func TestGroupedStaticCacheColdWarmAndDependencyClosure(t *testing.T) {
	// Expectations are literal behavior/cache vectors, not a second implementation
	// of production's group digest or its dependency traversal.
	vectors := []struct {
		name, before, after string
		want                int32
		hits                int
		changed             bool
	}{
		{"used declaration", "fn T add<T>(T left,T right)", "fn T add<T>(T lhs_,T right)", 42, 0, true},
		{"unused declaration", "fn T multiply<T>(T left,T right)", "fn T multiply<T>(T lhs_,T right)", 42, 0, true},
		{"used provider", "add32(i32 left,i32 right){return left+right;}", "add32(i32 left,i32 right){return left-right;}", -2, 0, true},
		{"unused sibling provider", "mul32(i32 left,i32 right){return left*right;}", "mul32(i32 left,i32 right){return left+right;}", 42, 0, true},
		{"unused scalar sibling provider", "mul64(i64 left,i64 right){return left*right;}", "mul64(i64 left,i64 right){return left+right;}", 42, 0, true},
		{"private body", "unused(){return 1;}", "unused(){return 2;}", 42, 1, false},
	}
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := groupedStaticCacheRoot(t, groupedStaticCacheLibrary, groupedStaticCacheMain)
			cacheRoot := t.TempDir()
			session := groupedStaticCacheSession(t, storage, cacheRoot)
			workspace, cold := checkGroupedStaticCache(t, session, root, 0, false)
			if cold.Cache.Stored != 2 || cold.Cache.Misses != 2 {
				t.Fatalf("cold must store both packages: %+v", cold.Cache)
			}
			assertWorkspaceResult(t, workspace, cold.Compiler, 42)
			assertGroupedStaticOnlyAdd(t, cold.Compiler)
			lower := session.lower
			forbidGroupedStaticLowering(t, session)
			workspace, warm := checkGroupedStaticCache(t, session, root, 2, true)
			assertWorkspaceResult(t, workspace, warm.Compiler, 42)
			if !reflect.DeepEqual(cold.Compiler.GenericProvenance.Specializations(), warm.Compiler.GenericProvenance.Specializations()) {
				t.Fatal("restoration changed specialization identity")
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = groupedStaticCacheSession(t, storage, cacheRoot)
				lower = session.lower
				forbidGroupedStaticLowering(t, session)
				workspace, warm = checkGroupedStaticCache(t, session, root, 2, true)
				assertWorkspaceResult(t, workspace, warm.Compiler, 42)
			}
			session.lower = lower
			for _, v := range vectors {
				t.Run(v.name, func(t *testing.T) {
					// A fresh session/cache for each edit keeps exact two-package
					// expectations independent of prior vectors and LRU eviction.
					writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), groupedStaticCacheLibrary)
					probe := groupedStaticCacheSession(t, storage, t.TempDir())
					_, baseline := checkGroupedStaticCache(t, probe, root, 0, false)
					changed := strings.Replace(groupedStaticCacheLibrary, v.before, v.after, 1)
					if len(v.before) != len(v.after) || changed == groupedStaticCacheLibrary || len(changed) != len(groupedStaticCacheLibrary) {
						t.Fatal("invalidation vector must be a real equal-width edit")
					}
					writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), changed)
					workspace, edited := checkGroupedStaticCache(t, probe, root, v.hits, false)
					assertWorkspaceResult(t, workspace, edited.Compiler, v.want)
					assertGroupedStaticOnlyAdd(t, edited.Compiler)
					if (groupedStaticAlgorithmID(t, baseline.Compiler) != groupedStaticAlgorithmID(t, edited.Compiler)) != v.changed {
						t.Fatal("dependent algorithm specialization identity violated edit vector")
					}
					beforeDecls, afterDecls := baseline.Compiler.GenericProvenance.Declarations(), edited.Compiler.GenericProvenance.Declarations()
					// The declaration vectors intentionally rename a parameter;
					// compare source ownership/spans without the edited signature.
					for i := range beforeDecls {
						beforeDecls[i].Signature = compiler.GenericSignature{}
					}
					for i := range afterDecls {
						afterDecls[i].Signature = compiler.GenericSignature{}
					}
					if !reflect.DeepEqual(beforeDecls, afterDecls) {
						t.Fatal("equal-width edit unexpectedly changed declaration ownership or spans")
					}
					if (reflect.DeepEqual(baseline.Compiler.GenericProvenance.PackageInterfaces(), edited.Compiler.GenericProvenance.PackageInterfaces())) == v.changed {
						t.Fatal("group/private edit violated exported interface stability")
					}
					t.Logf("edit-sha256=%x algorithm-before=%s algorithm-after=%s", sha256.Sum256([]byte(changed)), groupedStaticAlgorithmID(t, baseline.Compiler), groupedStaticAlgorithmID(t, edited.Compiler))
					forbidGroupedStaticLowering(t, probe)
					workspace, restored := checkGroupedStaticCache(t, probe, root, 2, true)
					assertWorkspaceResult(t, workspace, restored.Compiler, v.want)
				})
			}
		})
	}
}

func TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples(t *testing.T) {
	// This list intentionally does not read genericsource.scalarTypes.
	scalars := []string{"bool", "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64"}
	for size := 2; size <= 16; size++ {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			var library, main strings.Builder
			library.WriteString("module answer;\n")
			names := []string{"add", "multiply"}
			for i := 2; i < size; i++ {
				names = append(names, fmt.Sprintf("spare%d", i))
			}
			for _, name := range names {
				fmt.Fprintf(&library, "@interface(Arithmetic) pub fn T %s<T>(T left,T right){}\n", name)
			}
			for _, scalar := range scalars {
				for _, name := range names {
					fmt.Fprintf(&library, "@implements(%s,%s) fn %s %s_%s(%s left,%s right){return left;}\n", name, scalar, scalar, name, scalar, scalar, scalar)
				}
			}
			library.WriteString("pub fn T algorithm<T>(T left,T right){return add<T>(left,right);}")
			main.WriteString("module main;import answer as a;")
			for _, scalar := range scalars {
				// Typed inputs avoid extending the language's literal admission.
				// Every probe is checked/lowered and requests its own scalar tuple.
				fmt.Fprintf(&main, "fn %s probe_%s(%s value){return a::algorithm<%s>(value,value);}", scalar, scalar, scalar, scalar)
			}
			main.WriteString("fn i32 main(){assert(probe_bool(true));assert(probe_i64(7)==7);return a::algorithm<i32>(42,1);}")
			root := groupedStaticCacheRoot(t, library.String(), main.String())
			session := groupedStaticCacheSession(t, StorageMemory, "")
			workspace, cold := checkGroupedStaticCache(t, session, root, 0, false)
			assertWorkspaceResult(t, workspace, cold.Compiler, 42)
			if len(cold.Compiler.GenericProvenance.Specializations()) != 26 {
				t.Fatal("expected add+algorithm specialized for exactly thirteen scalars")
			}
			forbidGroupedStaticLowering(t, session)
			workspace, warm := checkGroupedStaticCache(t, session, root, 2, true)
			assertWorkspaceResult(t, workspace, warm.Compiler, 42)
			id, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cbackend.Emit(warm.Compiler.MIR, hir.QualifiedModulePath(id, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGroupedStaticCacheRejectsInvalidUnusedBundles(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := groupedStaticCacheRoot(t, groupedStaticCacheLibrary, groupedStaticCacheMain)
			session := groupedStaticCacheSession(t, storage, t.TempDir())
			checkGroupedStaticCache(t, session, root, 0, false)
			checkGroupedStaticCache(t, session, root, 2, true)
			for _, v := range []struct{ name, before, after string }{
				{"partial used scalar", "@implements(multiply,i32)", ""},
				{"partial unused scalar", "@implements(multiply,i64)", ""},
				{"duplicate used tuple", "fn i32 unused()", "@implements(add,i32) fn i32 duplicate(i32 a,i32 b){return a;}fn i32 unused()"},
				{"duplicate unused tuple", "fn i32 unused()", "@implements(multiply,i64) fn i64 duplicate(i64 a,i64 b){return a;}fn i32 unused()"},
				{"wrong unused sibling signature", "mul32(i32 left,i32 right)", "mul32(i64 left,i32 right)"},
				{"wrong unused scalar signature", "mul64(i64 left,i64 right)", "mul64(i32 left,i64 right)"},
			} {
				t.Run(v.name, func(t *testing.T) {
					writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), strings.Replace(groupedStaticCacheLibrary, v.before, v.after, 1))
					bad, err := session.Check(loadCacheWorkspace(t, root))
					if err != nil || bad.Compiler.Verified || bad.Cache.Outcome != OutcomeNotEligible || bad.Cache.Hits != 0 || bad.Cache.Stored != 0 || bad.Cache.Lowered || len(bad.Compiler.MIR.Functions) != 0 || len(bad.Compiler.GenericProvenance.Specializations()) != 0 || len(bad.Compiler.Semantic.Diagnostics) == 0 || bad.Compiler.Semantic.Diagnostics[0].Code != "E4501" {
						t.Fatalf("invalid bundle exposed cached executable: %v %+v %+v", err, bad.Cache, bad.Compiler.Semantic.Diagnostics)
					}
					t.Logf("rejected %+v", bad.Compiler.Semantic.Diagnostics)
				})
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), groupedStaticCacheLibrary)
			forbidGroupedStaticLowering(t, session)
			workspace, restored := checkGroupedStaticCache(t, session, root, 2, true)
			assertWorkspaceResult(t, workspace, restored.Compiler, 42)
		})
	}
}

func TestGroupedStaticCachePersistentNewProcess(t *testing.T) {
	root := groupedStaticCacheRoot(t, groupedStaticCacheLibrary, groupedStaticCacheMain)
	cacheRoot := t.TempDir()
	session := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	checkGroupedStaticCache(t, session, root, 0, false)
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGroupedStaticCachePersistentChild$", "-test.v", "-test.count=1")
	command.Env = append(os.Environ(), "FAIRYC_GROUPED_STATIC_ROOT="+root, "FAIRYC_GROUPED_STATIC_CACHE="+cacheRoot, "GOMAXPROCS=1")
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "grouped-static-new-process-disk-42") {
		t.Fatalf("fresh process: %v\n%s", err, output)
	}
	t.Logf("child output:\n%s", output)
}

func TestGroupedStaticCachePersistentChild(t *testing.T) {
	root := os.Getenv("FAIRYC_GROUPED_STATIC_ROOT")
	if root == "" {
		t.Skip("subprocess consumer only; executed by PersistentNewProcess")
	}
	session := groupedStaticCacheSession(t, StorageDisk, os.Getenv("FAIRYC_GROUPED_STATIC_CACHE"))
	forbidGroupedStaticLowering(t, session)
	workspace, restored := checkGroupedStaticCache(t, session, root, 2, true)
	assertWorkspaceResult(t, workspace, restored.Compiler, 42)
	assertGroupedStaticOnlyAdd(t, restored.Compiler)
	fmt.Println("grouped-static-new-process-disk-42")
}

func TestGroupedStaticCacheInnerMetadataForgeryRepair(t *testing.T) {
	root := groupedStaticCacheRoot(t, groupedStaticCacheLibrary, groupedStaticCacheMain)
	cacheRoot := t.TempDir()
	session := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	checkGroupedStaticCache(t, session, root, 0, false)
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	var records []string
	if err := filepath.WalkDir(cacheRoot, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".json" {
			records = append(records, path)
		}
		return err
	}); err != nil || len(records) != 2 {
		t.Fatalf("records: %v %v", records, err)
	}
	raw, err := os.ReadFile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the existing exact outer-envelope helper. Only inner interface
	// metadata is forged; MIR, source, key and the outer interface stay intact.
	var outer compositeOptionDiskEnvelope
	if err := json.Unmarshal(raw, &outer); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(outer)
	if err != nil || !reflect.DeepEqual(raw, canonical) || len(outer.Record.Artifacts) != 1 {
		t.Fatal("outer record helper no longer matches actual canonical storage", err)
	}
	var inner struct {
		Schema    string          `json:"schema"`
		Key       string          `json:"key"`
		Interface string          `json:"interface"`
		MIR       json.RawMessage `json:"mir"`
	}
	if err := json.Unmarshal(outer.Record.Artifacts[0].Data, &inner); err != nil {
		t.Fatal(err)
	}
	canonical, err = json.Marshal(inner)
	if err != nil || !reflect.DeepEqual(canonical, outer.Record.Artifacts[0].Data) || len(inner.Interface) < 64 {
		t.Fatal("inner envelope seam no longer matches canonical storage", err)
	}
	oldInterface := inner.Interface
	inner.Interface = inner.Interface[:len(inner.Interface)-64] + strings.Repeat("0", 64)
	if inner.Interface == oldInterface {
		t.Fatal("forgery did not change metadata")
	}
	outer.Record.Artifacts[0].Data, err = json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	forged, err := json.Marshal(outer)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(forged)
	forgedPath := filepath.Join(filepath.Dir(records[0]), hex.EncodeToString(digest[:])+".json")
	if err := os.WriteFile(forgedPath, forged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(records[0]); err != nil {
		t.Fatal(err)
	}
	repair := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	workspace, repaired := checkGroupedStaticCache(t, repair, root, 1, false)
	if repaired.Cache.Rejected != 1 || repaired.Cache.Stored != 2 {
		t.Fatalf("valid-checksum inner metadata forgery did not reject and repair: %+v", repaired.Cache)
	}
	assertWorkspaceResult(t, workspace, repaired.Compiler, 42)
	if _, err := os.Stat(forgedPath); !os.IsNotExist(err) {
		t.Fatalf("forged generation retained: %v", err)
	}
	if err := repair.Close(); err != nil {
		t.Fatal(err)
	}
	reader := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	forbidGroupedStaticLowering(t, reader)
	workspace, restored := checkGroupedStaticCache(t, reader, root, 2, true)
	assertWorkspaceResult(t, workspace, restored.Compiler, 42)
	t.Logf("forged inner interface=%s outer-record-sha256=%x; one rejection, repaired fresh reader without lowering", oldInterface, digest)
}

func TestGroupedStaticCacheRestoredWindowsOrderedArguments(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC; Linux deferred")
	}
	main := `module main;import answer as a;
unsafe extern C fn i32 grouped_produce(i32 index);
unsafe extern C fn i32 grouped_finish(i32 result);
fn i32 main(){let i32 result=a::algorithm<i32>(unsafe{grouped_produce(1)},unsafe{grouped_produce(2)});return unsafe{grouped_finish(result)};}`
	root := groupedStaticCacheRoot(t, groupedStaticCacheLibrary, main)
	cacheRoot := t.TempDir()
	writer := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	checkGroupedStaticCache(t, writer, root, 0, false)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader := groupedStaticCacheSession(t, StorageDisk, cacheRoot)
	forbidGroupedStaticLowering(t, reader)
	workspace, restored := checkGroupedStaticCache(t, reader, root, 2, true)
	assertGroupedStaticOnlyAdd(t, restored.Compiler)
	id, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(restored.Compiler.MIR, hir.QualifiedModulePath(id, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	support += `
static int32_t grouped_position=0;
static int32_t grouped_bad=0;
int32_t grouped_produce(int32_t index){if(index!=++grouped_position)grouped_bad=1;return index==1?20:22;}
int32_t grouped_finish(int32_t result){return grouped_bad==0&&grouped_position==2&&result==42?42:1;}
`
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, cdriver.C11ConsumerMode, unit.Source, support)
			if err != nil {
				t.Fatal(err)
			}
			if len(artifact.Objects) != 2 {
				t.Fatal("generated program and argument observer must be separate objects")
			}
			outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("restored native grouped provider: %+v %v", outcome, err)
			}
			t.Logf("MSVC=%s mode=c11 profile=%s generated-C-sha256=%x support-sha256=%x executable-sha256=%x; ordered once-only arguments; exit42 empty streams", tc.Version, profile.Name, sha256.Sum256([]byte(unit.Source)), sha256.Sum256([]byte(support)), sha256.Sum256(artifact.Bytes))
		})
	}
}
