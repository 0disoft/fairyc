package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
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
		return "frontend diagnostic"
	}
}
