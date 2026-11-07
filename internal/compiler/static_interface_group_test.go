package compiler_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

const groupedStaticSource = `module main;
@interface(Arithmetic) pub fn T add<T>(T left,T right){}
@interface(Arithmetic) pub fn U identity<U>(U value){}
@implements(add,i32) fn i32 add32(i32 left,i32 right){return left+right;}
@implements(identity,i32) fn i32 identity32(i32 value){return value;}
@implements(add,i64) fn i64 add64(i64 left,i64 right){return left+right;}
@implements(identity,i64) fn i64 identity64(i64 value){return value;}
pub fn T algorithm<T>(T value){return identity<T>(add<T>(value,value));}
fn i32 main(){assert(algorithm<i64>(21)==42);return algorithm<i32>(21);}`

func TestGroupedStaticInterfaceExecution(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", groupedStaticSource)
	before, _ := json.Marshal(file)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	out, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("interpret: %+v %v", out, err)
	}
	after, _ := json.Marshal(file)
	if string(before) != string(after) {
		t.Fatal("grouped compilation mutated the written AST")
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}

func groupedStaticDeclarations(count int, label string) string {
	var src strings.Builder
	src.WriteString("module main; // 한글\r\n")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&src, "@interface(%s) pub fn T op%02d<T>(T value){}\r\n", label, i)
	}
	return src.String()
}

// Invalid declarations must publish diagnostics only, with original byte spans.
func TestGroupedStaticInterfaceDiagnostics(t *testing.T) {
	base := groupedStaticDeclarations(2, "G")
	provider0 := `@implements(op00,i32) fn i32 first(i32 value){return value;}`
	provider1 := `@implements(op01,i32) fn i32 second(i32 value){return value;}`
	for _, tc := range []struct{ name, text, primary, code string }{
		{"singleton", groupedStaticDeclarations(1, "G"), "@interface(G) pub fn T op00<T>(T value){}", "E4501"},
		{"seventeen", groupedStaticDeclarations(17, "G"), "@interface(G) pub fn T op00<T>(T value){}", "E4501"},
		{"label65", groupedStaticDeclarations(2, strings.Repeat("a", 65)), "@interface(" + strings.Repeat("a", 65) + ") pub fn T op00<T>(T value){}", "E4501"},
		{"numeric label", groupedStaticDeclarations(2, "1"), "@interface(1) pub fn T op00<T>(T value){}", "E4501"},
		{"quoted label", groupedStaticDeclarations(2, `"G"`), `@interface("G") pub fn T op00<T>(T value){}`, "E4501"},
		{"extra label", groupedStaticDeclarations(2, "G,H"), "@interface(G,H) pub fn T op00<T>(T value){}", "E4501"},
		{"private", strings.Replace(base, "pub fn", "fn", 1), "@interface(G) fn T op00<T>(T value){}", "E4501"},
		{"body", strings.Replace(base, "(T value){}", "(T value){return value;}", 1), "@interface(G) pub fn T op00<T>(T value){return value;}", "E4501"},
		{"result", strings.Replace(base, "fn T op00", "fn i32 op00", 1), "@interface(G) pub fn i32 op00<T>(T value){}", "E4501"},
		{"five parameters", strings.Replace(base, "op00<T>(T value)", "op00<T>(T a,T b,T c,T d,T e)", 1), "@interface(G) pub fn T op00<T>(T a,T b,T c,T d,T e){}", "E4501"},
		{"two type parameters", strings.Replace(base, "op00<T>", "op00<T,U>", 1), "@interface(G) pub fn T op00<T,U>(T value){}", "E4501"},
		{"unsafe operation", strings.Replace(base, "pub fn T op00", "pub unsafe fn T op00", 1), "@interface(G) pub unsafe fn T op00<T>(T value){}", "E4501"},
		{"extra operation attribute", strings.Replace(base, "pub fn T op00", "@inline pub fn T op00", 1), "@interface(G) @inline pub fn T op00<T>(T value){}", "E4501"},
		{"duplicate operation attribute", strings.Replace(base, "pub fn T op00", "@interface(G) pub fn T op00", 1), "@interface(G) @interface(G) pub fn T op00<T>(T value){}", "E4501"},
		{"partial unused", base + provider0, provider0, "E4501"},
		{"partial other scalar", base + provider0 + provider1 + `@implements(op01,bool) fn bool flag(bool value){return value;}`, `@implements(op01,bool) fn bool flag(bool value){return value;}`, "E4501"},
		{"earliest provider", base + `@implements(op01,i64) fn i64 early(i64 value){return value;}` + provider0, `@implements(op01,i64) fn i64 early(i64 value){return value;}`, "E4501"},
		{"earliest across groups", base + `@interface(H) pub fn T h0<T>(T v){} @interface(H) pub fn T h1<T>(T v){}` + `@implements(h1,bool) fn bool early(bool value){return value;}` + provider0, `@implements(h1,bool) fn bool early(bool value){return value;}`, "E4501"},
		{"group label is not operation", base + `@implements(G,i32) fn i32 bad(i32 value){return value;}`, `@implements(G,i32) fn i32 bad(i32 value){return value;}`, "E4501"},
		{"duplicate before size", groupedStaticDeclarations(1, "G") + provider0 + `@implements(op00,i32) fn i32 duplicate(i32 value){return value;}`, `@implements(op00,i32) fn i32 duplicate(i32 value){return value;}`, "E4501"},
		{"structural before size", groupedStaticDeclarations(1, "G") + `@implements(op00,i32) fn bool wrong(bool value){return value;}`, `@implements(op00,i32) fn bool wrong(bool value){return value;}`, "E4501"},
		{"duplicate before partial", base + provider0 + `@implements(op00,i32) fn i32 duplicate(i32 value){return value;}`, `@implements(op00,i32) fn i32 duplicate(i32 value){return value;}`, "E4501"},
		{"size before partial", groupedStaticDeclarations(1, "G") + provider0, "@interface(G) pub fn T op00<T>(T value){}", "E4501"},
		{"absent all", base + `fn i32 main(){return op00<i32>(42);}`, "op00<i32>", "E4502"},
		{"absent scalar", base + provider0 + provider1 + `fn i64 main(){return op00<i64>(42);}`, "op00<i64>", "E4502"},
		{"nested absent root", base + `fn T relay<T>(T value){return op00<T>(value);} fn i32 main(){return relay<i32>(42);}`, "relay<i32>", "E4502"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := semantic.PackageInput{Files: []semantic.File{parseCompilerFile(t, 7, "main.fy", tc.text)}}
			before, _ := json.Marshal(input)
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: genericCoordinate(t)}
			analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
			r := analysis.Analysis.Semantic
			if err != nil || len(r.Diagnostics) != 1 || analysis.Interfaces != nil || analysis.OrdinaryInterfaces != nil || analysis.Specializations != nil || len(analysis.Plan.Specializations) != 0 || !reflect.DeepEqual(r.HIR, hir.Package{}) {
				t.Fatalf("failure published facts: %v %+v", err, analysis)
			}
			d := r.Diagnostics[0]
			if d.Code != tc.code || d.Primary.Package != 7 || fixtureSpanText(t, tc.text, d.Primary) != tc.primary {
				t.Fatalf("diagnostic: %+v text=%q want=%q", d, fixtureSpanText(t, tc.text, d.Primary), tc.primary)
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("failed preparation mutated AST")
			}
		})
	}
}

func TestGroupedStaticInterfaceBounds(t *testing.T) {
	for _, count := range []int{2, 16} {
		for _, label := range []string{"_", "Arithmetic2", strings.Repeat("a", 64)} {
			t.Run(fmt.Sprintf("members%d/label%d", count, len(label)), func(t *testing.T) {
				text := groupedStaticDeclarations(count, label) + "fn i32 main(){return 42;}"
				r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
				if err != nil || !r.Verified {
					t.Fatalf("unprovided group: %v %+v", err, r.Semantic)
				}
			})
		}
	}
	for _, count := range []int{2, 16} {
		t.Run(fmt.Sprintf("complete%d", count), func(t *testing.T) {
			var text strings.Builder
			text.WriteString(groupedStaticDeclarations(count, "Bundle"))
			for i := 0; i < count; i++ {
				fmt.Fprintf(&text, "@implements(op%02d,i32) fn i32 provider%02d(i32 value){return value;}", i, i)
			}
			fmt.Fprintf(&text, "fn i32 main(){return op%02d<i32>(op00<i32>(42));}", count-1)
			r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text.String())})
			if err != nil || !r.Verified {
				t.Fatalf("complete group: %v %+v", err, r.Semantic)
			}
			out, err := interpreter.Execute(r.MIR, "main::main")
			if err != nil || out.Value != int32(42) {
				t.Fatalf("complete group: %+v %v", out, err)
			}
		})
	}
	// Different groups may be partially populated independently; differing arities
	// and type parameter names are legal inside a complete group.
	text := `module main;
 @interface(Factory) pub fn T seed<T>(){}
 @interface(Factory) pub fn U fourth<U>(U a,U b,U c,U d){}
 @implements(seed,i32) fn i32 seed32(){return 42;}
 @implements(fourth,i32) fn i32 fourth32(i32 a,i32 b,i32 c,i32 d){return d;}
 fn i32 main(){return fourth<i32>(0,1,2,seed<i32>());}`
	r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !r.Verified {
		t.Fatalf("mixed arities: %v %+v", err, r.Semantic)
	}
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("mixed arities: %+v %v", out, err)
	}
}

func TestGroupedStaticInterfaceImport(t *testing.T) {
	text := strings.Replace(groupedStaticSource, "module main;", "module ops;", 1)
	// Keep the generic wrapper in the provider and request it from an importer.
	text = text[:strings.Index(text, "fn i32 main()")]
	files := []semantic.File{
		parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;import ops as o;fn i32 main(){return o::algorithm<i32>(21);}`),
		parseCompilerFile(t, hir.DefaultPackage, "ops.fy", text),
	}
	r, err := compiler.CheckExplicitFiles(files)
	if err != nil || !r.Verified {
		t.Fatalf("imported grouped operation: %v %+v", err, r.Semantic)
	}
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("imported grouped operation: %+v %v", out, err)
	}
}

// Use actual C seeds/observers for scalars without Fairy source literals. All
// thirteen traverse both grouped members and a generic dependent wrapper.
func TestGroupedStaticInterfaceAllScalarTransport(t *testing.T) {
	var src, extra, body strings.Builder
	src.WriteString(`module main;
 @interface(Transport) pub fn T first<T>(T value){}
 @interface(Transport) pub fn U second<U>(U value){}
 fn T relay<T>(T value){return second<T>(first<T>(value));}`)
	for _, v := range []struct{ fy, c, value string }{
		{"bool", "bool", "true"}, {"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"i32", "int32_t", "INT32_MIN"}, {"i64", "int64_t", "INT64_MIN"}, {"isize", "intptr_t", "INT64_C(-2147483649)"},
		{"u8", "uint8_t", "UINT8_MAX"}, {"u16", "uint16_t", "UINT16_MAX"}, {"u32", "uint32_t", "UINT32_MAX"}, {"u64", "uint64_t", "UINT64_MAX"}, {"usize", "uintptr_t", "UINT64_MAX"}, {"f32", "float", "1.25f"}, {"f64", "double", "-3.5"},
	} {
		fmt.Fprintf(&src, "@implements(first,%[1]s) fn %[1]s first_%[1]s(%[1]s value){return value;}@implements(second,%[1]s) fn %[1]s second_%[1]s(%[1]s value){return value;}", v.fy)
		if v.fy == "bool" {
			// Fairy bool is intentionally not a repr(C) field. Its existing
			// source literal still traverses both providers in every native mode.
			body.WriteString("assert(relay<bool>(true));assert(!relay<bool>(false));")
			continue
		}
		fmt.Fprintf(&src, "@repr(C) struct P_%[1]s{%[1]s value;}unsafe extern C fn P_%[1]s seed_%[1]s();unsafe extern C fn i32 observe_%[1]s(P_%[1]s value);", v.fy)
		fmt.Fprintf(&body, "let P_%[1]s input_%[1]s=unsafe{seed_%[1]s()};let %[1]s output_%[1]s=relay<%[1]s>(input_%[1]s.value);assert(unsafe{observe_%[1]s(P_%[1]s{.value=output_%[1]s})}==1);", v.fy)
		fmt.Fprintf(&extra, "typedef struct{%s value;} P_%s;P_%s seed_%s(void){P_%s p={%s};return p;}int32_t observe_%s(P_%s p){return p.value==%s;}\n", v.c, v.fy, v.fy, v.fy, v.fy, v.value, v.fy, v.fy, v.value)
	}
	fmt.Fprintf(&src, "fn i32 main(){%s return 42;}", body.String())
	result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", src.String())})
	if err != nil || !result.Verified {
		t.Fatalf("all scalar transport: %v %+v", err, result.Semantic)
	}
	runExpandedLanguageNative(t, result, "main::main", extra.String())
}
