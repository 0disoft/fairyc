package workspacecompiler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Forward-declared leaf, noncommutative arguments, and calls in both a local
// initializer and the return. The consumer itself is in another package.
const inlineCallsDependency = `module answer;
@inline pub fn i32 value(){let i32 first=leaf(48,4);return leaf(first,2);}
@inline pub fn i32 leaf(i32 a,i32 b){return a-b;}
@inline pub fn i32 spare(i32 a,i32 b){return a-b-1;}
@inline pub fn i32 unused(){return 7;}
@inline pub fn i32 legacy(i32 x){let i32 a=x+2;let b=a*2;return b/2;}
fn i32 private_helper(){return 1;}
`

func inlineCallsConfiguration(t *testing.T, storage, cacheRoot string) Configuration {
	t.Helper()
	if storage == StorageDisk {
		// The existing helper derives the exact test-executable build identity;
		// the fresh child runs that same executable, not a moving build label.
		return genericPersistentConfiguration(t, cacheRoot)
	}
	return Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
}

func newInlineCallsSession(t *testing.T, config Configuration) *Session {
	t.Helper()
	session, err := NewSession(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func forbidInlineCallsLowering(t *testing.T, session *Session) {
	t.Helper()
	session.lower = func(context.Context, compiler.Analysis) (compiler.Result, error) {
		t.Fatal("restored inline call graph invoked ordinary lowering")
		return compiler.Result{}, nil
	}
}

func checkInlineCallsCache(t *testing.T, session *Session, root string, want int32, hits int, allHit bool) (workspaceloader.Workspace, Result) {
	t.Helper()
	workspace := loadCacheWorkspace(t, root)
	// A flag alone is insufficient evidence of avoiding lowering.
	lower := session.lower
	if allHit {
		forbidInlineCallsLowering(t, session)
		defer func() { session.lower = lower }()
	}
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Packages != 2 || result.Cache.Hits != hits || (result.Cache.Outcome == OutcomeAllHit) != allHit || result.Cache.Lowered == allHit {
		t.Fatalf("inline calls cache: %v %+v diagnostics=%+v unsupported=%+v", err, result.Cache, result.Compiler.Semantic.Diagnostics, result.Compiler.Semantic.Unsupported)
	}
	if !allHit && result.Cache.Stored != 2 {
		t.Fatalf("did not store both packages: %+v", result.Cache)
	}
	assertWorkspaceResult(t, workspace, result.Compiler, want)
	if err := inlinebody.VerifyProgram(result.Compiler.Semantic.HIR, result.Compiler.MIR); err != nil {
		t.Fatal(err)
	}
	t.Logf("want=%d cache=%+v", want, result.Cache)
	return workspace, result
}

func inlineCallsMarkers(t *testing.T, result Result) map[string]string {
	t.Helper()
	markers := map[string]string{}
	for _, module := range result.Compiler.Semantic.HIR.Modules {
		if module.Path == "answer" {
			for _, fn := range module.Functions {
				markers[fn.Name] = fn.InlineBody
			}
		}
	}
	if markers["value"] == "" || markers["leaf"] == "" {
		t.Fatal("provider outer/leaf inline markers absent")
	}
	return markers
}

func assertInlineCallsRefused(t *testing.T, session *Session, root string) {
	t.Helper()
	// Session passes rejected analysis through lower to produce the ordinary
	// diagnostic result. Refusal forbids executable MIR/cache reuse, rather
	// than forbidding that dispatcher entry (which warm-hit tests do forbid).
	refused, err := session.Check(loadCacheWorkspace(t, root))
	if err != nil || refused.Compiler.Verified || refused.Cache.Outcome != OutcomeNotEligible || refused.Cache.Hits != 0 || refused.Cache.Stored != 0 || refused.Cache.Lowered || len(refused.Compiler.MIR.Functions) != 0 || len(refused.Compiler.Semantic.Unsupported) == 0 {
		t.Fatalf("inline source was not refused before executable/cache publication: %v %+v", err, refused)
	}
	t.Logf("refused: cache=%+v unsupported=%+v", refused.Cache, refused.Compiler.Semantic.Unsupported)
}

func TestInlineCallsCacheInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, inlineCallsDependency)
			config := inlineCallsConfiguration(t, storage, t.TempDir())
			if config.Persistent != nil {
				t.Logf("compiler-build=%s", config.Persistent.CompilerBuild)
			}
			session := newInlineCallsSession(t, config)
			_, cold := checkInlineCallsCache(t, session, root, 42, 0, false)
			t.Logf("fixture provider-sha256=%x consumer-sha256=%x", sha256.Sum256([]byte(inlineCallsDependency)), sha256.Sum256([]byte(testRootSource)))
			markers := inlineCallsMarkers(t, cold)
			// Fixed vectors, not computed by InlineFunctionID as a test oracle.
			// v3 is bindings(call leaf(literal 48,literal 4),
			//                call leaf(local 0,literal 2)); ordered args matter.
			for name, want := range map[string]string{
				"leaf":   "fyinlinefn1-sha256-e1c7867e72719b32aaf40f42cbe5afec989b512e5f88e14cd312525177e2ebe2",
				"legacy": "fyinlinefn1-sha256-6aac2a75fa26963e634daa3970fe133abdd3cd7e440ae96e9a12cefd94231aaf",
				"value":  "fyinlinefn1-sha256-9b8978c9825c3bcc925ddeb7fe1c1b7f023aad5cc2756916b82a7e1061c887af",
			} {
				if markers[name] != want {
					t.Fatalf("%s fixed identity: %s want %s", name, markers[name], want)
				}
			}
			checkInlineCallsCache(t, session, root, 42, 2, true)
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
				checkInlineCallsCache(t, session, root, 42, 2, true)
			}
			provider := filepath.Join(root, "libs", "answer", "src", "answer.fy")
			leafEdit := strings.Replace(inlineCallsDependency, "return a-b;", "return a-b+1;", 1)
			writeFile(t, provider, leafEdit)
			_, changed := checkInlineCallsCache(t, session, root, 44, 0, false)
			changedMarkers := inlineCallsMarkers(t, changed)
			if changedMarkers["value"] != markers["value"] || changedMarkers["leaf"] == markers["leaf"] {
				t.Fatal("leaf-only edit must preserve outer identity and change leaf identity")
			}
			checkInlineCallsCache(t, session, root, 44, 2, true)
			unusedEdit := strings.Replace(leafEdit, "return 7;", "return 8;", 1)
			writeFile(t, provider, unusedEdit)
			checkInlineCallsCache(t, session, root, 44, 0, false)
			checkInlineCallsCache(t, session, root, 44, 2, true)
			privateEdit := strings.Replace(unusedEdit, "return 1;", "return 2;", 1)
			writeFile(t, provider, privateEdit)
			_, private := checkInlineCallsCache(t, session, root, 44, 1, false)
			if inlineCallsMarkers(t, private)["value"] != markers["value"] {
				t.Fatal("private edit changed outer identity")
			}
			privateFound := false
			for _, fn := range private.Compiler.MIR.Functions {
				if strings.HasSuffix(fn.Name, "::private_helper") {
					privateFound = true
					out, err := interpreter.Execute(private.Compiler.MIR, fn.Name)
					if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 2 {
						t.Fatalf("owner private edit was not refreshed: %+v %v", out, err)
					}
				}
			}
			if !privateFound {
				t.Fatal("refreshed provider private helper missing")
			}
			checkInlineCallsCache(t, session, root, 44, 2, true)
			// Change call targets only; retain both original declarations.
			retarget := strings.Replace(privateEdit, "first=leaf(48,4);return leaf(first,2)", "first=spare(48,4);return spare(first,2)", 1)
			writeFile(t, provider, retarget)
			_, retargeted := checkInlineCallsCache(t, session, root, 40, 0, false)
			if inlineCallsMarkers(t, retargeted)["value"] == markers["value"] {
				t.Fatal("retargeting did not change caller identity")
			}
			checkInlineCallsCache(t, session, root, 40, 2, true)
			writeFile(t, provider, strings.Replace(retarget, "@inline pub fn i32 spare", "pub fn i32 spare", 1))
			assertInlineCallsRefused(t, session, root)
		})
	}
}

func TestInlineCallsCacheRefusesClosedTargets(t *testing.T) {
	for _, test := range []struct{ name, source, helper string }{
		{"ordinary", `module answer;@inline pub fn i32 value(){return leaf();}pub fn i32 leaf(){return 42;}`, ""},
		{"private", `module answer;@inline pub fn i32 value(){return leaf();}fn i32 leaf(){return 42;}`, ""},
		{"other_module", `module answer;import helper as h;@inline pub fn i32 value(){return h::leaf();}`, `module helper;@inline pub fn i32 leaf(){return 42;}`},
		{"self_cycle", `module answer;@inline pub fn i32 value(){return value();}`, ""},
		{"mutual_cycle", `module answer;@inline pub fn i32 value(){return leaf();}@inline pub fn i32 leaf(){return value();}`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := writeCacheWorkspace(t, strings.ReplaceAll(test.source, "@inline ", ""))
			if test.helper != "" {
				writeFile(t, filepath.Join(root, "libs", "answer", "src", "helper.fy"), test.helper)
			}
			session := newInlineCallsSession(t, inlineCallsConfiguration(t, StorageMemory, ""))
			// Ordinary control proves module resolution/typechecking is valid.
			control, err := session.Check(loadCacheWorkspace(t, root))
			if err != nil || !control.Compiler.Verified {
				t.Fatalf("ordinary control: %v %+v", err, control.Compiler.Semantic)
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), test.source)
			assertInlineCallsRefused(t, session, root)
		})
	}
}

func TestInlineCallsCacheGraphBound(t *testing.T) {
	// Repeated references to the next function form a shared DAG, not a tree.
	// value,f1..f63 is exactly 64 distinct reachable functions including root.
	for _, count := range []int{64, 65} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var source strings.Builder
			source.WriteString("module answer;")
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("f%d", i)
				if i == 0 {
					name = "value"
				}
				fmt.Fprintf(&source, "@inline pub fn i32 %s(){", name)
				if i == count-1 {
					source.WriteString("return 42;}")
				} else {
					fmt.Fprintf(&source, "let i32 unused=f%d();return f%d();}", i+1, i+1)
				}
			}
			root := writeCacheWorkspace(t, source.String())
			session := newInlineCallsSession(t, inlineCallsConfiguration(t, StorageMemory, ""))
			if count == 65 {
				assertInlineCallsRefused(t, session, root)
				return
			}
			// Do not execute this intentionally exponential runtime graph. The
			// admission/cache path must not expand or execute its shared bodies.
			workspace := loadCacheWorkspace(t, root)
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 {
				t.Fatalf("64-function DAG: %v %+v %+v", err, cold.Cache, cold.Compiler.Semantic)
			}
			forbidInlineCallsLowering(t, session)
			warm, err := session.Check(workspace)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
				t.Fatalf("64-function DAG restore: %v %+v", err, warm.Cache)
			}
		})
	}
}

func TestInlineCallsCacheInnerMarkerRepair(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, inlineCallsDependency)
			config := inlineCallsConfiguration(t, storage, t.TempDir())
			session := newInlineCallsSession(t, config)
			lower := session.lower
			calls := 0
			session.lower = func(ctx context.Context, a compiler.Analysis) (compiler.Result, error) {
				calls++
				r, err := lower(ctx, a)
				if err == nil && r.Verified && calls == 1 {
					found := false
					for i := range r.MIR.Functions {
						if strings.HasSuffix(r.MIR.Functions[i].Name, "::leaf") {
							r.MIR.Functions[i].InlineBody = hir.InlineFunctionPrefix + strings.Repeat("2", 64)
							found = true
						}
					}
					if !found {
						t.Fatal("no inner leaf marker to forge")
					}
				}
				return r, err
			}
			workspace := loadCacheWorkspace(t, root)
			seed, err := session.Check(workspace)
			if err != nil || !seed.Compiler.Verified || seed.Cache.Stored != 2 || calls != 1 {
				t.Fatalf("seed inner forgery: %v %+v", err, seed.Cache)
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
			}
			_, repair := checkInlineCallsCache(t, session, root, 42, 2, false)
			if !repair.Cache.ReassemblyFallback || repair.Cache.Outcome != OutcomeFallback {
				t.Fatalf("inner marker escaped fallback: %+v", repair.Cache)
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
			} else if calls != 2 {
				t.Fatalf("repair lowering calls=%d want 2", calls)
			}
			checkInlineCallsCache(t, session, root, 42, 2, true)
		})
	}
}

func assertInlineCallsNative(t *testing.T, workspace workspaceloader.Workspace, result Result) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native Linux deferred to batched validation; requires Windows amd64 MSVC")
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil || !strings.Contains(unit.Source, "static inline int32_t") {
		t.Fatal("restored inline graph missing from C", err)
	}
	support, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	support += `
static int32_t position=0;
int32_t inline_call_produce(int32_t index){if(index!=++position)return -1000;return index==1?48:4;}
int32_t inline_call_finish(int32_t value){return position==2&&value==42?42:1;}
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
				t.Fatal("expected separate restored caller and independent observer objects")
			}
			if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
				t.Fatal(err)
			}
			if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
				t.Fatal(err)
			}
			out, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
			if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
				t.Fatalf("restored inline calls native: %v %+v", err, out)
			}
			t.Logf("MSVC %s C11 %s: storage=%s generated-C-sha256=%x support-sha256=%x image-sha256=%x; ordered once-only args, local/return calls; exit42 empty streams", tc.Version, profile.Name, result.Cache.Storage, sha256.Sum256([]byte(unit.Source)), sha256.Sum256([]byte(support)), sha256.Sum256(artifact.Bytes))
		})
	}
}

func TestInlineCallsCacheNativeRestore(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native Linux deferred to batched validation; requires Windows amd64 MSVC")
	}
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, strings.Replace(inlineCallsDependency, "value(){let i32 first=leaf(48,4)", "value(i32 a,i32 b){let i32 first=leaf(a,b)", 1))
			writeFile(t, filepath.Join(root, "src", "main.fy"), `module main;import answer as a;
unsafe extern C fn i32 inline_call_produce(i32 index);
unsafe extern C fn i32 inline_call_finish(i32 result);
fn i32 main(){let i32 result=a::value(unsafe{inline_call_produce(1)},unsafe{inline_call_produce(2)});return unsafe{inline_call_finish(result)};}`)
			cacheRoot := t.TempDir()
			config := inlineCallsConfiguration(t, storage, cacheRoot)
			if config.Persistent != nil {
				t.Logf("compiler-build=%s", config.Persistent.CompilerBuild)
			}
			session := newInlineCallsSession(t, config)
			workspace := loadCacheWorkspace(t, root)
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("native cold: %v %+v %+v", err, cold.Cache, cold.Compiler.Semantic)
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
			}
			forbidInlineCallsLowering(t, session)
			warm, err := session.Check(workspace)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
				t.Fatalf("native warm: %v %+v", err, warm.Cache)
			}
			if storage == StorageMemory {
				assertInlineCallsNative(t, workspace, warm)
				return
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInlineCallsCacheChild$", "-test.count=1", "-test.v")
			command.Env = append(os.Environ(), "FAIRYC_INLINE_CALLS_CHILD="+root, "FAIRYC_INLINE_CALLS_STORE="+cacheRoot)
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "inline-calls-new-process-disk-42") {
				t.Fatalf("fresh child: %v %s", err, output)
			}
			t.Logf("fresh child: %s", output)
		})
	}
}

func TestInlineCallsCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_INLINE_CALLS_CHILD")
	if root == "" {
		t.Skip("fresh-process consumer only")
	}
	session := newInlineCallsSession(t, inlineCallsConfiguration(t, StorageDisk, os.Getenv("FAIRYC_INLINE_CALLS_STORE")))
	forbidInlineCallsLowering(t, session)
	workspace := loadCacheWorkspace(t, root)
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Storage != StorageDisk || result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 2 || result.Cache.Lowered {
		t.Fatalf("fresh-process disk hit: %v %+v", err, result.Cache)
	}
	if err := inlinebody.VerifyProgram(result.Compiler.Semantic.HIR, result.Compiler.MIR); err != nil {
		t.Fatal(err)
	}
	assertInlineCallsNative(t, workspace, result)
	fmt.Println("inline-calls-new-process-disk-42")
}
