package semantic

import (
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

func TestResultFailureSpanTracksSecondTypeArgument(t *testing.T) {
	t.Parallel()
	reference := ast.NamedTypeReference("Result", diagnostic.Span{StartByte: 105, EndByte: 129})
	reference.Arguments = []ast.TypeReference{
		ast.NamedTypeReference("i32", diagnostic.Span{StartByte: 112, EndByte: 115}),
		ast.NamedTypeReference("SourceError", diagnostic.Span{StartByte: 117, EndByte: 128}),
	}
	actual, ok := resultFailureSpan(reference)
	if !ok || actual.StartByte != 117 || actual.EndByte != 128 {
		t.Fatalf("resultFailureSpan() = %+v, %v", actual, ok)
	}
}

func TestResultFailureSpanRejectsNonResultType(t *testing.T) {
	t.Parallel()
	actual, ok := resultFailureSpan(ast.NamedTypeReference("Option", diagnostic.Span{StartByte: 10, EndByte: 21}))
	if ok || actual != (diagnostic.Span{}) {
		t.Fatalf("resultFailureSpan() = %+v, %v", actual, ok)
	}
}
