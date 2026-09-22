package cli

import (
	"fmt"
	"regexp"

	"github.com/0disoft/fairyc/internal/semantic"
)

var diagnosticPackageQualifier = regexp.MustCompile(`\$p[0-9]+::`)

func semanticDiagnosticMessage(value semantic.Diagnostic) string {
	summary := semanticDiagnosticSummary(value.Code)
	if value.Code != "E1006" || value.ExpectedType == "" || value.ActualType == "" {
		return summary
	}
	expected, actual := string(value.ExpectedType), string(value.ActualType)
	shortExpected := diagnosticPackageQualifier.ReplaceAllString(expected, "")
	shortActual := diagnosticPackageQualifier.ReplaceAllString(actual, "")
	if shortExpected != shortActual {
		expected, actual = shortExpected, shortActual
	}
	return fmt.Sprintf("%s: expected %q, got %q", summary, expected, actual)
}
