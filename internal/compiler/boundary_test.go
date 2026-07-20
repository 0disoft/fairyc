package compiler_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSourceToMIRFixtureRunnersUseSharedCompilerBoundary(t *testing.T) {
	t.Parallel()
	runners := []string{
		filepath.Join("..", "cbackendfixture", "runner.go"),
		filepath.Join("..", "cbackendnative", "runner.go"),
		filepath.Join("..", "interpreterfixture", "runner.go"),
		filepath.Join("..", "mirfixture", "runner.go"),
	}
	for _, runner := range runners {
		runner := runner
		t.Run(filepath.ToSlash(runner), func(t *testing.T) {
			t.Parallel()
			assertSharedCompilerBoundary(t, runner)
		})
	}
}

func assertSharedCompilerBoundary(t *testing.T, filename string) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	aliases := make(map[string]string)
	for _, spec := range parsed.Imports {
		importPath, unquoteErr := strconv.Unquote(spec.Path.Value)
		if unquoteErr != nil {
			t.Fatalf("decode import %s in %s: %v", spec.Path.Value, filename, unquoteErr)
		}
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." && (importPath == "github.com/0disoft/fairyc/internal/compiler" ||
			importPath == "github.com/0disoft/fairyc/internal/semantic" ||
			importPath == "github.com/0disoft/fairyc/internal/mir") {
			t.Errorf("%s dot-imports compiler pipeline package %s", filename, importPath)
		}
		aliases[name] = importPath
	}
	hasCompilerCheck := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		importPath := aliases[identifier.Name]
		switch importPath {
		case "github.com/0disoft/fairyc/internal/compiler":
			hasCompilerCheck = hasCompilerCheck || selector.Sel.Name == "Check"
		case "github.com/0disoft/fairyc/internal/semantic":
			if selector.Sel.Name == "Analyze" || selector.Sel.Name == "AnalyzePackages" {
				t.Errorf("%s bypasses internal/compiler with semantic.%s", filename, selector.Sel.Name)
			}
		case "github.com/0disoft/fairyc/internal/mir":
			if selector.Sel.Name == "Lower" || selector.Sel.Name == "Verify" {
				t.Errorf("%s bypasses internal/compiler with mir.%s", filename, selector.Sel.Name)
			}
		}
		return true
	})
	if !hasCompilerCheck {
		t.Errorf("%s does not call compiler.Check", filename)
	}
}
