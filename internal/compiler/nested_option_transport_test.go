package compiler_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const nestedOptionHelpers = `
fn Option<Option<T>> nested<T>(T value,i32 mode){if(mode==0){return .none;}if(mode==1){return .some(.none);}return .some(.some(value));}
fn Option<Option<T>> relay_nested<T>(Option<Option<T>> value){let Option<Option<T>> copied=value;return copied;}
fn i32 state_nested<T>(Option<Option<T>> value){return switch(value){case .none:{0}case .some(inner):{switch(inner){case .none:{1}case .some(_):{2}}}};}
fn T choose_nested<T>(Option<Option<T>> value,T fallback){return switch(value){case .none:{fallback}case .some(inner):{switch(inner){case .none:{fallback}case .some(actual):{actual}}}};}
`

func TestNestedOptionTransportAdmission(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		t.Run(scalar, func(t *testing.T) {
			text := "module main;" + nestedOptionHelpers + fmt.Sprintf(`pub fn %[1]s forward(%[1]s value,%[1]s fallback,i32 mode){let Option<Option<%[1]s>> original=nested<%[1]s>(value,mode);let Option<Option<%[1]s>> copied=relay_nested<%[1]s>(original);return choose_nested<%[1]s>(copied,fallback);}fn i32 main(){return 42;}`, scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			before, _ := json.Marshal(file.Module)
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || !result.Verified {
				t.Fatalf("nested option: %v %+v", err, result.Semantic)
			}
			after, _ := json.Marshal(file.Module)
			if string(before) != string(after) {
				t.Fatal("nested option changed authored AST")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(result.MIR, "main::main", profile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestNestedOptionTransportNative(t *testing.T) {
	var source, body, observer strings.Builder
	source.WriteString("module main;" + nestedOptionHelpers)
	observer.WriteString("#include <stdint.h>\n#include <string.h>\nstatic int32_t calls;\n")
	values := []struct{ fy, c, value, fallback string }{
		{"i8", "int8_t", "INT8_MIN", "INT8_MAX"}, {"i16", "int16_t", "INT16_MIN", "INT16_MAX"},
		{"isize", "intptr_t", "(-INTPTR_MAX-1)", "INTPTR_MAX"}, {"u16", "uint16_t", "UINT16_MAX", "0"}, {"u64", "uint64_t", "UINT64_MAX", "0"},
		{"f32", "float", "-0.0f", "1.25f"}, {"f32", "float", "0.0f", "-3.5f"}, {"f32", "float", "1.25f", "0.0f"},
		{"f64", "double", "-0.0", "-3.5"}, {"f64", "double", "0.0", "1.25"}, {"f64", "double", "-3.5", "0.0"},
	}
	for i, v := range values {
		fmt.Fprintf(&source, `@repr(C) struct Pair%[1]d{%[2]s value;%[2]s fallback;}unsafe extern C fn Pair%[1]d seed%[1]d();unsafe extern C fn i32 observe%[1]d(Pair%[1]d value,i32 mode);`, i, v.fy)
		fmt.Fprintf(&body, `let Pair%[1]d seed%[1]d=unsafe{seed%[1]d()};var i32 mode%[1]d=0;while(mode%[1]d<3){let Option<Option<%[2]s>> original=nested<%[2]s>(seed%[1]d.value,mode%[1]d);let Option<Option<%[2]s>> copied=relay_nested<%[2]s>(original);assert(state_nested<%[2]s>(original)==mode%[1]d && state_nested<%[2]s>(copied)==mode%[1]d);let %[2]s first=choose_nested<%[2]s>(original,seed%[1]d.fallback);let %[2]s second=choose_nested<%[2]s>(copied,seed%[1]d.fallback);assert(unsafe{observe%[1]d(Pair%[1]d{.value=first,.fallback=second},mode%[1]d)}==1);mode%[1]d+=1;}`, i, v.fy)
		fmt.Fprintf(&observer, "typedef struct{%[1]s value,fallback;}Pair%[2]d;Pair%[2]d seed%[2]d(void){Pair%[2]d p={%[3]s,%[4]s};return p;}int32_t observe%[2]d(Pair%[2]d p,int32_t mode){%[1]s expected=mode==2?(%[1]s)(%[3]s):(%[1]s)(%[4]s);calls++;return memcmp(&p.value,&expected,sizeof expected)==0&&memcmp(&p.fallback,&expected,sizeof expected)==0;}\n", v.c, i, v.value, v.fallback)
	}
	source.WriteString("unsafe extern C fn i32 observations();fn i32 main(){" + body.String() + "assert(unsafe{observations()}==33);return 42;}")
	observer.WriteString("int32_t observations(void){return calls;}\n")
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source.String())
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("nested option native: %v %+v", err, result.Semantic)
	}
	t.Run("source", func(t *testing.T) { runExpandedLanguageNative(t, result, "main::main", observer.String()) })
	encoded, err := mircodec.Encode(result.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(encoded, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("restored", func(t *testing.T) {
		runExpandedLanguageNative(t, compiler.Result{MIR: restored}, "main::main", observer.String())
	})
}
