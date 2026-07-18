package semantic

import (
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

func TestResultFailureSpanTracksSecondTypeArgument(t *testing.T) {
	t.Parallel()
	reference := ast.TypeReference{
		Text: "Result<i32, SourceError>",
		Span: diagnostic.Span{StartByte: 105, EndByte: 129},
	}
	actual, ok := resultFailureSpan(reference)
	if !ok || actual.StartByte != 117 || actual.EndByte != 128 {
		t.Fatalf("resultFailureSpan() = %+v, %v", actual, ok)
	}
}

func TestParseResultTypeHandlesNestedSuccess(t *testing.T) {
	t.Parallel()
	actual, ok := parseResultType("Result<Result<i32, Inner>, Outer>")
	if !ok || actual.success != "Result<i32, Inner>" || actual.failure != "Outer" {
		t.Fatalf("parseResultType() = %+v, %v", actual, ok)
	}
}
