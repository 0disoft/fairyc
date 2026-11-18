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
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The Option cache suite owns the shared ordinary configuration and exact disk
// envelope helper. Result adds no cache, artifact, target or planning limits.
const compositeResultCacheLibrary = `module answer;
pub fn Result<T,u32> pack<T>(T value){return .ok(value);}
pub fn Result<T,u32> relay<T>(Result<T,u32> value){
 let Result<T,u32> copied=value;discard value;
 var Result<T,u32> replaced=.err(9);discard replaced;
 replaced=copied;discard copied;return replaced;
}
pub fn Result<T,E> bind_error<T,E>(Result<T,E> value){return value;}
pub fn Result<T,u32> propagate<T>(Result<T,u32> value){let T actual=value?;return .ok(actual);}
pub fn T unpack<T>(Result<T,u32> value,T fallback){return switch(value){case .ok(actual):{actual}case .err(_):{fallback}};}
pub fn Result<T,u32> failure<T>(u32 code){return .err(code);}
pub fn u32 error_code(){return 77;}
`

func compositeResultCacheErrorSource() string {
	var text strings.Builder
	text.WriteString(`module main;import answer as a;fn i32 main(){
let Result<i8,u32> payload_choice=a::failure<i8>(77);discard a::relay<i8>(payload_choice);discard payload_choice;`)
	for i, scalar := range compositeOptionCacheScalars {
		fmt.Fprintf(&text, `let Result<%[1]s,u32> original%[2]d=a::failure<%[1]s>(a::error_code());
let Result<%[1]s,u32> copied%[2]d=a::relay<%[1]s>(original%[2]d);discard original%[2]d;
let Result<%[1]s,u32> forwarded%[2]d=a::bind_error<%[1]s,u32>(a::propagate<%[1]s>(copied%[2]d));discard copied%[2]d;
let u32 code%[2]d=switch(forwarded%[2]d){case .ok(_):{0}case .err(error):{error}};
assert(code%[2]d==a::error_code());`, scalar, i)
	}
	text.WriteString("return if(a::error_code()==77){42}else{43};}")
	return text.String()
}

func compositeResultCacheRoot(t *testing.T, main string) string {
	t.Helper()
	root := writeCacheWorkspace(t, compositeResultCacheLibrary)
	writeFile(t, filepath.Join(root, "src", "main.fy"), main)
	if parent := os.Getenv("FAIRYC_RESULT_CACHE_RECEIPTS"); parent != "" {
		if err := os.MkdirAll(parent, 0700); err != nil {
			t.Fatal(err)
		}
		retained, err := os.MkdirTemp(parent, "workspace-")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.CopyFS(retained, os.DirFS(root)); err != nil {
			t.Fatal(err)
		}
		root = retained
		t.Logf("retained source workspace: %s", root)
	}
	t.Logf("provider-sha256=%x main-sha256=%x", sha256.Sum256([]byte(compositeResultCacheLibrary)), sha256.Sum256([]byte(main)))
	return root
}

func compositeResultCacheSession(t *testing.T, storage, cacheRoot string) *GenericSession {
	t.Helper()
	return newGenericPersistentFixture(t, compositeOptionCacheConfiguration(t, storage, cacheRoot))
}

func forbidCompositeResultCacheLowering(t *testing.T, session *GenericSession) {
	t.Helper()
	session.lower = func(context.Context, compiler.Analysis, admission.Capabilities) (compiler.Result, error) {
		t.Fatal("complete Result cache hit invoked actual ordinary lowering")
		return compiler.Result{}, nil
	}
}

func checkCompositeResultCache(t *testing.T, session *GenericSession, root string, allHit bool) (workspaceloader.Workspace, Result) {
	t.Helper()
	workspace := loadCacheWorkspace(t, root)
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified {
		t.Fatalf("Result cache: %v %+v diagnostics=%+v unsupported=%+v", err, result.Cache, result.Compiler.Semantic.Diagnostics, result.Compiler.Semantic.Unsupported)
	}
	if result.Cache.Packages != 2 || allHit && (result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 2 || result.Cache.Misses != 0 || result.Cache.Rejected != 0 || result.Cache.Lowered || result.Cache.Stored != 0) {
		t.Fatalf("Result complete hit=%t: %+v", allHit, result.Cache)
	}
	if failures := mir.Verify(result.Compiler.MIR); len(failures) != 0 {
		t.Fatal(failures)
	}
	assertCompositeResultCacheOrigins(t, workspace, result.Compiler)
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(owner, "main")+"::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("cache=%+v", result.Cache)
	return workspace, result
}

func assertCompositeResultCacheOrigins(t *testing.T, workspace workspaceloader.Workspace, result compiler.Result) {
	t.Helper()
	declarations := result.GenericProvenance.Declarations()
	origins := result.GenericProvenance.Specializations()
	if len(declarations) != 6 || len(origins) == 0 {
		t.Fatalf("Result declarations/origins: %d/%d", len(declarations), len(origins))
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	current := map[genericplan.DeclarationID]compiler.GenericDeclaration{}
	for _, declaration := range declarations {
		matched := false
		for _, file := range input.Files {
			matched = matched || file.Package == declaration.Package && file.Path == declaration.Span.Source && file.Module.Path == declaration.Module
		}
		if !matched || !declaration.Public {
			t.Fatal("Result provenance lost current public source owner")
		}
		current[declaration.ID] = declaration
	}
	functions, types := map[string]bool{}, map[hir.Type]bool{}
	for _, function := range result.MIR.Functions {
		functions[function.Name] = true
		types[function.ResultType] = true
		for _, typ := range function.ParameterTypes {
			types[typ] = true
		}
	}
	for _, scalar := range compositeOptionCacheScalars {
		if !types[hir.Result(hir.ParseType(scalar), hir.TypeU32)] {
			t.Fatalf("restored MIR lost Result<%s,u32>", scalar)
		}
	}
	for _, origin := range origins {
		declaration, ok := current[origin.Declaration]
		if !ok || !functions[hir.QualifiedModulePath(declaration.Package, declaration.Module)+"::"+origin.GeneratedName] {
			t.Fatal("Result specialization did not join current declarations and restored MIR")
		}
	}
	encoded, err := json.Marshal(origins)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("current-specializations=%d provenance-sha256=%x", len(origins), sha256.Sum256(encoded))
}

func TestCompositeResultCacheMemoryAndDisk(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			main := compositeResultCacheErrorSource()
			root := compositeResultCacheRoot(t, main)
			cacheRoot := t.TempDir()
			session := compositeResultCacheSession(t, storage, cacheRoot)
			workspace, cold := checkCompositeResultCache(t, session, root, false)
			if cold.Cache.Misses != 2 || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("cold: %+v", cold.Cache)
			}
			assertWorkspaceResult(t, workspace, cold.Compiler, 42)
			lower := session.lower
			forbidCompositeResultCacheLowering(t, session)
			_, warm := checkCompositeResultCache(t, session, root, true)
			assertCompositeResultCachePayload(t, warm.Compiler, hir.TypeI8)
			if !reflect.DeepEqual(cold.Compiler.GenericProvenance.Declarations(), warm.Compiler.GenericProvenance.Declarations()) || !reflect.DeepEqual(cold.Compiler.GenericProvenance.Specializations(), warm.Compiler.GenericProvenance.Specializations()) {
				t.Fatal("Result warm provenance differs from cold current source")
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = compositeResultCacheSession(t, storage, cacheRoot)
				lower = session.lower
				forbidCompositeResultCacheLowering(t, session)
				checkCompositeResultCache(t, session, root, true)
			}
			mainPath := filepath.Join(root, "src", "main.fy")
			providerPath := filepath.Join(root, "libs", "answer", "src", "answer.fy")
			provider := compositeResultCacheLibrary
			for _, edit := range []string{"source", "provider", "payload", "u32-error-value"} {
				session.lower = lower
				switch edit {
				case "source":
					main += "\n// current Result consumer source\n"
				case "provider":
					provider = strings.Replace(provider, "discard copied;return replaced;", "discard replaced;return copied;", 1)
				case "payload":
					main = strings.Replace(main, "Result<i8,u32> payload_choice=a::failure<i8>(77);discard a::relay<i8>(payload_choice)", "Result<i16,u32> payload_choice=a::failure<i16>(77);discard a::relay<i16>(payload_choice)", 1)
				case "u32-error-value":
					provider = strings.Replace(provider, "return 77;", "return 78;", 1)
				}
				writeFile(t, mainPath, main)
				writeFile(t, providerPath, provider)
				workspace, changed := checkCompositeResultCache(t, session, root, false)
				if changed.Cache.Misses < 1 || !changed.Cache.Lowered || changed.Cache.Outcome == OutcomeAllHit || changed.Cache.Stored != 2 {
					t.Fatalf("%s retained stale Result cache: %+v", edit, changed.Cache)
				}
				if edit == "provider" && (changed.Cache.Misses != 2 || reflect.DeepEqual(warm.Compiler.GenericProvenance.Specializations(), changed.Compiler.GenericProvenance.Specializations())) {
					t.Fatal("provider body change retained old specialization closure")
				}
				if edit == "payload" {
					assertCompositeResultCachePayload(t, changed.Compiler, hir.TypeI16)
				}
				value := int32(42)
				if edit == "u32-error-value" {
					value = 43
				}
				assertWorkspaceResult(t, workspace, changed.Compiler, value)
				t.Logf("edit=%s provider-sha256=%x main-sha256=%x", edit, sha256.Sum256([]byte(provider)), sha256.Sum256([]byte(main)))
				forbidCompositeResultCacheLowering(t, session)
				workspace, hit := checkCompositeResultCache(t, session, root, true)
				if edit == "payload" || edit == "u32-error-value" {
					assertCompositeResultCachePayload(t, hit.Compiler, hir.TypeI16)
				}
				assertWorkspaceResult(t, workspace, hit.Compiler, value)
			}
			for _, invalid := range []struct{ name, extra, code string }{
				{"must-handle-copy", "fn void pending(Result<i8,u32> original){let Result<i8,u32> copied=original;discard original;}", "E4101"},
				{"error-contract", "fn Result<i8,i32> mismatch(Result<i8,u32> value){let i8 actual=value?;return .ok(actual);}", "E4103"},
			} {
				session.lower = lower
				writeFile(t, mainPath, main+invalid.extra)
				refused, err := session.Check(loadCacheWorkspace(t, root))
				if err != nil || refused.Compiler.Verified || refused.Cache.Outcome != OutcomeNotEligible || refused.Cache.Hits != 0 || refused.Cache.Stored != 0 || len(refused.Compiler.MIR.Functions) != 0 || len(refused.Compiler.GenericProvenance.Declarations()) != 0 {
					t.Fatalf("%s reused/published Result artifacts: %v %+v", invalid.name, err, refused.Cache)
				}
				found := false
				for _, diagnostic := range refused.Compiler.Semantic.Diagnostics {
					found = found || diagnostic.Code == invalid.code
				}
				if !found {
					t.Fatalf("%s lost source diagnostic %s: %+v", invalid.name, invalid.code, refused.Compiler.Semantic)
				}
				writeFile(t, mainPath, main)
				forbidCompositeResultCacheLowering(t, session)
				checkCompositeResultCache(t, session, root, true)
			}
		})
	}
}

func assertCompositeResultCachePayload(t *testing.T, result compiler.Result, scalar hir.Type) {
	t.Helper()
	var initializer hir.Span
	for _, module := range result.Semantic.HIR.Modules {
		for _, function := range module.Functions {
			if function.Name == "main" {
				for _, local := range function.Locals {
					if local.Name == "payload_choice" && local.Type == hir.Result(scalar, hir.TypeU32) {
						for _, statement := range function.Body.Statements {
							if statement.Kind == hir.StatementBinding && statement.Binding == local.Symbol {
								initializer = statement.Expression.Span
							}
						}
					}
				}
			}
		}
	}
	if initializer.EndByte == 0 {
		t.Fatal("current source Result payload selection was not rechecked")
	}
	results := map[hir.SymbolID]hir.Type{}
	for _, function := range result.MIR.Functions {
		results[function.Symbol] = function.ResultType
	}
	for _, function := range result.MIR.Functions {
		for _, block := range function.Blocks {
			call := block.Terminator
			if call.Kind == mir.TerminatorCall && call.Span.Source == initializer.Source && call.Span.StartByte == initializer.StartByte && call.Span.EndByte == initializer.EndByte && results[call.Callee] == hir.Result(scalar, hir.TypeU32) {
				return
			}
		}
	}
	t.Fatal("restored MIR payload-choice call lost the current initializer/type")
}

func compositeResultCacheNativeSource() (string, string) {
	var source, body, observer strings.Builder
	source.WriteString("module main;import answer as a;unsafe extern C fn void mark();unsafe extern C fn i32 finish();")
	observer.WriteString("#include <stdint.h>\n#include <string.h>\nstatic int32_t observed,marked,seeds,errors;void mark(void){marked++;}\n")
	values := []struct{ fy, c, value string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "(-INTPTR_MAX-1)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"},
		{"f32", "float", "1.25f"}, {"f32", "float", "0.0f"}, {"f32", "float", "-0.0f"},
		{"f64", "double", "-3.5"}, {"f64", "double", "0.0"}, {"f64", "double", "-0.0"},
	}
	for i, value := range values {
		code := []string{"0", "77", "4294967295"}[i%3]
		fmt.Fprintf(&source, `@repr(C) struct P%[1]d{%[2]s value;}unsafe extern C fn P%[1]d seed%[1]d();unsafe extern C fn i32 observe%[1]d(P%[1]d value);
unsafe extern C fn u32 error_seed%[1]d();unsafe extern C fn i32 error_observe%[1]d(u32 code);
fn Result<%[2]s,u32> written%[1]d(Result<%[2]s,u32> value){return value;}
fn Result<%[2]s,u32> counted%[1]d(bool fail){unsafe{mark();}if(fail){return .err(unsafe{error_seed%[1]d()});}let P%[1]d p=unsafe{seed%[1]d()};return .ok(p.value);}
fn Result<%[2]s,u32> forward%[1]d(bool fail){let %[2]s actual=a::propagate<%[2]s>(counted%[1]d(fail))?;return .ok(actual);}`, i, value.fy)
		fmt.Fprintf(&body, `let P%[1]d p%[1]d=unsafe{seed%[1]d()};
let Result<%[2]s,u32> original%[1]d=a::pack<%[2]s>(p%[1]d.value);
let Result<%[2]s,u32> copied%[1]d=written%[1]d(a::bind_error<%[2]s,u32>(a::relay<%[2]s>(original%[1]d)));
assert(switch(original%[1]d){case .ok(_):{true}case .err(_):{false}});assert(switch(copied%[1]d){case .ok(_):{true}case .err(_):{false}});
let %[2]s x%[1]d=a::unpack<%[2]s>(original%[1]d,p%[1]d.value);
let %[2]s y%[1]d=a::unpack<%[2]s>(copied%[1]d,p%[1]d.value);
let Result<%[2]s,u32> successful%[1]d=forward%[1]d(false);
let %[2]s z%[1]d=switch(successful%[1]d){case .ok(value):{value}case .err(_):{assert(false);p%[1]d.value}};
assert(unsafe{observe%[1]d(P%[1]d{.value=x%[1]d})}==1);assert(unsafe{observe%[1]d(P%[1]d{.value=y%[1]d})}==1);assert(unsafe{observe%[1]d(P%[1]d{.value=z%[1]d})}==1);
let Result<%[2]s,u32> failed%[1]d=forward%[1]d(true);
let u32 code%[1]d=switch(failed%[1]d){case .ok(_):{assert(false);0}case .err(code):{code}};
assert(unsafe{error_observe%[1]d(code%[1]d)}==1);`, i, value.fy)
		fmt.Fprintf(&observer, "typedef struct{%s value;} P%d;P%d seed%d(void){P%d p={%s};seeds++;return p;}int32_t observe%d(P%d p){%s expected=%s;observed++;return memcmp(&p.value,&expected,sizeof expected)==0;}uint32_t error_seed%d(void){errors++;return UINT32_C(%s);}int32_t error_observe%d(uint32_t code){return code==UINT32_C(%s);}\n", value.c, i, i, i, i, value.value, i, i, value.c, value.value, i, code, i, code)
	}
	source.WriteString("fn i32 main(){" + body.String() + "assert(unsafe{finish()}==1);return 42;}")
	observer.WriteString("int32_t finish(void){return observed==33 && marked==22 && seeds==22 && errors==11;}\n")
	return source.String(), observer.String()
}

func assertCompositeResultCacheNative(t *testing.T, workspace workspaceloader.Workspace, result Result) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 native Result observer; Linux remains batched")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(owner, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	_, observer := compositeResultCacheNativeSource()
	t.Logf("restored-C-sha256=%x observer-sha256=%x", sha256.Sum256([]byte(unit.Source)), sha256.Sum256([]byte(observer)))
	var retained string
	if parent := os.Getenv("FAIRYC_RESULT_CACHE_RECEIPTS"); parent != "" {
		retained, err = os.MkdirTemp(parent, "native-")
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(retained, "restored.c"), unit.Source)
		writeFile(t, filepath.Join(retained, "support.c"), runtimeSource+"\n"+observer)
		t.Logf("retained actual native inputs/images: %s", retained)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.Source, runtimeSource+"\n"+observer)
			if err != nil {
				t.Fatal(err)
			}
			if retained != "" {
				if err := os.WriteFile(filepath.Join(retained, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
			}
			outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("restored Result observer: %v %+v", err, outcome)
			}
			t.Logf("MSVC %s %s C11 image-sha256=%x bytes=%d exit42 empty streams; 33 payload matches, 22 marks/seeds, 11 active u32 errors", toolchain.Version, profile.Name, sha256.Sum256(artifact.Bytes), len(artifact.Bytes))
		})
	}
}

func TestCompositeResultCacheNativeRestore(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			source, _ := compositeResultCacheNativeSource()
			root := compositeResultCacheRoot(t, source)
			cacheRoot := t.TempDir()
			session := compositeResultCacheSession(t, storage, cacheRoot)
			_, cold := checkCompositeResultCache(t, session, root, false)
			if cold.Cache.Misses != 2 || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("native cold: %+v", cold.Cache)
			}
			forbidCompositeResultCacheLowering(t, session)
			checkCompositeResultCache(t, session, root, true)
			if storage == StorageMemory {
				workspace, warm := checkCompositeResultCache(t, session, root, true)
				t.Run("native", func(t *testing.T) {
					assertCompositeResultCacheNative(t, workspace, warm)
				})
				return
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			reader := compositeResultCacheSession(t, storage, cacheRoot)
			forbidCompositeResultCacheLowering(t, reader)
			checkCompositeResultCache(t, reader, root, true)
			if err := reader.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCompositeResultCacheChild$", "-test.count=1", "-test.v")
			command.Env = append(os.Environ(), "FAIRYC_COMPOSITE_RESULT_ROOT="+root, "FAIRYC_COMPOSITE_RESULT_STORE="+cacheRoot)
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "composite-result-new-process-disk-hit") {
				t.Fatalf("fresh Result cache process: %v %s", err, output)
			}
			t.Logf("fresh child: %s", output)
		})
	}
}

func TestCompositeResultCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_COMPOSITE_RESULT_ROOT")
	if root == "" {
		t.Skip("subprocess consumer only")
	}
	session := compositeResultCacheSession(t, StorageDisk, os.Getenv("FAIRYC_COMPOSITE_RESULT_STORE"))
	forbidCompositeResultCacheLowering(t, session)
	workspace, restored := checkCompositeResultCache(t, session, root, true)
	t.Run("native", func(t *testing.T) {
		assertCompositeResultCacheNative(t, workspace, restored)
	})
	if !t.Failed() {
		fmt.Println("composite-result-new-process-disk-hit")
	}
}

func TestCompositeResultCacheInnerMetadataForgeryRepair(t *testing.T) {
	root := compositeResultCacheRoot(t, compositeResultCacheErrorSource())
	cacheRoot := t.TempDir()
	session := compositeResultCacheSession(t, StorageDisk, cacheRoot)
	_, cold := checkCompositeResultCache(t, session, root, false)
	if cold.Cache.Stored != 2 {
		t.Fatal(cold.Cache)
	}
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
		t.Fatalf("actual disk records: %v %v", records, err)
	}
	raw, err := os.ReadFile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	var outer compositeOptionDiskEnvelope
	if err := json.Unmarshal(raw, &outer); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(outer)
	if err != nil || !reflect.DeepEqual(raw, canonical) || len(outer.Record.Artifacts) != 1 {
		t.Fatal("actual outer envelope is not the shared canonical form", err)
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
		t.Fatal("actual inner metadata is not the stable envelope", err)
	}
	old := inner.Interface
	inner.Interface = old[:len(old)-64] + strings.Repeat("0", 64)
	if inner.Interface == old {
		t.Fatal("forgery did not change the inner interface")
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
	path := filepath.Join(filepath.Dir(records[0]), hex.EncodeToString(digest[:])+".json")
	if err := os.WriteFile(path, forged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(records[0]); err != nil {
		t.Fatal(err)
	}
	repair := compositeResultCacheSession(t, StorageDisk, cacheRoot)
	workspace, repaired := checkCompositeResultCache(t, repair, root, false)
	if repaired.Cache.Rejected != 1 || repaired.Cache.Hits != 1 || !repaired.Cache.Lowered || repaired.Cache.Stored != 2 || repaired.Cache.Outcome == OutcomeAllHit {
		t.Fatalf("forged Result metadata was not rejected/healed: %+v", repaired.Cache)
	}
	assertWorkspaceResult(t, workspace, repaired.Compiler, 42)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("forged generation remains: %v", err)
	}
	if err := repair.Close(); err != nil {
		t.Fatal(err)
	}
	reader := compositeResultCacheSession(t, StorageDisk, cacheRoot)
	forbidCompositeResultCacheLowering(t, reader)
	workspace, restored := checkCompositeResultCache(t, reader, root, true)
	assertWorkspaceResult(t, workspace, restored.Compiler, 42)
	t.Logf("inner interface=%s forged-outer-sha256=%x; MIR/source/outer interface intact, fresh healed reader forbids lowering", old, digest)
}
