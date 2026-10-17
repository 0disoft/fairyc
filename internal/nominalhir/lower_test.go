package nominalhir

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

func literal(t hir.Type, value string) hir.Expression {
	return hir.Expression{Kind: hir.ExpressionLiteral, Type: t, Value: value}
}
func path(t hir.Type, id hir.SymbolID) hir.Expression {
	return hir.Expression{Kind: hir.ExpressionPath, Type: t, Symbol: id}
}
func pointer(e hir.Expression) *hir.Expression { return &e }
func call(t hir.Type, id hir.SymbolID, args ...hir.Expression) hir.Expression {
	return hir.Expression{Kind: hir.ExpressionCall, Type: t, Symbol: id, Children: append([]hir.Expression{{Kind: hir.ExpressionPath, Symbol: id}}, args...)}
}

func fixture(nominal bool) Program {
	index := Declaration{Module: "scanner", Name: "TokenIndex", Underlying: hir.TypeUSize}
	offset := Declaration{Module: "scanner", Name: "ByteOffset", Underlying: hir.TypeUSize}
	typ := hir.TypeUSize
	if nominal {
		typ = index.Type()
	}
	wrap := func(e hir.Expression) hir.Expression {
		if nominal {
			return hir.Expression{Kind: hir.ExpressionNominalWrap, Type: typ, Children: []hir.Expression{e}}
		}
		return e
	}
	unwrap := func(e hir.Expression) hir.Expression {
		if nominal {
			return hir.Expression{Kind: hir.ExpressionNominalUnwrap, Type: hir.TypeUSize, Children: []hir.Expression{e}}
		}
		return e
	}
	identity := hir.Function{Symbol: 1, Name: "identity", ResultType: typ, Parameters: []hir.Parameter{{Symbol: 10, Name: "index", Type: typ}}, Body: hir.Block{Tail: pointer(path(typ, 10))}}
	next := hir.Function{Symbol: 2, Name: "next", ResultType: hir.TypeUSize, Body: hir.Block{Tail: pointer(literal(hir.TypeUSize, "7"))}}
	main := hir.Function{Symbol: 3, Name: "main", ResultType: hir.TypeI32, Locals: []hir.Local{{Symbol: 20, Name: "index", Type: typ, Mutable: true}}, Body: hir.Block{Statements: []hir.Statement{
		{Kind: hir.StatementBinding, Binding: 20, Expression: pointer(wrap(call(hir.TypeUSize, 2)))},
		{Kind: hir.StatementAssignment, Operator: "=", Target: pointer(path(typ, 20)), Expression: pointer(call(typ, 1, path(typ, 20)))},
		{Kind: hir.StatementDiscard, Expression: pointer(hir.Expression{Kind: hir.ExpressionBinary, Type: hir.TypeBool, Operator: "==", Children: []hir.Expression{path(typ, 20), wrap(literal(hir.TypeUSize, "7"))}})},
		{Kind: hir.StatementDiscard, Expression: pointer(unwrap(path(typ, 20)))},
		{Kind: hir.StatementReturn, Expression: pointer(literal(hir.TypeI32, "0"))},
	}}}
	return Program{Declarations: []Declaration{index, offset}, Modules: []hir.Module{{Path: "scanner", Functions: []hir.Function{identity, next, main}}}}
}

func TestLowerNominalMatchesScalarMIR(t *testing.T) {
	input := fixture(true)
	before, _ := json.Marshal(input)
	got, err := Lower(input)
	if err != nil {
		t.Fatal(err)
	}
	base := fixture(false)
	want, err := mir.Lower(hir.Package{Modules: base.Modules})
	if err != nil {
		t.Fatal(err)
	}
	if failures := mir.Verify(want); len(failures) != 0 {
		t.Fatal(failures)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nominal MIR differs from scalar MIR\ngot=%+v\nwant=%+v", got, want)
	}
	generated, err := cbackend.Emit(got, "scanner::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := cbackend.Emit(want, "scanner::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if generated.Source != baseline.Source {
		t.Fatal("nominal C differs from scalar C")
	}
	outcome, err := interpreter.Execute(got, "scanner::main")
	if err != nil || outcome.Kind != "return" || outcome.Value != 0 {
		t.Fatalf("execute: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("lowering mutated retained HIR")
	}
	calls := 0
	for _, f := range got.Functions {
		for _, b := range f.Blocks {
			if b.Terminator.Kind == mir.TerminatorCall && b.Terminator.Callee == 2 {
				calls++
			}
		}
	}
	if calls != 1 {
		t.Fatalf("wrapped call emitted %d times", calls)
	}
	input.Modules[0].Functions[2].Locals[0].Type = hir.TypeBool
	if !reflect.DeepEqual(got, want) {
		t.Fatal("published MIR aliases input")
	}
}

func TestLowerNominalRejectsBeforeErasure(t *testing.T) {
	cases := map[string]func(*Program){
		"missing declarations":  func(p *Program) { p.Declarations = nil },
		"duplicate declaration": func(p *Program) { p.Declarations = append(p.Declarations, p.Declarations[0]) },
		"wrong base":            func(p *Program) { p.Declarations[0].Underlying = hir.TypeU32 },
		"unknown owner":         func(p *Program) { p.Declarations[0].Module = "absent" },
		"cross package":         func(p *Program) { p.Declarations[0].Package = 1 },
		"unknown type":          func(p *Program) { p.Modules[0].Functions[0].Parameters[0].Type = hir.ParseType("@nominal:missing") },
		"raw argument": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[1].Expression.Children[1] = literal(hir.TypeUSize, "2")
		},
		"wrong argument domain": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[1].Expression.Children[1].Type = p.Declarations[1].Type()
		},
		"forged path type": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[3].Expression.Children[0].Type = p.Declarations[1].Type()
		},
		"wrong local domain": func(p *Program) { p.Modules[0].Functions[2].Locals[0].Type = p.Declarations[1].Type() },
		"wrong assignment": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[1].Expression = pointer(literal(hir.TypeUSize, "2"))
		},
		"immutable assignment": func(p *Program) { p.Modules[0].Functions[2].Locals[0].Mutable = false },
		"wrong return":         func(p *Program) { p.Modules[0].Functions[0].Body.Tail = pointer(literal(hir.TypeUSize, "2")) },
		"wrong call result":    func(p *Program) { p.Modules[0].Functions[2].Body.Statements[1].Expression.Type = hir.TypeUSize },
		"wrong comparison": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[2].Expression.Children[1].Type = p.Declarations[1].Type()
		},
		"nominal arithmetic": func(p *Program) {
			e := p.Modules[0].Functions[2].Body.Statements[2].Expression
			e.Operator = "+"
			e.Type = p.Declarations[0].Type()
		},
		"implicit literal": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[0].Expression = pointer(literal(p.Declarations[0].Type(), "2"))
		},
		"invalid wrap": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[0].Expression.Children[0] = literal(hir.TypeBool, "true")
		},
		"invalid unwrap": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[3].Expression.Children[0] = literal(hir.TypeUSize, "2")
		},
		"duplicate function symbol": func(p *Program) { p.Modules[0].Functions[1].Symbol = 1 },
		"unknown callee":            func(p *Program) { p.Modules[0].Functions[2].Body.Statements[1].Expression.Symbol = 99 },
		"callee mismatch":           func(p *Program) { p.Modules[0].Functions[2].Body.Statements[1].Expression.Children[0].Symbol = 2 },
		"reference": func(p *Program) {
			p.Modules[0].Functions[0].Parameters[0].Type = hir.SharedBorrow(p.Declarations[0].Type())
		},
		"aggregate":           func(p *Program) { p.Modules[0].Structs = []hir.Struct{{Name: "No"}} },
		"foreign":             func(p *Program) { p.Modules[0].ForeignFunctions = []hir.ForeignFunction{{Name: "No"}} },
		"unsupported control": func(p *Program) { p.Modules[0].Functions[2].Body.Statements[0].Kind = hir.StatementWhile },
		"use before initialization": func(p *Program) {
			p.Modules[0].Functions[2].Body.Statements[0].Expression = pointer(path(p.Declarations[0].Type(), 20))
		},
		"nested metadata": func(p *Program) { p.Modules[0].Functions[2].Body.Statements[0].Expression.Block = &hir.Block{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := fixture(true)
			mutate(&p)
			before, _ := json.Marshal(p)
			got, err := Lower(p)
			if err == nil || !reflect.DeepEqual(got, mir.Program{}) {
				t.Fatalf("published invalid MIR: %+v, %v", got, err)
			}
			after, _ := json.Marshal(p)
			if string(before) != string(after) {
				t.Fatal("failed validation mutated input")
			}
		})
	}
}

func TestNominalIdentityAndDiagnostic(t *testing.T) {
	p := fixture(true)
	other := p.Declarations[0]
	other.Module = "other"
	p.Declarations = append(p.Declarations, other)
	p.Modules = append(p.Modules, hir.Module{Path: "other"})
	if _, err := Lower(p); err != nil {
		t.Fatal(err)
	}
	p.Modules[0].Functions[0].Body.Tail = pointer(literal(hir.TypeUSize, "2"))
	_, err := Lower(p)
	if err == nil || !strings.Contains(err.Error(), "TokenIndex") || !strings.Contains(err.Error(), "usize") {
		t.Fatalf("missing type names: %v", err)
	}
	if other.Type() == p.Declarations[0].Type() {
		t.Fatal("same name in different modules aliases")
	}
}

func TestNominalExtractionPreservesCheckedArithmetic(t *testing.T) {
	p := fixture(true)
	value := p.Modules[0].Functions[2].Body.Statements[0].Expression
	value.Children[0] = hir.Expression{Kind: hir.ExpressionBinary, Type: hir.TypeUSize, Operator: "+", Children: []hir.Expression{
		literal(hir.TypeUSize, "18446744073709551615"), literal(hir.TypeUSize, "1"),
	}}
	program, err := Lower(p)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(program, "scanner::main")
	if err != nil || outcome.Kind != "trap" || outcome.TrapCategory != "integer-overflow" {
		t.Fatalf("overflow check lost: %+v %v", outcome, err)
	}
}
