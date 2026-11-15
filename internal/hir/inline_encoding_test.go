package hir

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func inlineEncodingLiteral() *Expression {
	return &Expression{Kind: ExpressionLiteral, Type: TypeI32, Value: "0x2_a"}
}

func inlineEncodingFunction() Function {
	return Function{Symbol: 1, Public: true, ResultType: TypeI32, Body: Block{Tail: inlineEncodingLiteral()}}
}

func TestInlineEncodingCanonicalVersions(t *testing.T) {
	literal := `{"Op":"literal","Value":"42","Children":null}`
	for _, version := range []int{1, 2, 3, 4} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			fn := inlineEncodingFunction()
			root := literal
			callee := inlineEncodingFunction()
			callee.Symbol, callee.Name, callee.InlineBody = 9, "leaf", "admitted"
			if version == 2 || version == 4 {
				fn.Locals = []Local{{Symbol: 2, Type: TypeI32, Mutable: version == 4}}
				fn.Body.Statements = []Statement{{Kind: StatementBinding, Binding: 2, Expression: inlineEncodingLiteral()}}
				fn.Body.Tail = &Expression{Kind: ExpressionPath, Type: TypeI32, Symbol: 2}
				root = `{"Op":"bindings","Value":"","Children":[` + literal + `,{"Op":"local","Value":"0","Children":null}]}`
			}
			if version == 3 {
				fn.Body.Tail = &Expression{Kind: ExpressionCall, Type: TypeI32, Symbol: 9, Children: []Expression{{Kind: ExpressionPath, Symbol: 9}}}
				root = `{"Op":"call","Value":"leaf","Children":null}`
			}
			before, _ := json.Marshal(fn)
			data, callees, err := inlineFunctionEncoding(fn, func(SymbolID) *Function { return &callee })
			want := fmt.Sprintf(`{"Version":"fairyc.inline-function/v%d","Arity":0,"Root":%s}`, version, root)
			if err != nil || string(data) != want {
				t.Fatalf("encoding = %s, %v; want %s", data, err, want)
			}
			if (version == 3 && !reflect.DeepEqual(callees, []*Function{&callee})) || (version != 3 && callees != nil) {
				t.Fatalf("unexpected callees: %+v", callees)
			}
			after, _ := json.Marshal(fn)
			if string(before) != string(after) {
				t.Fatal("encoding mutated caller-owned HIR")
			}
		})
	}
}

func TestInlineEncodingBoundariesAndValidationOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*Function)
		want string
	}{
		{"four-parameters", func(fn *Function) { inlineEncodingParameters(fn, 4) }, ""},
		{"five-parameters", func(fn *Function) { inlineEncodingParameters(fn, 5) }, "@inline requires a public safe i32 function with at most four i32 parameters and no effects or owner and at most 16 i32 locals"},
		{"sixteen-locals", func(fn *Function) { inlineEncodingLocals(fn, 16) }, ""},
		{"seventeen-locals", func(fn *Function) { inlineEncodingLocals(fn, 17) }, "@inline requires a public safe i32 function with at most four i32 parameters and no effects or owner and at most 16 i32 locals"},
		{"64-nodes", func(fn *Function) { inlineEncodingNodes(fn, 64) }, ""},
		{"65-nodes", func(fn *Function) { inlineEncodingNodes(fn, 65) }, "@inline requires at most 64 pure i32 expression nodes"},
		{"63-steps-plus-return", func(fn *Function) { inlineEncodingSteps(fn, 63) }, ""},
		{"64-steps-exhaust-node-budget", func(fn *Function) { inlineEncodingSteps(fn, 64) }, "@inline requires at most 64 pure i32 expression nodes"},
		{"65-steps", func(fn *Function) { inlineEncodingSteps(fn, 65) }, "@inline requires at most 64 straight-line steps"},
		{"parameter-before-body", func(fn *Function) {
			inlineEncodingSteps(fn, 65)
			fn.Parameters = []Parameter{{Type: TypeBool}}
		}, "@inline requires immutable i32 parameters"},
		{"step-budget-before-local-type", func(fn *Function) {
			inlineEncodingSteps(fn, 65)
			fn.Locals[0].Type = TypeBool
		}, "@inline requires at most 64 straight-line steps"},
		{"assignment-shape-before-ownership", func(fn *Function) {
			fn.Body.Statements = []Statement{{Kind: StatementAssignment, Operator: "+="}}
		}, "@inline assignment requires a mutable local path"},
		{"assignment-before-declaration", func(fn *Function) {
			inlineEncodingSteps(fn, 2)
			fn.Body.Statements[0], fn.Body.Statements[1] = fn.Body.Statements[1], fn.Body.Statements[0]
		}, "@inline assignment requires a preceding mutable i32 local"},
		{"self-initializer-is-unbound", func(fn *Function) {
			inlineEncodingLocals(fn, 1)
			fn.Body.Statements[0].Expression = &Expression{Kind: ExpressionPath, Type: TypeI32, Symbol: 10}
		}, "inline path must name a parameter or preceding local"},
		{"local-count-before-return", func(fn *Function) {
			fn.Locals = []Local{{Symbol: 2, Type: TypeI32}}
			fn.Body.Tail = nil
		}, "@inline local declaration count mismatch"},
		{"missing-return", func(fn *Function) { fn.Body.Tail = nil }, "@inline requires at most 64 pure i32 expression nodes"},
		{"invalid-literal", func(fn *Function) { fn.Body.Tail.Value = "0xinvalid" }, "invalid inline i32 literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inlineEncodingFunction()
			tc.edit(&fn)
			before, _ := json.Marshal(fn)
			_, _, err := inlineFunctionEncoding(fn, nil)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("error = %q; want %q", got, tc.want)
			}
			after, _ := json.Marshal(fn)
			if string(before) != string(after) {
				t.Fatal("encoding mutated caller-owned HIR")
			}
		})
	}
}

func inlineEncodingParameters(fn *Function, count int) {
	for i := 0; i < count; i++ {
		fn.Parameters = append(fn.Parameters, Parameter{Symbol: SymbolID(100 + i), Type: TypeI32})
	}
}

func inlineEncodingLocals(fn *Function, count int) {
	for i := 0; i < count; i++ {
		symbol := SymbolID(10 + i)
		fn.Locals = append(fn.Locals, Local{Symbol: symbol, Type: TypeI32})
		fn.Body.Statements = append(fn.Body.Statements, Statement{Kind: StatementBinding, Binding: symbol, Expression: inlineEncodingLiteral()})
	}
}

func inlineEncodingNodes(fn *Function, count int) {
	for i := 1; i < count; i++ {
		fn.Body.Tail = &Expression{Kind: ExpressionPrefix, Type: TypeI32, Operator: "+", Children: []Expression{*fn.Body.Tail}}
	}
}

func inlineEncodingSteps(fn *Function, count int) {
	inlineEncodingLocals(fn, 1)
	fn.Locals[0].Mutable = true
	for i := 1; i < count; i++ {
		fn.Body.Statements = append(fn.Body.Statements, Statement{Kind: StatementAssignment, Operator: "=", Target: &Expression{Kind: ExpressionPath, Type: TypeI32, Symbol: 10}, Expression: inlineEncodingLiteral()})
	}
}

func TestInlineEncodingExplicitReturn(t *testing.T) {
	fn := inlineEncodingFunction()
	want, _, err := inlineFunctionEncoding(fn, nil)
	if err != nil {
		t.Fatal(err)
	}
	fn.Body.Statements = []Statement{{Kind: StatementReturn, Expression: fn.Body.Tail}}
	fn.Body.Tail = nil
	got, _, err := inlineFunctionEncoding(fn, nil)
	if err != nil || string(got) != string(want) {
		t.Fatalf("explicit return = %s, %v; want %s", got, err, want)
	}
}

func TestInlineEncodingAssignmentCallOrder(t *testing.T) {
	fn := inlineEncodingFunction()
	inlineEncodingSteps(&fn, 2)
	fn.Body.Tail = &Expression{Kind: ExpressionPath, Type: TypeI32, Symbol: 10}
	outer, inner := inlineEncodingFunction(), inlineEncodingFunction()
	outer.Symbol, outer.Name, outer.InlineBody = 200, "outer", "admitted"
	inner.Symbol, inner.Name, inner.InlineBody = 201, "inner", "admitted"
	outer.Parameters, inner.Parameters = []Parameter{{Type: TypeI32}}, []Parameter{{Type: TypeI32}}
	call := func(symbol SymbolID, argument Expression) Expression {
		return Expression{Kind: ExpressionCall, Type: TypeI32, Symbol: symbol, Children: []Expression{{Kind: ExpressionPath, Symbol: symbol}, argument}}
	}
	expression := call(outer.Symbol, call(inner.Symbol, *fn.Body.Tail))
	fn.Body.Statements[1].Expression = &expression
	before, _ := json.Marshal([]Function{fn, outer, inner})
	var calls []SymbolID
	data, callees, err := inlineFunctionEncoding(fn, func(symbol SymbolID) *Function {
		calls = append(calls, symbol)
		if symbol == outer.Symbol {
			return &outer
		}
		return &inner
	})
	want := `{"Version":"fairyc.inline-function/v4","Arity":0,"Root":{"Op":"bindings","Value":"","Children":[{"Op":"literal","Value":"42","Children":null},{"Op":"call","Value":"outer","Children":[{"Op":"call","Value":"inner","Children":[{"Op":"local","Value":"0","Children":null}]}]},{"Op":"local","Value":"1","Children":null}]}}`
	if err != nil || string(data) != want {
		t.Fatalf("assignment versions = %s, %v; want %s", data, err, want)
	}
	if !reflect.DeepEqual(calls, []SymbolID{200, 201}) || !reflect.DeepEqual(callees, []*Function{&outer, &inner}) {
		t.Fatalf("call order = %v, callees = %+v", calls, callees)
	}
	after, _ := json.Marshal([]Function{fn, outer, inner})
	if string(before) != string(after) {
		t.Fatal("encoding mutated caller or resolved declarations")
	}
}

func TestInlineEncodingWideMalformedExpression(t *testing.T) {
	for _, invalidFirstChild := range []bool{false, true} {
		fn := inlineEncodingFunction()
		fn.Body.Tail.Children = make([]Expression, 128)
		for i := range fn.Body.Tail.Children {
			fn.Body.Tail.Children[i] = *inlineEncodingLiteral()
		}
		want := "@inline requires at most 64 pure i32 expression nodes"
		if invalidFirstChild {
			fn.Body.Tail.Children[0].Value = "invalid"
			want = "invalid inline i32 literal"
		}
		_, _, err := inlineFunctionEncoding(fn, nil)
		if err == nil || err.Error() != want {
			t.Fatalf("invalidFirstChild=%v: error = %v; want %s", invalidFirstChild, err, want)
		}
	}
}

func BenchmarkInlineEncoding(b *testing.B) {
	var tree func(int) Expression
	tree = func(depth int) Expression {
		if depth == 0 {
			return *inlineEncodingLiteral()
		}
		return Expression{Kind: ExpressionBinary, Type: TypeI32, Operator: "+", Children: []Expression{tree(depth - 1), tree(depth - 1)}}
	}
	for _, depth := range []int{0, 1, 5} {
		b.Run(fmt.Sprintf("nodes-%d", (1<<(depth+1))-1), func(b *testing.B) {
			fn := inlineEncodingFunction()
			root := tree(depth)
			fn.Body.Tail = &root
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				data, _, err := inlineFunctionEncoding(fn, nil)
				if err != nil || len(data) == 0 {
					b.Fatal(err)
				}
			}
		})
	}
}
