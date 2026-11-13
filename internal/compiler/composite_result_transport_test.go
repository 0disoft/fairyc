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
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

var compositeResultScalars = []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"}

const compositeResultHelpers = `
fn Result<T,u32> pack_result<T>(T value){return .ok(value);}
fn Result<T,u32> relay_result<T>(Result<T,u32> value){
 let Result<T,u32> copied=value;
 discard value;
 var Result<T,u32> replaced=.err(9);
 discard replaced;
 replaced=copied;
 discard copied;
 return replaced;
}
fn Result<T,E> error_binding<T,E>(Result<T,E> value){return value;}
fn Result<T,u32> propagate_result<T>(Result<T,u32> value){let T actual=value?;return .ok(actual);}
fn T unpack_result<T>(Result<T,u32> value,T fallback){return switch(value){case .ok(actual):{actual}case .err(_):{fallback}};}
fn i32 error_branch<T>(){let Result<T,u32> value=.err(77);return switch(value){case .ok(_):{0}case .err(error):{if(error==77){42}else{0}}};}
`

func TestCompositeResultAdmissionAndSourceImmutability(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		t.Run(scalar, func(t *testing.T) {
			text := "module main;" + compositeResultHelpers + fmt.Sprintf(`
fn Result<%[1]s,u32> written(Result<%[1]s,u32> value){return value;}
fn Result<%[1]s,u32> written_generic<T>(Result<%[1]s,u32> value){return value;}
fn i32 main(){let Result<%[1]s,u32> value=.err(77);discard written(written_generic<i32>(error_binding<%[1]s,u32>(propagate_result<%[1]s>(relay_result<%[1]s>(value)))));return error_branch<%[1]s>();}`, scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			before, _ := json.Marshal(file.Module)
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
			after, _ := json.Marshal(file.Module)
			if string(before) != string(after) {
				t.Fatal("source AST changed")
			}
		})
	}
}

func TestCompositeResultMustHandleEachCopy(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		for _, tc := range []struct{ name, body, primary string }{
			{"copy pending", "let Result<S,u32> copied=original;discard original;", "original"},
			{"later original evaluation", "let Result<S,u32> copied=original;discard copied;original;", "original"},
		} {
			t.Run(scalar+"/"+tc.name, func(t *testing.T) {
				text := strings.ReplaceAll("module main;fn void pending(Result<S,u32> original){"+tc.body+"}", "S", scalar)
				file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
				result, err := compiler.CheckExplicitFiles([]semantic.File{file})
				if err != nil || result.Verified || len(result.Semantic.Diagnostics) != 1 || result.Semantic.Diagnostics[0].Code != "E4101" {
					t.Fatalf("must-handle copy: %v diagnostics=%+v unsupported=%+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
				}
				d := result.Semantic.Diagnostics[0]
				if text[d.Primary.StartByte:d.Primary.EndByte] != tc.primary {
					t.Fatalf("must-handle origin: %+v", d)
				}
				want := strings.Index(text, "=original") + 1
				if tc.name == "later original evaluation" {
					want = strings.LastIndex(text, "original;")
				}
				if d.Primary.StartByte != want {
					t.Fatalf("Result evaluation producer changed: %+v want byte %d", d, want)
				}
			})
		}
	}
}

func TestCompositeResultErrorMismatchOrigin(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		t.Run(scalar, func(t *testing.T) {
			text := strings.ReplaceAll(`module main; // 한글
fn Result<S,i32> mismatch(Result<S,u32> value){let S actual=value?;return .ok(actual);}`, "S", scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || result.Verified || len(result.Semantic.Diagnostics) != 1 {
				t.Fatalf("error mismatch: %v %+v", err, result.Semantic)
			}
			d := result.Semantic.Diagnostics[0]
			if d.Code != "E4103" || text[d.Primary.StartByte:d.Primary.EndByte] != "?" {
				t.Fatalf("error mismatch origin: %+v", d)
			}
			related := map[string]string{}
			for _, span := range d.Related {
				related[span.Role] = text[span.Span.StartByte:span.Span.EndByte]
			}
			if related["actual-error"] != "u32" || related["expected-error"] != "i32" {
				t.Fatalf("error type origins: %+v", d)
			}
		})
	}
}

func TestCompositeResultKeepsOtherGenericShapesClosed(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		for _, shape := range []string{"Result<T,i64>", "Result<u32,T>", "Result<T,T>", "Result<Option<T>,u32>", "Result<Result<T,u32>,u32>", "Option<Result<T,u32>>", "ref Result<T,u32>", "ptr Result<T,u32>", "Result<T,u32>[2]"} {
			t.Run(scalar+"/"+shape, func(t *testing.T) {
				concrete := strings.ReplaceAll(shape, "T", scalar)
				text := fmt.Sprintf("module main;fn %[1]s id<T>(%[1]s value){return value;}fn %[2]s forward(%[2]s value){discard id<%[3]s>(value);return id<%[3]s>(value);}", shape, concrete, scalar)
				file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
				before, _ := json.Marshal(file.Module)
				result, err := compiler.CheckExplicitFiles([]semantic.File{file})
				if err != nil || result.Verified || len(result.Semantic.Unsupported) != 2 || len(result.Semantic.Diagnostics) != 0 {
					t.Fatalf("closed generic shape: %v %+v", err, result.Semantic)
				}
				for _, row := range result.Semantic.Unsupported {
					if row.Feature != "type "+shape || text[row.Span.StartByte:row.Span.EndByte] != shape {
						t.Fatalf("original signature spelling/span: %+v", row)
					}
				}
				after, _ := json.Marshal(file.Module)
				if string(before) != string(after) {
					t.Fatal("rejected signature mutated AST")
				}
			})
		}
	}
}

func TestCompositeResultKeepsCompositeArgumentsClosed(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		text := fmt.Sprintf("module main;fn T id<T>(T value){return value;}fn i32 main(){let Result<%[1]s,u32> value=.err(1);discard id<Result<%[1]s,u32>>(value);return 42;}", scalar)
		result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err == nil && result.Verified {
			t.Fatalf("composite generic argument admitted: %s", scalar)
		}
	}
}

func TestCompositeResultEmitterKeepsOtherOrdinaryWrappersClosed(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		for _, shape := range []string{"Result<S,i64>", "Result<u32,S>", "Result<Option<S>,u32>", "Result<Result<S,u32>,u32>", "Option<Result<S,u32>>"} {
			t.Run(scalar+"/"+shape, func(t *testing.T) {
				concrete := strings.ReplaceAll(shape, "S", scalar)
				value := ".err(1)"
				if shape == "Result<u32,S>" {
					value = ".ok(1)"
				}
				finish := "return switch(value){case .ok(_):{42}case .err(_):{42}};"
				if shape == "Result<Result<S,u32>,u32>" {
					finish = "return switch(value){case .ok(inner):{discard inner;42}case .err(_):{42}};"
				}
				if strings.HasPrefix(shape, "Option") {
					value = ".none"
					finish = "return 42;"
				}
				text := fmt.Sprintf("module main;fn %s choose(){return %s;}fn i32 main(){let %s value=choose();%s}", concrete, value, concrete, finish)
				result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
				if err != nil || !result.Verified {
					t.Fatalf("ordinary semantics: %v %+v", err, result.Semantic)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(result.MIR, "main::main", profile); err == nil {
						t.Fatal("unadmitted ordinary wrapper emitted")
					}
				}
			})
		}
	}
}

func compositeResultNativeProgram() (string, string) {
	var src, body, observer strings.Builder
	src.WriteString("module main;" + compositeResultHelpers)
	observer.WriteString("#include <stdint.h>\n#include <string.h>\nstatic int32_t observed,marked,seeds;\nvoid mark(void){marked++;}\n")
	src.WriteString("unsafe extern C fn void mark();unsafe extern C fn i32 observations();unsafe extern C fn i32 marks();unsafe extern C fn i32 seed_count();")
	values := []struct{ fy, c, value string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "(-INTPTR_MAX-1)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"},
		{"f32", "float", "1.25f"}, {"f32", "float", "0.0f"}, {"f32", "float", "-0.0f"},
		{"f64", "double", "-3.5"}, {"f64", "double", "0.0"}, {"f64", "double", "-0.0"},
	}
	for i, v := range values {
		fmt.Fprintf(&src, `@repr(C) struct P%[1]d{%[2]s value;}unsafe extern C fn P%[1]d seed%[1]d();unsafe extern C fn i32 observe%[1]d(P%[1]d value);
fn Result<%[2]s,u32> written%[1]d(Result<%[2]s,u32> value){return value;}
fn Result<%[2]s,u32> counted%[1]d(bool fail){unsafe{mark();}if(fail){return .err(77);}let P%[1]d input=unsafe{seed%[1]d()};return .ok(input.value);}
fn Result<%[2]s,u32> forward%[1]d(bool fail){let %[2]s actual=propagate_result<%[2]s>(counted%[1]d(fail))?;return .ok(actual);}`, i, v.fy)
		fmt.Fprintf(&body, `let P%[1]d p%[1]d=unsafe{seed%[1]d()};
let Result<%[2]s,u32> original%[1]d=pack_result<%[2]s>(p%[1]d.value);
let Result<%[2]s,u32> copied%[1]d=written%[1]d(error_binding<%[2]s,u32>(relay_result<%[2]s>(original%[1]d)));
let %[2]s a%[1]d=unpack_result<%[2]s>(original%[1]d,p%[1]d.value);
let %[2]s b%[1]d=unpack_result<%[2]s>(copied%[1]d,p%[1]d.value);
let %[2]s c%[1]d=unpack_result<%[2]s>(forward%[1]d(false),p%[1]d.value);
assert(unsafe{observe%[1]d(P%[1]d{.value=a%[1]d})}==1);
assert(unsafe{observe%[1]d(P%[1]d{.value=b%[1]d})}==1);
assert(unsafe{observe%[1]d(P%[1]d{.value=c%[1]d})}==1);
let Result<%[2]s,u32> failed%[1]d=forward%[1]d(true);
let u32 error%[1]d=switch(failed%[1]d){case .ok(_):{0}case .err(code):{code}};
assert(error%[1]d==77);assert(error_branch<%[2]s>()==42);`, i, v.fy)
		fmt.Fprintf(&observer, "typedef struct{%s value;} P%d;P%d seed%d(void){P%d p={%s};seeds++;return p;}int32_t observe%d(P%d p){%s expected=%s;observed++;return memcmp(&p.value,&expected,sizeof expected)==0;}\n", v.c, i, i, i, i, v.value, i, i, v.c, v.value)
	}
	fmt.Fprintf(&src, "fn i32 main(){%s assert(unsafe{observations()}==33);assert(unsafe{marks()}==22);assert(unsafe{seed_count()}==22);return 42;}", body.String())
	observer.WriteString("int32_t observations(void){return observed;}int32_t marks(void){return marked;}int32_t seed_count(void){return seeds;}\n")
	return src.String(), observer.String()
}

func TestCompositeResultNativePayloadsAndPropagationOnce(t *testing.T) {
	text, observer := compositeResultNativeProgram()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	before, _ := json.Marshal(file.Module)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("Result native source: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("native fixture AST mutated")
	}
	t.Run("source", func(t *testing.T) {
		runExpandedLanguageNative(t, result, "main::main", observer)
	})
	encoded, err := mircodec.Encode(result.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(encoded, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("restored", func(t *testing.T) {
		// Both executions use authored C expected values/counters. Neither
		// run's output supplies the expected values for the other run.
		runExpandedLanguageNative(t, compiler.Result{MIR: restored}, "main::main", observer)
	})
}

func TestCompositeResultKeepsDeferAndStorageClosed(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		t.Run(scalar+"/defer", func(t *testing.T) {
			text := fmt.Sprintf("module main;fn void consume(Result<%[1]s,u32> value){discard value;}fn i32 main(){let Result<%[1]s,u32> value=.err(1);defer consume(value);discard value;return 42;}", scalar)
			result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || result.Verified || len(result.Semantic.Unsupported) == 0 {
				t.Fatalf("Result defer capture admission widened: %v %+v", err, result.Semantic)
			}
			found := false
			for _, u := range result.Semantic.Unsupported {
				if strings.HasPrefix(u.Feature, "defer ") {
					found = true
					if text[u.Span.StartByte:u.Span.EndByte] != "defer consume(value);" {
						t.Fatalf("defer source origin: %+v", u)
					}
				}
			}
			if !found {
				t.Fatalf("defer refused for another reason: %+v", result.Semantic)
			}
		})
		t.Run(scalar+"/storage", func(t *testing.T) {
			text := fmt.Sprintf("module main;struct Box{Result<%[1]s,u32> value;}fn Box choose(){return Box{.value=.err(1)};}fn i32 main(){let Box box=choose();discard box;return 42;}", scalar)
			result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !result.Verified {
				t.Fatalf("storage semantic prerequisite: %v %+v", err, result.Semantic)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(result.MIR, "main::main", profile); err == nil {
					t.Fatal("Result struct storage admission widened")
				}
			}
		})
	}
}
