package ast

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestVectorForPreparationUsesActualSDKImportAlias(t *testing.T) {
	for _, alias := range []string{"v", "items"} {
		t.Run(alias, func(t *testing.T) {
			input := "module main; import std::vec as " + alias + "; fn void show(ref " + alias +
				"::Vec<i32> values) { for (let ref i32 item in values) { assert(*item == 7); } }"
			tree := syntax.Parse(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("parser diagnostics = %+v", tree.Diagnostics)
			}
			module, err := Lower(tree)
			if err != nil || len(module.Functions) != 1 || len(module.Functions[0].Body.Statements) != 1 {
				t.Fatalf("lowered module = %+v, %v", module, err)
			}
			statement := module.Functions[0].Body.Statements[0]
			if statement.ByteFor == nil || statement.ByteFor.View == nil || statement.ByteFor.View.Text != "values" ||
				statement.ByteFor.Type.Kind != TypeKindReference || len(statement.ByteFor.Body.Statements) != 1 {
				t.Fatalf("source loop was not retained: %+v", statement.ByteFor)
			}
			if statement.Expression != nil {
				t.Fatalf("candidate preparation changed executable statement: %+v", statement.Expression)
			}
			prepared := statement.ByteFor.PreparedVector
			if prepared == nil || prepared.Block == nil || len(prepared.Block.Statements) != 3 {
				t.Fatalf("prepared loop = %+v", prepared)
			}
			owner := prepared.Block.Statements[0].Binding
			bound := prepared.Block.Statements[1].Binding
			classic := prepared.Block.Statements[2].For
			if owner == nil || owner.Name != vectorForOwnerName || owner.Expression != statement.ByteFor.View ||
				owner.Type == nil || owner.Type.Element == nil || owner.Type.Element.Path != alias+"::Vec" ||
				bound == nil || bound.Name != vectorForBoundName || classic == nil || classic.Body == nil ||
				classic.Initializer == nil || classic.Initializer.Binding.Name != vectorForIndexName ||
				classic.Post == nil || classic.Post.Operator != "+=" {
				t.Fatalf("prepared owner/bound/loop = %+v / %+v / %+v", owner, bound, classic)
			}
			checkVectorCall(t, bound.Expression, alias+"::len", 1)
			if len(classic.Body.Statements) != 2 || classic.Body.Statements[0].Binding == nil ||
				classic.Body.Statements[0].Binding.Name != "item" {
				t.Fatalf("prepared body = %+v", classic.Body)
			}
			checkVectorCall(t, classic.Body.Statements[0].Binding.Expression, alias+"::at", 2)
		})
	}
}

func TestVectorForPreparationRetainsExplicitBorrowAndRejectsOtherImports(t *testing.T) {
	for _, sample := range []struct {
		name, importPath string
		prepared         bool
	}{
		{name: "SDK module", importPath: "std::vec", prepared: true},
		{name: "unrelated module", importPath: "other::vec"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			input := "module main; import " + sample.importPath + " as v; fn void show(v::Vec<i32> values) { " +
				"if (true) { for (let ref i32 item in ref values) {} } }"
			tree := syntax.Parse(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("parser diagnostics = %+v", tree.Diagnostics)
			}
			module, err := Lower(tree)
			if err != nil || len(module.Functions) != 1 {
				t.Fatalf("lowered module = %+v, %v", module, err)
			}
			outer := module.Functions[0].Body.Statements[0].Expression
			if outer == nil || len(outer.Children) != 2 || outer.Children[1].Block == nil ||
				len(outer.Children[1].Block.Statements) != 1 {
				t.Fatalf("outer if = %+v", outer)
			}
			loop := outer.Children[1].Block.Statements[0]
			if loop.ByteFor == nil || loop.ByteFor.View == nil || loop.ByteFor.View.Kind != syntax.KindPrefix ||
				!strings.Contains(input[loop.ByteFor.View.Span.StartByte:loop.ByteFor.View.Span.EndByte], "ref values") {
				t.Fatalf("source borrow = %+v", loop.ByteFor)
			}
			if loop.Expression != nil || (loop.ByteFor.PreparedVector != nil) != sample.prepared {
				t.Fatalf("unexpected prepared loop for %s: %+v / %+v", sample.importPath, loop.Expression, loop.ByteFor.PreparedVector)
			}
		})
	}
}

func TestVectorForPreparationAcceptsBorrowedFieldPlace(t *testing.T) {
	input := []byte("module main; import std::vec as v; fn void show(ref Holder stream) { " +
		"for (let ref i32 item in ref stream.cooked) {} }")
	tree := syntax.Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("parser diagnostics = %+v", tree.Diagnostics)
	}
	module, err := Lower(tree)
	if err != nil || len(module.Functions) != 1 {
		t.Fatalf("lowered module = %+v, %v", module, err)
	}
	loop := module.Functions[0].Body.Statements[0].ByteFor
	if loop == nil || loop.PreparedVector == nil || loop.PreparedVector.Block == nil ||
		len(loop.PreparedVector.Block.Statements) != 3 ||
		loop.PreparedVector.Block.Statements[0].Binding.Expression != loop.View {
		t.Fatalf("borrowed field was not retained exactly once: %+v", loop)
	}
}

func checkVectorCall(t *testing.T, expression *Expression, want string, arguments int) {
	t.Helper()
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) != arguments+1 {
		t.Fatalf("call = %+v", expression)
	}
	callee := expression.Children[0]
	if callee == nil || callee.Kind != syntax.KindSpecialize || len(callee.Children) != 1 ||
		callee.Children[0].Text != want || len(callee.TypeArguments) != 1 || callee.TypeArguments[0].Path != "i32" {
		t.Fatalf("specialized call = %+v", callee)
	}
}
