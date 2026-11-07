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

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const compositeOptionCacheLibrary = `module answer;
pub fn Option<T> pack<T>(T value){return .some(value);}
pub fn Option<T> relay<T>(Option<T> value){
 let Option<T> copied=value;
 var Option<T> replaced=.none;
 replaced=copied;
 discard value;
 return replaced;
}
pub fn T unpack<T>(Option<T> value,T fallback){
 return switch(value){case .some(actual):{actual}case .none:{fallback}};
}
pub fn Option<T> empty<T>(){return .none;}
pub fn i32 tag<T>(Option<T> value){return switch(value){case .some(_):{1}case .none:{0}};}
`

var compositeOptionCacheScalars = []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"}

func compositeOptionCacheNoneSource() string {
	var text strings.Builder
	text.WriteString(`module main;import answer as a;fn i32 main(){
let Option<i8> payload_choice=a::empty<i8>();discard a::relay<i8>(payload_choice);`)
	for i, scalar := range compositeOptionCacheScalars {
		fmt.Fprintf(&text, `let Option<%[1]s> value%[2]d=a::empty<%[1]s>();let Option<%[1]s> copied%[2]d=a::relay<%[1]s>(value%[2]d);assert(a::tag<%[1]s>(copied%[2]d)==0);assert(a::tag<%[1]s>(value%[2]d)==0);`, scalar, i)
	}
	text.WriteString("return 42;}")
	return text.String()
}

func compositeOptionCacheConfiguration(t *testing.T, storage, cacheRoot string) Configuration {
	t.Helper()
	if storage == StorageDisk {
		return genericPersistentConfiguration(t, cacheRoot)
	}
	// Reuse the existing broader-generics fixture policy; no new cache limits.
	return Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
}

func newCompositeOptionCacheSession(t *testing.T, config Configuration) *GenericSession {
	t.Helper()
	session, err := NewGenericSession(config, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func forbidCompositeOptionCacheLowering(t *testing.T, session *GenericSession) {
	t.Helper()
	session.lower = func(context.Context, compiler.Analysis, admission.Capabilities) (compiler.Result, error) {
		t.Fatal("complete Option cache hit invoked ordinary lowering")
		return compiler.Result{}, nil
	}
}

func checkCompositeOptionCache(t *testing.T, session *GenericSession, root string, allHit bool) (workspaceloader.Workspace, Result) {
	t.Helper()
	workspace := loadCacheWorkspace(t, root) // Re-read current source on every check.
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified {
		t.Fatalf("Option cache compilation: %v %+v", err, result.Cache)
	}
	if allHit && (result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 2 || result.Cache.Misses != 0 || result.Cache.Rejected != 0 || result.Cache.Lowered || result.Cache.Stored != 0) {
		t.Fatalf("complete Option hit: %+v", result.Cache)
	}
	if failures := mir.Verify(result.Compiler.MIR); len(failures) != 0 {
		t.Fatal(failures)
	}
	assertCompositeOptionCacheOrigins(t, workspace, result.Compiler)
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(rootPackage, "main")+"::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	return workspace, result
}

func assertCompositeOptionCacheOrigins(t *testing.T, workspace workspaceloader.Workspace, result compiler.Result) {
	t.Helper()
	declarations := result.GenericProvenance.Declarations()
	origins := result.GenericProvenance.Specializations()
	if len(declarations) != 5 || len(origins) == 0 {
		t.Fatalf("current Option declarations/origins: %d/%d", len(declarations), len(origins))
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
			t.Fatal("Option provenance lost current public source owner")
		}
		current[declaration.ID] = declaration
	}
	functions := map[string]mir.Function{}
	types := map[hir.Type]bool{}
	for _, function := range result.MIR.Functions {
		functions[function.Name] = function
		types[function.ResultType] = true
		for _, typ := range function.ParameterTypes {
			types[typ] = true
		}
	}
	for _, scalar := range compositeOptionCacheScalars {
		if !types[hir.Option(hir.ParseType(scalar))] {
			t.Fatalf("restored MIR lost Option<%s>", scalar)
		}
	}
	for _, origin := range origins {
		declaration, ok := current[origin.Declaration]
		if !ok {
			t.Fatal("Option specialization has no current declaration")
		}
		qualified := hir.QualifiedModulePath(declaration.Package, declaration.Module) + "::" + origin.GeneratedName
		if _, ok := functions[qualified]; !ok {
			t.Fatal("current Option specialization did not join restored MIR")
		}
	}
}

func assertCompositeOptionCachePayloadChoice(t *testing.T, result compiler.Result, scalar hir.Type) {
	t.Helper()
	var initializer hir.Span
	for _, module := range result.Semantic.HIR.Modules {
		for _, function := range module.Functions {
			if function.Name != "main" {
				continue
			}
			for _, local := range function.Locals {
				if local.Name != "payload_choice" {
					continue
				}
				if local.Type != hir.Option(scalar) {
					t.Fatalf("current source payload choice: %s", local.Type)
				}
				for _, statement := range function.Body.Statements {
					if statement.Kind == hir.StatementBinding && statement.Binding == local.Symbol {
						initializer = statement.Expression.Span
					}
				}
			}
		}
	}
	if initializer.EndByte == 0 {
		t.Fatal("fixture has no current payload-choice initializer")
	}
	// Inert Option locals have no resource metadata; their call result type
	// comes from the selected concrete MIR callee, with the current source span.
	results := map[hir.SymbolID]hir.Type{}
	for _, function := range result.MIR.Functions {
		results[function.Symbol] = function.ResultType
	}
	for _, function := range result.MIR.Functions {
		for _, block := range function.Blocks {
			call := block.Terminator
			if call.Kind == mir.TerminatorCall && call.Span.Source == initializer.Source && call.Span.StartByte == initializer.StartByte && call.Span.EndByte == initializer.EndByte && results[call.Callee] == hir.Option(scalar) {
				return
			}
		}
	}
	t.Fatal("restored MIR lost the current payload-choice initializer/type")
}

func TestCompositeOptionCacheMemoryAndDisk(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, compositeOptionCacheLibrary)
			main := compositeOptionCacheNoneSource()
			writeFile(t, filepath.Join(root, "src", "main.fy"), main)
			config := compositeOptionCacheConfiguration(t, storage, t.TempDir())
			session := newCompositeOptionCacheSession(t, config)
			workspace, cold := checkCompositeOptionCache(t, session, root, false)
			if cold.Cache.Misses != 2 || cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("cold: %+v", cold.Cache)
			}
			assertWorkspaceResult(t, workspace, cold.Compiler, 42)
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newCompositeOptionCacheSession(t, config)
			}
			lower := session.lower
			forbidCompositeOptionCacheLowering(t, session)
			workspace, warm := checkCompositeOptionCache(t, session, root, true)
			assertWorkspaceResult(t, workspace, warm.Compiler, 42)
			assertCompositeOptionCachePayloadChoice(t, warm.Compiler, hir.TypeI8)
			if !reflect.DeepEqual(cold.Compiler.GenericProvenance.Declarations(), warm.Compiler.GenericProvenance.Declarations()) || !reflect.DeepEqual(cold.Compiler.GenericProvenance.Specializations(), warm.Compiler.GenericProvenance.Specializations()) {
				t.Fatal("warm Option provenance differs from current cold source")
			}
			// Restore the real lowerer only for deliberately changed current source.
			session.lower = lower
			main += "\n// current root source edit\n"
			writeFile(t, filepath.Join(root, "src", "main.fy"), main)
			_, sourceEdit := checkCompositeOptionCache(t, session, root, false)
			if sourceEdit.Cache.Misses < 1 || !sourceEdit.Cache.Lowered || sourceEdit.Cache.Outcome == OutcomeAllHit {
				t.Fatalf("source edit reused stale artifact: %+v", sourceEdit.Cache)
			}
			library := strings.Replace(compositeOptionCacheLibrary, "return replaced;", "return copied;", 1)
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), library)
			_, helperEdit := checkCompositeOptionCache(t, session, root, false)
			if helperEdit.Cache.Misses != 2 || !helperEdit.Cache.Lowered || reflect.DeepEqual(warm.Compiler.GenericProvenance.Specializations(), helperEdit.Compiler.GenericProvenance.Specializations()) {
				t.Fatalf("public helper edit reused old closure: %+v", helperEdit.Cache)
			}
			main = strings.Replace(main, "Option<i8> payload_choice=a::empty<i8>();discard a::relay<i8>(payload_choice);", "Option<i16> payload_choice=a::empty<i16>();discard a::relay<i16>(payload_choice);", 1)
			writeFile(t, filepath.Join(root, "src", "main.fy"), main)
			workspace, payloadEdit := checkCompositeOptionCache(t, session, root, false)
			if payloadEdit.Cache.Misses < 1 || !payloadEdit.Cache.Lowered {
				t.Fatalf("payload edit reused old source: %+v", payloadEdit.Cache)
			}
			assertWorkspaceResult(t, workspace, payloadEdit.Compiler, 42)
			assertCompositeOptionCachePayloadChoice(t, payloadEdit.Compiler, hir.TypeI16)
			forbidCompositeOptionCacheLowering(t, session)
			_, payloadWarm := checkCompositeOptionCache(t, session, root, true)
			assertCompositeOptionCachePayloadChoice(t, payloadWarm.Compiler, hir.TypeI16)
			session.lower = lower
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), strings.Replace(library, "return copied;", "return false;", 1))
			invalid, err := session.Check(loadCacheWorkspace(t, root))
			if err != nil || invalid.Compiler.Verified || invalid.Cache.Outcome != OutcomeNotEligible || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || len(invalid.Compiler.GenericProvenance.Declarations()) != 0 {
				t.Fatalf("invalid current Option source reused/published artifacts: %v %+v", err, invalid.Cache)
			}
			t.Logf("%s cold=%+v source=%+v helper=%+v payload=%+v; current-source warm hits forbid lowering", storage, cold.Cache, sourceEdit.Cache, helperEdit.Cache, payloadEdit.Cache)
		})
	}
}

func compositeOptionCacheNativeSource() (string, string) {
	var source, body, observer strings.Builder
	source.WriteString("module main;import answer as a;")
	observer.WriteString("#include <stdint.h>\n#include <string.h>\nstatic int32_t observed;\n")
	values := []struct{ fy, c, value string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "(-INTPTR_MAX-1)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"},
		{"f32", "float", "1.25f"}, {"f32", "float", "0.0f"}, {"f32", "float", "-0.0f"},
		{"f64", "double", "-3.5"}, {"f64", "double", "0.0"}, {"f64", "double", "-0.0"},
	}
	for i, value := range values {
		fmt.Fprintf(&source, `@repr(C) struct P%[1]d{%[2]s value;}unsafe extern C fn P%[1]d seed%[1]d();unsafe extern C fn i32 observe%[1]d(P%[1]d value);`, i, value.fy)
		fmt.Fprintf(&body, `let P%[1]d p%[1]d=unsafe{seed%[1]d()};let Option<%[2]s> original%[1]d=a::pack<%[2]s>(p%[1]d.value);let Option<%[2]s> copied%[1]d=a::relay<%[2]s>(original%[1]d);assert(a::tag<%[2]s>(original%[1]d)==1);assert(a::tag<%[2]s>(copied%[1]d)==1);let %[2]s x%[1]d=a::unpack<%[2]s>(original%[1]d,p%[1]d.value);let %[2]s y%[1]d=a::unpack<%[2]s>(copied%[1]d,p%[1]d.value);assert(unsafe{observe%[1]d(P%[1]d{.value=x%[1]d})}==1);assert(unsafe{observe%[1]d(P%[1]d{.value=y%[1]d})}==1);assert(a::tag<%[2]s>(a::empty<%[2]s>())==0);`, i, value.fy)
		fmt.Fprintf(&observer, "typedef struct{%s value;} P%d;P%d seed%d(void){P%d p={%s};return p;}int32_t observe%d(P%d p){%s expected=%s;observed++;return memcmp(&p.value,&expected,sizeof expected)==0;}\n", value.c, i, i, i, i, value.value, i, i, value.c, value.value)
	}
	source.WriteString("unsafe extern C fn i32 count();fn i32 main(){" + body.String() + "assert(unsafe{count()}==22);return 42;}")
	observer.WriteString("int32_t count(void){return observed;}\n")
	return source.String(), observer.String()
}

func assertCompositeOptionCacheNative(t *testing.T, workspace workspaceloader.Workspace, result Result) {
	t.Helper()
	t.Run("windows-native", func(t *testing.T) {
		if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
			t.Skip("actual payload execution requires Windows amd64")
		}
		toolchain, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		root, err := workspace.RootPackageID()
		if err != nil {
			t.Fatal(err)
		}
		unit, err := cbackend.Emit(result.Compiler.MIR, hir.QualifiedModulePath(root, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		runtimeSource, err := cruntime.ScalarSource(192)
		if err != nil {
			t.Fatal(err)
		}
		_, observer := compositeOptionCacheNativeSource()
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.Source, runtimeSource+"\n"+observer)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
				if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("restored Option payloads: %v %+v", err, outcome)
				}
				t.Logf("MSVC %s C11 %s: %d bytes, exit42 empty streams; eleven payloads/22 active-byte matches, Some/None/copy/reassign", toolchain.Version, profile.Name, len(artifact.Bytes))
			})
		}
	})
}

func TestCompositeOptionCacheNativeRestore(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			root := writeCacheWorkspace(t, compositeOptionCacheLibrary)
			source, _ := compositeOptionCacheNativeSource()
			writeFile(t, filepath.Join(root, "src", "main.fy"), source)
			cacheRoot := t.TempDir()
			config := compositeOptionCacheConfiguration(t, storage, cacheRoot)
			session := newCompositeOptionCacheSession(t, config)
			_, cold := checkCompositeOptionCache(t, session, root, false)
			if cold.Cache.Stored != 2 || !cold.Cache.Lowered {
				t.Fatalf("native cold: %+v", cold.Cache)
			}
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newCompositeOptionCacheSession(t, config)
			}
			forbidCompositeOptionCacheLowering(t, session)
			workspace, warm := checkCompositeOptionCache(t, session, root, true)
			if storage == StorageMemory {
				assertCompositeOptionCacheNative(t, workspace, warm)
				return
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			// The same exact test executable has a fresh address space and no memory
			// artifacts. Its helper derives build identity from that executable.
			command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCompositeOptionCacheChild$", "-test.count=1", "-test.v")
			command.Env = append(os.Environ(), "FAIRYC_COMPOSITE_OPTION_CACHE_CHILD="+root, "FAIRYC_COMPOSITE_OPTION_CACHE_STORE="+cacheRoot)
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "composite-option-new-process-disk-42") {
				t.Fatalf("new-process Option restore: %v %s", err, output)
			}
			t.Logf("fresh child: %s", output)
		})
	}
}

func TestCompositeOptionCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_COMPOSITE_OPTION_CACHE_CHILD")
	if root == "" {
		t.Skip("fresh-process consumer only")
	}
	session := newCompositeOptionCacheSession(t, genericPersistentConfiguration(t, os.Getenv("FAIRYC_COMPOSITE_OPTION_CACHE_STORE")))
	forbidCompositeOptionCacheLowering(t, session)
	workspace, result := checkCompositeOptionCache(t, session, root, true)
	assertCompositeOptionCacheNative(t, workspace, result)
	fmt.Println("composite-option-new-process-disk-42")
}

// Preserve the actual disk-envelope spelling/ordering while corrupting the
// inner stable artifact. The renamed generation has a valid outer checksum,
// so this exercises artifact rejection rather than only a torn JSON record.
type compositeOptionDiskEnvelope struct {
	Schema  string `json:"schema"`
	Version string `json:"version"`
	Record  struct {
		Key             packagecache.Key     `json:"key"`
		PublicInterface moduleinterface.Hash `json:"publicInterface"`
		Artifacts       []struct {
			Kind string `json:"kind"`
			Data []byte `json:"data"`
		} `json:"artifacts"`
	} `json:"record"`
}

func TestCompositeOptionCacheCorruptArtifactRepair(t *testing.T) {
	root := writeCacheWorkspace(t, compositeOptionCacheLibrary)
	writeFile(t, filepath.Join(root, "src", "main.fy"), compositeOptionCacheNoneSource())
	cacheRoot := t.TempDir()
	config := genericPersistentConfiguration(t, cacheRoot)
	session := newCompositeOptionCacheSession(t, config)
	_, cold := checkCompositeOptionCache(t, session, root, false)
	if cold.Cache.Stored != 2 {
		t.Fatalf("repair cold: %+v", cold.Cache)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	var records []string
	if err := filepath.WalkDir(cacheRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".json" {
			records = append(records, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("cold record count: %d", len(records))
	}
	raw, err := os.ReadFile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	var envelope compositeOptionDiskEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(envelope)
	if err != nil || !reflect.DeepEqual(raw, canonical) || len(envelope.Record.Artifacts) != 1 {
		t.Fatal("test disk envelope differs from actual canonical record", err)
	}
	envelope.Record.Artifacts[0].Data = []byte("corrupt Option stable artifact")
	damaged, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(damaged)
	corruptPath := filepath.Join(filepath.Dir(records[0]), hex.EncodeToString(digest[:])+".json")
	if err := os.WriteFile(corruptPath, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(records[0]); err != nil {
		t.Fatal(err)
	}
	repair := newCompositeOptionCacheSession(t, config)
	workspace, repaired := checkCompositeOptionCache(t, repair, root, false)
	if repaired.Cache.Rejected != 1 || repaired.Cache.Hits != 1 || !repaired.Cache.Lowered || repaired.Cache.Stored != 2 || repaired.Cache.Outcome == OutcomeAllHit {
		t.Fatalf("corrupt artifact did not repair from source: %+v", repaired.Cache)
	}
	assertWorkspaceResult(t, workspace, repaired.Compiler, 42)
	if _, err := os.Stat(corruptPath); !os.IsNotExist(err) {
		t.Fatalf("rejected generation retained: %v", err)
	}
	if err := repair.Close(); err != nil {
		t.Fatal(err)
	}
	reader := newCompositeOptionCacheSession(t, config)
	forbidCompositeOptionCacheLowering(t, reader)
	workspace, restored := checkCompositeOptionCache(t, reader, root, true)
	assertWorkspaceResult(t, workspace, restored.Compiler, 42)
	t.Logf("valid outer checksum, rejected inner artifact: %+v; fresh reader restored two repaired packages without lowering", repaired.Cache)
}
