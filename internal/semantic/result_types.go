package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

type resultTypeText struct {
	success      string
	failure      string
	failureStart int
	failureEnd   int
}

func parseResultType(text string) (resultTypeText, bool) {
	trimmed := strings.TrimSpace(text)
	leading := len(text) - len(strings.TrimLeft(text, " \t\r\n"))
	const prefix = "Result<"
	if len(trimmed) <= len(prefix)+1 || !strings.HasPrefix(trimmed, prefix) || trimmed[len(trimmed)-1] != '>' {
		return resultTypeText{}, false
	}
	insideStart := len(prefix)
	inside := trimmed[insideStart : len(trimmed)-1]
	depth := 0
	comma := -1
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return resultTypeText{}, false
			}
		case ',':
			if depth == 0 {
				if comma >= 0 {
					return resultTypeText{}, false
				}
				comma = index
			}
		}
	}
	if depth != 0 || comma < 0 {
		return resultTypeText{}, false
	}
	leftStart, leftEnd := trimRange(inside, 0, comma)
	rightStart, rightEnd := trimRange(inside, comma+1, len(inside))
	if leftStart == leftEnd || rightStart == rightEnd {
		return resultTypeText{}, false
	}
	return resultTypeText{
		success:      inside[leftStart:leftEnd],
		failure:      inside[rightStart:rightEnd],
		failureStart: leading + insideStart + rightStart,
		failureEnd:   leading + insideStart + rightEnd,
	}, true
}

func trimRange(text string, start, end int) (int, int) {
	for start < end && strings.ContainsRune(" \t\r\n", rune(text[start])) {
		start++
	}
	for end > start && strings.ContainsRune(" \t\r\n", rune(text[end-1])) {
		end--
	}
	return start, end
}

func resultFailureSpan(reference ast.TypeReference) (diagnostic.Span, bool) {
	parsed, ok := parseResultType(reference.Text)
	if !ok {
		return diagnostic.Span{}, false
	}
	return diagnostic.Span{
		StartByte: reference.Span.StartByte + parsed.failureStart,
		EndByte:   reference.Span.StartByte + parsed.failureEnd,
	}, true
}

func displayResultType(module string, value hir.Type) string {
	success, failure, ok := hir.ResultElements(value)
	if ok {
		return "Result<" + displayResultType(module, success) + ", " + displayResultType(module, failure) + ">"
	}
	return strings.TrimPrefix(string(value), module+"::")
}

func (c *typeChecker) checkPropagate(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = "?"
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	operandExpression := expression.Children[0]
	operand, operandType := c.checkExpression(operandExpression, hir.TypeInvalid)
	result.Children = appendExpression(result.Children, operand)
	success, actualError, ok := hir.ResultElements(operandType)
	question := questionSpan(c.module.file.Path, expression)
	if !ok {
		c.analyzer.addDiagnostic("E4102", question, RelatedSpan{
			Role: "result-expression",
			Span: span(c.module.file.Path, operandExpression.Span),
		})
		return hir.TypeInvalid
	}
	_, expectedError, enclosingResult := hir.ResultElements(c.analyzer.functions[c.function].typ)
	if !enclosingResult || actualError != expectedError {
		related := make([]RelatedSpan, 0, 2)
		if actualSpan, found := c.resultErrorSpan(operandExpression); found {
			related = append(related, RelatedSpan{Role: "actual-error", Span: actualSpan})
		}
		if expectedSpan := c.analyzer.functions[c.function].resultError; expectedSpan.EndByte > expectedSpan.StartByte {
			related = append(related, RelatedSpan{Role: "expected-error", Span: span(c.module.file.Path, expectedSpan)})
		} else if c.function.ResultType.Span.EndByte > c.function.ResultType.Span.StartByte {
			related = append(related, RelatedSpan{Role: "enclosing-return", Span: span(c.module.file.Path, c.function.ResultType.Span)})
		}
		c.analyzer.addDiagnostic("E4103", question, related...)
	}
	return success
}

func (c *typeChecker) resultErrorSpan(expression *ast.Expression) (hir.Span, bool) {
	if expression == nil {
		return hir.Span{}, false
	}
	if value := c.analyzer.resolved[expression]; value != nil && value.resultError.EndByte > value.resultError.StartByte {
		return span(value.source, value.resultError), true
	}
	if expression.Kind == "call" && len(expression.Children) > 0 {
		if value := c.analyzer.resolved[expression.Children[0]]; value != nil && value.resultError.EndByte > value.resultError.StartByte {
			return span(value.source, value.resultError), true
		}
	}
	if expression.Kind == "group" && len(expression.Children) == 1 {
		return c.resultErrorSpan(expression.Children[0])
	}
	return hir.Span{}, false
}

func questionSpan(source string, expression *ast.Expression) hir.Span {
	end := expression.Span.EndByte
	start := end
	if end > expression.Span.StartByte {
		start = end - 1
	}
	return sourceSpan(source, start, end)
}
