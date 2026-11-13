package compiler_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const compositeOptionHelpers = `
fn Option<T> pack<T>(T value){return .some(value);}
fn Option<T> relay<T>(Option<T> value){
 let Option<T> copied=value;
 var Option<T> replaced=.none;
 replaced=copied;
 discard value;
 return replaced;
}
fn T unpack<T>(Option<T> value,T fallback){
 return switch(value){case .some(actual):{actual}case .none:{fallback}};
}
fn i32 none_branch<T>(){
 let Option<T> value=.none;
 return switch(value){case .some(_):{0}case .none:{42}};
}
fn i32 some_branch<T>(Option<T> value){
 return switch(value){case .some(_):{1}case .none:{0}};
}
`

func TestCompositeOptionAdmissionAndSourceImmutability(t *testing.T) {
	for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		t.Run(scalar, func(t *testing.T) {
			// Ordinary named payloads and binder substitution share value transport.
			text := "module main;" + compositeOptionHelpers + fmt.Sprintf(`
fn Option<%[1]s> written(Option<%[1]s> value){return value;}
fn Option<%[1]s> written_generic<T>(Option<%[1]s> value){return value;}
fn i32 main(){let Option<%[1]s> value=.none;discard written(written_generic<i32>(relay<%[1]s>(value)));return none_branch<%[1]s>();}`, scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			before, err := json.Marshal(file.Module)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"private", "explicit", "package"} {
				t.Run(mode, func(t *testing.T) {
					var result compiler.Result
					var err error
					switch mode {
					case "private":
						result, _, err = compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
					case "explicit":
						result, err = compiler.CheckExplicitFiles([]semantic.File{file})
					case "package":
						input, coordinates := genericPackageInput(t)
						packageFile := file
						packageFile.Package = 1
						input.Files = []semantic.File{packageFile}
						input.Imports = nil
						result, _, err = compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
					}
					if err != nil || !result.Verified {
						t.Fatalf("%s: %v %+v", mode, err, result.Semantic)
					}
					if failures := mir.Verify(result.MIR); len(failures) != 0 {
						t.Fatal(failures)
					}
					entry := "main::main"
					if mode == "package" {
						entry = hir.QualifiedModulePath(1, "main") + "::main"
					}
					for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
						if _, err := cbackend.Emit(result.MIR, entry, profile); err != nil {
							t.Fatal(err)
						}
					}
				})
			}
			after, err := json.Marshal(file.Module)
			if err != nil || string(before) != string(after) {
				t.Fatalf("source AST changed: %v", err)
			}
		})
	}
}

func TestCompositeOptionKeepsCompositeArgumentsClosed(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;fn T id<T>(T value){return value;}fn i32 main(){let Option<u16> value=.none;discard id<Option<u16>>(value);return 42;}`)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err == nil && result.Verified {
		t.Fatal("composite generic argument admitted")
	}
}

func TestCompositeOptionEmitterKeepsOtherOrdinaryWrappersClosed(t *testing.T) {
	for _, scalar := range []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		for _, shape := range []string{"Option<Option<S>>", "Result<S,i64>", "Result<Option<S>,i32>", "Option<Result<S,i32>>"} {
			t.Run(scalar+"/"+shape, func(t *testing.T) {
				concrete := strings.ReplaceAll(shape, "S", scalar)
				value := ".none"
				finish := "return 42;"
				if strings.HasPrefix(shape, "Result") {
					value = ".err(1)"
					finish = "return switch(value){case .ok(_):{42}case .err(_):{42}};"
				}
				text := fmt.Sprintf("module main;fn %s choose(){return %s;}fn i32 main(){let %s value=choose();%s}", concrete, value, concrete, finish)
				file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
				result, err := compiler.CheckExplicitFiles([]semantic.File{file})
				if err != nil || !result.Verified {
					t.Fatalf("ordinary semantic prerequisite: %v %+v", err, result.Semantic)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(result.MIR, "main::main", profile); err == nil {
						t.Fatal("unsupported ordinary wrapper emitted")
					}
				}
			})
		}
	}
}

func compositeOptionNativeProgram() (string, string) {
	var src, body, observer strings.Builder
	src.WriteString("module main;" + compositeOptionHelpers)
	observer.WriteString("#include <stdint.h>\n#include <string.h>\nstatic int32_t observed;\n")
	values := []struct{ fy, c, value string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "(-INTPTR_MAX-1)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"},
		{"f32", "float", "1.25f"}, {"f32", "float", "0.0f"}, {"f32", "float", "-0.0f"},
		{"f64", "double", "-3.5"}, {"f64", "double", "0.0"}, {"f64", "double", "-0.0"},
	}
	for i, v := range values {
		fmt.Fprintf(&src, `@repr(C) struct P%[1]d{%[2]s value;}unsafe extern C fn P%[1]d seed%[1]d();unsafe extern C fn i32 observe%[1]d(P%[1]d value);fn Option<%[2]s> written%[1]d(Option<%[2]s> value){return value;}`, i, v.fy)
		fmt.Fprintf(&body, `let P%[1]d p%[1]d=unsafe{seed%[1]d()};let Option<%[2]s> original%[1]d=pack<%[2]s>(p%[1]d.value);let Option<%[2]s> copied%[1]d=written%[1]d(relay<%[2]s>(original%[1]d));assert(some_branch<%[2]s>(original%[1]d)==1);assert(some_branch<%[2]s>(copied%[1]d)==1);let %[2]s a%[1]d=unpack<%[2]s>(original%[1]d,p%[1]d.value);let %[2]s b%[1]d=unpack<%[2]s>(copied%[1]d,p%[1]d.value);assert(unsafe{observe%[1]d(P%[1]d{.value=a%[1]d})}==1);assert(unsafe{observe%[1]d(P%[1]d{.value=b%[1]d})}==1);assert(none_branch<%[2]s>()==42);`, i, v.fy)
		fmt.Fprintf(&observer, "typedef struct{%s value;} P%d;P%d seed%d(void){P%d p={%s};return p;}int32_t observe%d(P%d p){%s expected=%s;observed++;return memcmp(&p.value,&expected,sizeof expected)==0;}\n", v.c, i, i, i, i, v.value, i, i, v.c, v.value)
	}
	src.WriteString("unsafe extern C fn i32 count();fn i32 main(){" + body.String() + "assert(unsafe{count()}==22);return 42;}")
	observer.WriteString("int32_t count(void){return observed;}\n")
	return src.String(), observer.String()
}

func TestCompositeOptionNativePayloads(t *testing.T) {
	text, observer := compositeOptionNativeProgram()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("Option native source: %v %+v", err, result.Semantic)
	}
	// Existing helper runs real Windows low/optimized C11/C17 consumers serially;
	// on Linux it uses the existing compiler/profile matrix when the batch runs.
	runExpandedLanguageNative(t, result, "main::main", observer)
}
