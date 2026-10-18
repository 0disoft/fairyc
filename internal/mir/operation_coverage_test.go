package mir_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
)

// Default rejection is not exhaustive handling: every consumer must explicitly
// implement, ignore, or reject each declared opcode when the vocabulary grows.
func TestOperationDispatchCoverage(t *testing.T) {
	declarations, err := parser.ParseFile(token.NewFileSet(), "operation_kind.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]bool)
	for _, declaration := range declarations.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			for _, name := range spec.(*ast.ValueSpec).Names {
				if strings.HasPrefix(name.Name, "Operation") && name.Name != "OperationInvalid" {
					expected[name.Name] = true
				}
			}
		}
	}
	if len(expected) != len(mir.AllOperationKinds()) {
		t.Fatal("operation inventory and declarations disagree")
	}
	for _, consumer := range []struct{ directory, function string }{
		{".", "verifyFunction"},
		{filepath.Join("..", "interpreter"), "call"},
		{filepath.Join("..", "cbackend"), "inspectFunction"},
		{filepath.Join("..", "cbackend"), "emitOperation"},
	} {
		t.Run(consumer.function, func(t *testing.T) {
			packages, err := parser.ParseDir(token.NewFileSet(), consumer.directory, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			var selected *ast.SwitchStmt
			for _, pkg := range packages {
				for name, file := range pkg.Files {
					if strings.HasSuffix(name, "_test.go") {
						continue
					}
					for _, decl := range file.Decls {
						function, ok := decl.(*ast.FuncDecl)
						if !ok || function.Name.Name != consumer.function {
							continue
						}
						ast.Inspect(function.Body, func(node ast.Node) bool {
							branch, ok := node.(*ast.SwitchStmt)
							if !ok {
								return true
							}
							selector, ok := branch.Tag.(*ast.SelectorExpr)
							if !ok || selector.Sel.Name != "Kind" {
								return true
							}
							owner, ok := selector.X.(*ast.Ident)
							if !ok || owner.Name != "operation" {
								return true
							}
							if selected != nil {
								t.Fatal("ambiguous main operation dispatch")
							}
							selected = branch
							return false
						})
					}
				}
			}
			if selected == nil {
				t.Fatal("main operation dispatch is missing")
			}
			covered := make(map[string]bool)
			for _, item := range selected.Body.List {
				for _, expression := range item.(*ast.CaseClause).List {
					switch value := expression.(type) {
					case *ast.Ident:
						covered[value.Name] = true
					case *ast.SelectorExpr:
						covered[value.Sel.Name] = true
					}
				}
			}
			for name := range expected {
				if !covered[name] {
					t.Errorf("%s requires explicit handling of %s", consumer.function, name)
				}
			}
		})
	}
}
