package compiler_test

import (
	"encoding/json"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

const copiedClosureSource = `module main;fn i32 main(){
var i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};offset=100;
assert(add(1)==41);assert(add(2)==42);
if(true){let i32 add=7;assert(add==7);}
let second=fn [i32 offset] i32(i32 x){offset+x};assert(second(1)==101);
let no_capture=fn [] i32(i32 x){return x*2;};assert(no_capture(21)==42);
for(var i32 i=0;i<2;i+=1){assert(add(2)==42);}
let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;
let full=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 w,i32 x,i32 y,i32 z){return a+b+c+d+w+x+y+z;};assert(full(5,6,7,14)==42);
return add(2);}`

func TestCopiedClosureExecution(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", copiedClosureSource)
	before, _ := json.Marshal(file.Module)
	r, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !r.Verified {
		t.Fatalf("compile: %v diagnostics=%+v unsupported=%+v", err, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("interpret: %+v %v", out, err)
	}
	runExpandedLanguageNative(t, r, "main::main", "")
}
func TestCopiedClosureRejectsEscapesAndUnsupportedCaptures(t *testing.T) {
	for name, text := range map[string]string{
		"return":           strings.Replace(copiedClosureSource, "return add(2);", "return add;", 1),
		"copy":             strings.Replace(copiedClosureSource, "return add(2);", "let other=add;return other(2);", 1),
		"mutation":         strings.Replace(copiedClosureSource, "return offset+x;", "offset=1;return offset+x;", 1),
		"implicit capture": strings.Replace(copiedClosureSource, "fn [i32 offset]", "fn []", 1),
		"resource capture": strings.Replace(copiedClosureSource, "fn [i32 offset]", "fn [ref i32 offset]", 1),
		"duplicate":        strings.Replace(copiedClosureSource, "fn [i32 offset]", "fn [i32 offset,i32 offset]", 1),
		"nested":           strings.Replace(copiedClosureSource, "let add=fn", "if(true){let add=fn", 1) + "}",
		"annotation":       strings.Replace(copiedClosureSource, "let add=fn", "let i32 add=fn", 1),
		"hidden helper":    strings.Replace(copiedClosureSource, "return add(2);", "return _fy_closure_main_1(40,2);", 1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err == nil && r.Verified {
				t.Fatal("closed closure accepted")
			}
		})
	}
}

func TestCopiedClosureInGenericFunction(t *testing.T) {
	text := `module main;fn T keep<T>(T value){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};assert(add(2)==42);return value;}fn i32 main(){return keep<i32>(42);}`
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	r, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !r.Verified {
		t.Fatalf("generic closure: %v %+v", err, r.Semantic)
	}
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("generic closure: %+v %v", out, err)
	}
}

func TestCopiedClosureOrderedArguments(t *testing.T) {
	text := `module main;unsafe extern C fn i32 ordered(i32 index);unsafe extern C fn i32 finished();fn i32 main(){let i32 base=39;let sum=fn [i32 base] i32(i32 a,i32 b){return base+a+b;};let i32 answer=sum(unsafe{ordered(1)},unsafe{ordered(2)});assert(unsafe{finished()}==2);return answer;}`
	r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !r.Verified {
		t.Fatalf("argument closure: %v %+v", err, r.Semantic)
	}
	runExpandedLanguageNative(t, r, "main::main", `static int32_t position=0;int32_t ordered(int32_t index){return ++position==index?index:-1000;}int32_t finished(void){return position;}`)
}
func TestCopiedClosureBudgets(t *testing.T) {
	for _, text := range []string{
		`module main;fn i32 main(){let c=fn [] i32(i32 x){return ` + strings.Repeat("x+", 33) + `x;};return c(1);}`,
		`module main;fn i32 main(){let i32 a=1;let c=fn [i32 a,i32 b,i32 c,i32 d,i32 e] i32(i32 x){return x;};return c(42);}`,
		`module main;fn i32 main(){let c=fn [] i32(i32 a,i32 b,i32 c,i32 d,i32 e){return a;};return c(42,1,2,3,4);}`,
		`module main;fn i32 main(){if(true){let c=fn [] i32(i32 x){return x;};return c(42);}return 0;}`,
		`module main;fn i32 run(i32 c){let c=fn [] i32(i32 x){return x;};return c(42);}fn i32 main(){return run(1);}`,
	} {
		r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err == nil && r.Verified {
			t.Fatal("closure budget/scope escaped")
		}
	}
}

func TestCopiedClosureGenericExpansionDetachesSyntax(t *testing.T) {
	text := `module main;fn T keep<T>(T value){let i32 base=40;let add=fn [i32 base] i32(i32 x){return base+x;};assert(add(2)==42);return value;}fn i32 main(){return keep<i32>(42);}`
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	before, _ := json.Marshal(file.Module)
	expansion, err := genericsource.PrepareModule(file.Module, "closure_owner", genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range expansion.Modules[0].Functions {
		if fn.Body == nil {
			continue
		}
		for _, s := range fn.Body.Statements {
			if s.Binding == nil || s.Binding.Expression == nil || s.Binding.Expression.Closure == nil {
				continue
			}
			c := s.Binding.Expression.Closure
			c.Captures[0].Type.Path = "u32"
			c.Function.Parameters[0].Name = "changed"
			c.Function.Body.Statements[0].Expression.Text = "changed"
			found = true
		}
	}
	after, _ := json.Marshal(file.Module)
	if !found || string(before) != string(after) {
		t.Fatal("generic closure syntax is absent or aliases original input")
	}
}

func TestCopiedClosureShadowedViewIteration(t *testing.T) {
	text := `module main;fn i32 main(){let c=fn [] i32(i32 x){return x;};if(true){let i32 c=7;let ref u8[] view=b"xx"[..];for(let u8 x in view){discard x;}assert(c==7);}return c(42);}`
	result, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !result.Verified {
		t.Fatalf("shadowed closure iteration: %v %+v", err, result.Semantic)
	}
	out, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("shadowed closure iteration: %+v %v", out, err)
	}
}
