package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/hir"
	"regexp"

	"github.com/0disoft/fairyc/internal/semantic"
)

var diagnosticPackageQualifier = regexp.MustCompile(`\$p[0-9]+::`)

func semanticDiagnosticMessage(value semantic.Diagnostic) string {
	summary := semanticDiagnosticSummary(value.Code)
	if value.Code != "E1006" || value.ExpectedType == hir.ParseType("") || value.ActualType == hir.ParseType("") {
		return summary
	}
	expected, actual := value.ExpectedType.String(), value.ActualType.String()
	shortExpected := diagnosticPackageQualifier.ReplaceAllString(expected, "")
	shortActual := diagnosticPackageQualifier.ReplaceAllString(actual, "")
	if shortExpected != shortActual {
		expected, actual = shortExpected, shortActual
	}
	return fmt.Sprintf("%s: expected %q, got %q", summary, expected, actual)
}
