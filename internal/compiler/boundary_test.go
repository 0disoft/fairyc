package compiler_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSourceToMIRFixtureRunnersUseSharedCompilerBoundary(t *testing.T) {
	t.Parallel()
	runners := []string{
		filepath.Join("..", "cbackendfixture"),
		filepath.Join("..", "cbackendnative"),
		filepath.Join("..", "interpreterfixture"),
		filepath.Join("..", "mirfixture"),
	}
	for _, runner := range runners {
		runner := runner
		t.Run(filepath.ToSlash(runner), func(t *testing.T) {
			t.Parallel()
			files, err := filepath.Glob(filepath.Join(runner, "*.go"))
			if err != nil {
				t.Fatal(err)
			}
			hasCompilerCheck := false
			for _, filename := range files {
				if strings.HasSuffix(filename, "_test.go") {
					continue
				}
				// Inspect every production file, even after finding the shared entry point.
				checked := inspectSharedCompilerBoundary(t, filename)
				hasCompilerCheck = hasCompilerCheck || checked
			}
			if !hasCompilerCheck {
				t.Errorf("%s does not call compiler.Check", runner)
			}
		})
	}
}

func inspectSharedCompilerBoundary(t *testing.T, filename string) bool {
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
	return hasCompilerCheck
}
