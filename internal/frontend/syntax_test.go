package frontend

import (
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestAnalyzeSyntaxLowersDiagnosticFreeInput(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { return 0; }\n")
	result, err := AnalyzeSyntax(source.NewFile("main.fy", input), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if result.Tree == nil || !result.Tree.ErrorFree() {
		t.Fatalf("AnalyzeSyntax() tree = %+v, want diagnostic-free tree", result.Tree)
	}
	if result.AST == nil || result.AST.Path != "main" {
		t.Fatalf("AnalyzeSyntax() AST = %+v, want module main", result.AST)
	}
}

func TestAnalyzeSyntaxKeepsDiagnosticTreeWithoutAST(t *testing.T) {
	t.Parallel()
	input := []byte("module main\n")
	result, err := AnalyzeSyntax(source.NewFile("main.fy", input), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if result.Tree == nil || len(result.Tree.Diagnostics) == 0 {
		t.Fatalf("AnalyzeSyntax() tree = %+v, want diagnostics", result.Tree)
	}
	if result.AST != nil {
		t.Fatalf("AnalyzeSyntax() AST = %+v, want nil", result.AST)
	}
}
