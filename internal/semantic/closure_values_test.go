package semantic_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func closureValueAnalysis(t *testing.T, text string) semantic.Result {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("closure source parse: %v %+v diagnostics=%+v", err, parsed, parsed.Tree.Diagnostics)
	}
	before, err := json.Marshal(parsed.AST)
	if err != nil {
		t.Fatal(err)
	}
	result := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}})
	after, err := json.Marshal(parsed.AST)
	if err != nil || string(before) != string(after) {
		t.Fatal("closure semantic preparation mutated original AST JSON", err)
	}
	return result
}

func closureValueAccepted(t *testing.T, text string) semantic.Result {
	t.Helper()
	result := closureValueAnalysis(t, text)
	if result.CompletedThrough != semantic.PhaseFlow || result.FailedPhase != semantic.PhaseNone || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("closure value source rejected: phase=%s diagnostics=%+v unsupported=%+v", result.CompletedThrough, result.Diagnostics, result.Unsupported)
	}
	return result
}

const closureValueFactories = `module main;
struct Box{fn i32(i32) action;}
fn fn i32(i32) make(i32 offset){
 let fn i32(i32) action=fn [i32 offset] i32(i32 x){let value=offset+x;return value;};
 return action;
}
fn fn i32(i32) subtract(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset-x;};return action;}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn Box boxed(i32 offset){let fn i32(i32) action=make(offset);return Box{.action=action};}
fn i32 main(){let first=make(40);let second=subtract(44);let same=first;
 let Box stored=boxed(40);let Box copy=stored;let fn i32(i32) extracted=copy.action;
 return apply(first,2)+apply(second,2)+same(2)+extracted(2);
}`

func TestClosureValueSemanticsFactoriesStructCopiesAndExplicitHIR(t *testing.T) {
	result := closureValueAccepted(t, closureValueFactories)
	functions := map[hir.SymbolID]hir.Function{}
	for _, fn := range result.HIR.Modules[0].Functions {
		functions[fn.Symbol] = fn
	}
	constructors, calls, targets := 0, 0, 0
	for _, function := range functions {
		if function.ClosureTarget != nil {
			targets++
			if function.ClosureTarget.CaptureCount != 1 || function.Public || function.Unsafe || len(function.Effects) != 0 || function.ResultType != hir.TypeI32 || len(function.Parameters) != 2 {
				t.Fatalf("invalid real lifted target: %+v", function)
			}
			for _, parameter := range function.Parameters {
				if parameter.Type != hir.TypeI32 || parameter.Mutable {
					t.Fatal("target capture/value parameters lost immutable i32 order")
				}
			}
		}
		var expression func(*hir.Expression)
		expression = func(value *hir.Expression) {
			if value == nil {
				return
			}
			if value.Kind == hir.ExpressionClosureValue {
				constructors++
				target := functions[value.Symbol]
				if value.Type != hir.TypeClosureI32 || target.ClosureTarget == nil || len(value.Children) != target.ClosureTarget.CaptureCount || len(value.Children) != 1 || value.Children[0].Type != hir.TypeI32 {
					t.Fatal("constructor lost actual target/capture facts")
				}
			}
			if value.Kind == hir.ExpressionClosureCall {
				calls++
				if value.Type != hir.TypeI32 || value.Symbol != hir.NoSymbol || len(value.Children) != 2 || value.Children[0].Kind != hir.ExpressionPath || value.Children[0].Type != hir.TypeClosureI32 || value.Children[1].Type != hir.TypeI32 {
					t.Fatal("value invoke lost ordered callable/argument operands")
				}
			}
			for i := range value.Children {
				expression(&value.Children[i])
			}
		}
		for _, statement := range function.Body.Statements {
			expression(statement.Expression)
		}
		expression(function.Body.Tail)
	}
	if constructors != 2 || targets != 2 || calls != 3 {
		t.Fatalf("explicit HIR facts: constructors=%d targets=%d invokes=%d", constructors, targets, calls)
	}
}

func TestClosureValueSemanticsCaptureArityOrderAndBodyBounds(t *testing.T) {
	for count := 0; count <= 4; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var declarations, captures strings.Builder
			for i := 0; i < count; i++ {
				fmt.Fprintf(&declarations, "let i32 c%d=%d;", i, i+1)
				if i != 0 {
					captures.WriteByte(',')
				}
				fmt.Fprintf(&captures, "i32 c%d", i)
			}
			text := "module main;fn i32 main(){" + declarations.String() + "let fn i32(i32) action=fn [" + captures.String() + "] i32(i32 x){return x;};return action(42);}"
			result := closureValueAccepted(t, text)
			helper := result.HIR.Modules[0].Functions[1]
			if helper.ClosureTarget == nil || helper.ClosureTarget.CaptureCount != count || len(helper.Parameters) != count+1 {
				t.Fatal("zero-to-four capture target fact lost")
			}
			for i := 0; i < count; i++ {
				if helper.Parameters[i].Name != fmt.Sprintf("c%d", i) {
					t.Fatal("capture parameter written order lost")
				}
			}
			if helper.Parameters[count].Name != "x" {
				t.Fatal("call argument must follow capture parameters")
			}
		})
	}
	var sixteen strings.Builder
	sixteen.WriteString("let v0=x;")
	for i := 1; i < 16; i++ {
		fmt.Fprintf(&sixteen, "let v%d=v%d+1;", i, i-1)
	}
	for _, body := range []string{sixteen.String() + "return v15;", "let unused=1;return " + strings.Repeat("x+", 31) + "x;", "let unused=1/0;let second=2147483647+x;return x;"} {
		closureValueAccepted(t, "module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){"+body+"};return action(1);}")
	}
}

func TestClosureValueSemanticsKeepsForbiddenFamiliesClosed(t *testing.T) {
	const literal = "fn [] i32(i32 x){return x;}"
	for name, text := range map[string]string{
		"mutable":               "fn i32 main(){var fn i32(i32) action=" + literal + ";return action(1);}",
		"function-path":         "fn fn i32(i32) make(){let fn i32(i32) action=" + literal + ";return action;}fn i32 main(){let copy=make;return copy(1);}",
		"capture-function":      "fn i32 named(){return 1;}fn i32 main(){let fn i32(i32) action=fn [i32 named] i32(i32 x){return named+x;};return action(1);}",
		"assign":                "fn i32 main(){let fn i32(i32) action=" + literal + ";action=action;return 0;}",
		"mutable-copy":          "fn i32 main(){let fn i32(i32) action=" + literal + ";var copy=action;return copy(1);}",
		"nested":                "fn i32 main(){if(true){let fn i32(i32) action=" + literal + ";discard action;}return 0;}",
		"direct-return":         "fn fn i32(i32) make(){return " + literal + ";}",
		"untyped-value":         "fn fn i32(i32) make(){let action=" + literal + ";return action;}",
		"signature":             "fn void take(fn bool(i32) action){discard action;}",
		"arity-zero":            "fn i32 main(){let fn i32(i32) action=fn [] i32(){return 1;};return 0;}",
		"arity-two":             "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x,i32 y){return x+y;};return 0;}",
		"call-arity":            "fn i32 take(fn i32(i32) action){return action();}",
		"call-type":             "fn i32 take(fn i32(i32) action){return action(true);}",
		"labels":                "fn i32 take(fn i32(i32) action){return action(.x=1);}",
		"field-call":            "struct Box{fn i32(i32) action;}fn i32 take(Box box){return box.action(1);}",
		"option":                "fn void take(Option<fn i32(i32)> action){discard action;}",
		"result":                "fn void take(Result<fn i32(i32),u32> action){discard action;}",
		"borrow":                "fn void take(ref fn i32(i32) action){discard action;}",
		"pointer":               "fn void take(ptr fn i32(i32) action){discard action;}",
		"array":                 "fn void take((fn i32(i32))[2] action){discard action;}",
		"inferred-array":        "fn i32 main(){let fn i32(i32) action=" + literal + ";let array=[action,action];discard array;return 0;}",
		"reprC":                 "@repr(C) struct Box{fn i32(i32) action;}fn void take(Box box){discard box;}",
		"nominal-wrap":          "struct Box{fn i32(i32) action;}fn void take(Option<Box> box){discard box;}",
		"forward-field-wrapper": "struct Outer{Option<Inner> box;}struct Inner{fn i32(i32) action;}",
		"nested-reprC":          "@repr(C) struct Outer{Inner box;}struct Inner{fn i32(i32) action;}",
		"five-captures":         "fn i32 main(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let i32 e=5;let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d,i32 e] i32(i32 x){return x;};return 0;}",
		"value-equality":        "fn i32 main(){let fn i32(i32) action=" + literal + ";assert(action==action);return 0;}",
		"value-borrow":          "fn i32 main(){let fn i32(i32) action=" + literal + ";let pointer=ref action;discard pointer;return 0;}",
		"nominal-ptr":           "struct Box{fn i32(i32) action;}fn void take(ptr Box box){discard box;}",
		"foreign":               "unsafe extern C fn void take(fn i32(i32) action);",
		"foreign-box":           "struct Box{fn i32(i32) action;}unsafe extern C fn Box take(Box box);",
		"resource":              "resource Box{fn i32(i32) action;cleanup{}}fn void take(Box box){discard box;}",
		"capture-value":         "fn i32 main(){let fn i32(i32) action=" + literal + ";let fn i32(i32) other=fn [i32 action] i32(i32 x){return action+x;};return 0;}",
		"capture-bool":          "fn i32 main(){let bool flag=true;let fn i32(i32) action=fn [i32 flag] i32(i32 x){return flag+x;};return 0;}",
		"capture-ref":           "fn i32 take(ref i32 refvalue){let fn i32(i32) action=fn [ref i32 refvalue] i32(i32 x){return x;};return 0;}",
		"mutable-body":          "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){var i32 a=x;return a;};return 0;}",
		"effect-body":           "fn i32 helper(){return 1;}fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return helper();};return 0;}",
		"seventeen":             "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){" + "let a0=x;let a1=x;let a2=x;let a3=x;let a4=x;let a5=x;let a6=x;let a7=x;let a8=x;let a9=x;let a10=x;let a11=x;let a12=x;let a13=x;let a14=x;let a15=x;let a16=x;return x;};return 0;}",
		"sixty-five":            "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let a=1;let b=2;return " + strings.Repeat("x+", 31) + "x;};return 0;}",
	} {
		t.Run(name, func(t *testing.T) {
			result := closureValueAnalysis(t, "module main;"+text)
			if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("forbidden closure value source accepted")
			}
		})
	}
}

// Private lifting metadata cannot turn an ordinary function into a safe target.
func TestClosureValueSemanticsRejectForgedTargetFacts(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		module := &ast.Module{Path: "main", Functions: []ast.Function{{Name: "_fy_closure_forged", Unsafe: unsafe,
			LiftedClosure: &ast.LiftedClosure{CaptureCount: 5}, ResultType: ast.NamedTypeReference("i32", diagnostic.Span{}),
		}}}
		result := semantic.Analyze([]semantic.File{{Path: "forged.fy", Module: module}})
		if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
			t.Fatal("forged target capture count admitted")
		}
	}
}

func TestClosureValueSemanticsGenericForwardingChecksWrittenOrigin(t *testing.T) {
	for name, text := range map[string]string{
		"parameter-result": `module main;fn fn i32(i32) forward<T>(fn i32(i32) action,T marker){discard marker;return action;}fn i32 main(fn i32(i32) action){discard forward<i32>(action,1);return 0;}`,
		"parameter-invoke": `module main;fn i32 invoke<T>(fn i32(i32) action,T marker){discard marker;return action(2);}fn i32 main(fn i32(i32) action){return invoke<i32>(action,1);}`,
		"nominal-forward":  `module main;struct Box{fn i32(i32) action;}fn Box forward<T>(Box box,T marker){discard marker;return box;}fn i32 main(Box box){discard forward<i32>(box,1);return 0;}`,
	} {
		t.Run(name, func(t *testing.T) {
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				t.Fatalf("generic source parse: %v %+v", err, parsed.Tree.Diagnostics)
			}
			before, _ := json.Marshal(parsed.AST)
			written := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
			if len(written.Diagnostics) == 0 && len(written.Unsupported) == 0 {
				t.Fatal("unexpanded generic closure forwarding accepted")
			}
			expansion, err := genericsource.PrepareModule(parsed.AST, "owner", genericplan.DefaultPlanLimits())
			if err != nil {
				t.Fatal("test must reach ordinary semantic checking after real generic preparation:", err)
			}
			prepared := *expansion.Modules[0]
			// Keep only the real specialized declaration and rename it. This
			// verifies admission uses its source fact, not a generated-name prefix.
			prepared.Functions = nil
			for _, function := range expansion.Modules[0].Functions {
				if function.SourceGeneric {
					if len(function.TypeParameters) != 0 {
						t.Fatal("generic preparation did not erase template parameters")
					}
					function.Name = "ordinary_name"
					prepared.Functions = append(prepared.Functions, function)
				}
			}
			if len(prepared.Functions) != 1 {
				t.Fatal("written generic origin was not retained by preparation")
			}
			checked := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: &prepared}})
			found := false
			for _, unsupported := range checked.Unsupported {
				found = found || strings.Contains(unsupported.Feature, "monomorphic source function")
			}
			if !found {
				t.Fatalf("specialized closure signature bypassed monomorphic gate: %+v %+v", checked.Diagnostics, checked.Unsupported)
			}
			after, _ := json.Marshal(parsed.AST)
			if string(before) != string(after) || strings.Contains(string(after), "SourceGeneric") {
				t.Fatal("original source JSON changed or private generic origin leaked")
			}
		})
	}
}

func TestClosureValueSemanticsSnapshotAndCallOrderingEdges(t *testing.T) {
	result := closureValueAccepted(t, `module main;
fn fn i32(i32) make(i32 seed){var i32 a=seed;let i32 b=3;
 let fn i32(i32) action=fn [i32 a,i32 b] i32(i32 x){let unused=1/0;return a-b*x;};
 a=99;return action;}
fn i32 argument(){return 2;}
fn i32 main(){let original=make(20);let copied=original;return copied(argument());}`)
	functions := map[string]hir.Function{}
	targets := map[hir.SymbolID]hir.Function{}
	for _, function := range result.HIR.Modules[0].Functions {
		functions[function.Name] = function
		targets[function.Symbol] = function
	}
	factory := functions["make"]
	creation := factory.Body.Statements[2]
	if creation.Expression.Kind != hir.ExpressionClosureValue || len(creation.Expression.Children) != 2 {
		t.Fatal("snapshot construction lost written capture operands")
	}
	for i, child := range creation.Expression.Children {
		if child.Kind != hir.ExpressionPath || child.Symbol != factory.Body.Statements[i].Binding || child.Type != hir.TypeI32 {
			t.Fatal("capture must be one copied scalar read in declaration order")
		}
	}
	if factory.Body.Statements[3].Kind != hir.StatementAssignment || factory.Body.Statements[3].Target.Symbol != creation.Expression.Children[0].Symbol ||
		factory.Body.Statements[4].Expression.Symbol != creation.Binding {
		t.Fatal("snapshot creation must precede mutation and return the constructed value")
	}
	helper := targets[creation.Expression.Symbol]
	if helper.ClosureTarget == nil || helper.ClosureTarget.CaptureCount != 2 || len(helper.Parameters) != 3 || len(helper.Body.Statements) != 2 {
		t.Fatal("factory must target a separate capture/value parameter frame retaining the unused trap")
	}
	for i, parameter := range helper.Parameters[:2] {
		if parameter.Symbol == creation.Expression.Children[i].Symbol {
			t.Fatal("helper must read its own copied capture parameters, not its creator frame")
		}
	}
	main := functions["main"]
	call := main.Body.Statements[2].Expression
	if call.Kind != hir.ExpressionClosureCall || len(call.Children) != 2 ||
		call.Children[0].Kind != hir.ExpressionPath || call.Children[0].Symbol != main.Body.Statements[1].Binding ||
		call.Children[1].Kind != hir.ExpressionCall || call.Children[1].Symbol != functions["argument"].Symbol {
		t.Fatal("call must retain callable first and once-only argument evaluation second")
	}
}

func TestClosureValueSemanticsLegacyDirectClosuresRemainDistinct(t *testing.T) {
	result := closureValueAccepted(t, `module main;fn i32 main(){let i32 base=20;let action=fn [i32 base] i32(i32 x,i32 y){let a=base-x;return a*y;};return action(2,3);}`)
	for _, function := range result.HIR.Modules[0].Functions {
		if function.ClosureTarget != nil {
			t.Fatal("legacy untyped direct closure unexpectedly became a closure value")
		}
	}
	call := result.HIR.Modules[0].Functions[0].Body.Statements[2].Expression
	if call.Kind != hir.ExpressionCall || len(call.Children) != 4 {
		t.Fatal("legacy helper call lost scalar snapshot and ordinary arguments")
	}
}
