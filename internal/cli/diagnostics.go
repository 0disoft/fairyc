package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/semantic"
)

func writeDiagnostics(writer io.Writer, relativePath string, sourceBytes []byte, values []diagnostic.Diagnostic) error {
	for _, value := range values {
		line, column := bytePosition(sourceBytes, value.Span.StartByte)
		if _, err := fmt.Fprintf(
			writer,
			"%s:%d:%d: %s: %s [bytes %d..%d, repair=%s]\n",
			safeDisplayPath(relativePath),
			line,
			column,
			value.Code,
			diagnosticSummary(value),
			value.Span.StartByte,
			value.Span.EndByte,
			value.Repair,
		); err != nil {
			return err
		}
	}
	return nil
}

func writeSemanticDiagnostics(writer io.Writer, sources map[string][]byte, values []semantic.Diagnostic) error {
	for _, value := range values {
		contents := sources[value.Primary.Source]
		line, column := bytePosition(contents, value.Primary.StartByte)
		if _, err := fmt.Fprintf(
			writer,
			"%s:%d:%d: %s: %s [bytes %d..%d]\n",
			safeDisplayPath(value.Primary.Source),
			line,
			column,
			value.Code,
			semanticDiagnosticSummary(value.Code),
			value.Primary.StartByte,
			value.Primary.EndByte,
		); err != nil {
			return err
		}
		for _, related := range value.Related {
			relatedContents := sources[related.Span.Source]
			relatedLine, relatedColumn := bytePosition(relatedContents, related.Span.StartByte)
			if _, err := fmt.Fprintf(
				writer,
				"  %s: %s:%d:%d [bytes %d..%d]\n",
				related.Role,
				safeDisplayPath(related.Span.Source),
				relatedLine,
				relatedColumn,
				related.Span.StartByte,
				related.Span.EndByte,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeUnsupportedSemantics(writer io.Writer, values []semantic.Unsupported) error {
	for _, value := range values {
		if _, err := fmt.Fprintf(
			writer,
			"fairyc: experimental semantic subset does not support %s at %s [bytes %d..%d]\n",
			value.Feature,
			safeDisplayPath(value.Span.Source),
			value.Span.StartByte,
			value.Span.EndByte,
		); err != nil {
			return err
		}
	}
	return nil
}

func safeDisplayPath(relativePath string) string {
	if strings.ContainsAny(relativePath, "\r\n\t") {
		return strconv.Quote(relativePath)
	}
	return relativePath
}

func diagnosticSummary(value diagnostic.Diagnostic) string {
	switch value.Code {
	case "E0001":
		return "invalid UTF-8 or unsupported identifier syntax"
	case "E0002":
		return "module declaration does not match the source-root-relative file path"
	case "E0003":
		return "invalid Stage 0 import"
	case "E0004":
		return "invalid numeric literal"
	case "E0005":
		return "invalid string escape or unterminated string"
	case "E0006":
		return "C-string contains an interior NUL"
	case "E0007":
		if value.Token != "" {
			return fmt.Sprintf("expected %q", value.Token)
		}
		return "required token is missing"
	case "E0008":
		return "unexpected token"
	case "E0009":
		return "comparison operators require explicit grouping"
	case "E0010":
		return "frontend resource budget exhausted"
	default:
		if entry, found := diagnostic.Lookup(value.Code); found {
			return entry.Meaning
		}
		return "frontend diagnostic"
	}
}

func semanticDiagnosticSummary(code string) string {
	switch code {
	case "E0101":
		return "unresolved name or path"
	case "E0102":
		return "duplicate declaration in one scope"
	case "E0103":
		return "item is not visible from this module"
	case "E1006":
		return "incompatible expected and actual types"
	case "E1007":
		return "condition or logical operand must be bool"
	case "E1008":
		return "assignment target is not mutable"
	case "E1009":
		return "return value does not match the declared result type"
	case "E1010":
		return "call argument count does not match the declaration"
	case "E1011":
		return "signed left shift is not supported in Stage 0"
	case "E1012":
		return "break and continue require an enclosing while loop"
	case "E1101":
		return "resource is used after ownership was moved"
	case "E1103":
		return "moving a field out of a resource is not supported"
	case "E1104":
		return "branch join leaves resource ownership path-dependent"
	case "E1105":
		return "loop body changes the resource state required by the next iteration"
	case "E3101":
		return "Region-derived value escapes its lexical region"
	case "E3102":
		return "plain Region placement cannot contain a resource that needs drop"
	case "E3103":
		return "compiler-owned Region capability cannot be stored"
	case "E3104":
		return "compiler-owned Region capability cannot be constructed directly"
	case "E4101":
		return "Result value must be returned, propagated, matched, or explicitly discarded"
	case "E4102":
		return "postfix ? requires a Result value"
	case "E4103":
		return "propagated Result error type does not match the enclosing return type"
	case "E5106":
		return "@link_name requires exactly one nonempty string and may appear only once"
	default:
		if entry, found := diagnostic.Lookup(code); found {
			return entry.Meaning
		}
		return "semantic diagnostic"
	}
}
