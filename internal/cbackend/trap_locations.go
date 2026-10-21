package cbackend

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
)

func trapLocationArguments(span hir.Span) (string, bool) {
	if len(span.Source) == 0 || len(span.Source) > 256 || !utf8.ValidString(span.Source) ||
		strings.ContainsRune(span.Source, 0) || span.StartByte < 0 || span.EndByte < span.StartByte {
		return "", false
	}
	// Fixed-width octal escapes preserve exact UTF-8 bytes and cannot terminate
	// the C literal, even for quotes, backslashes, control bytes or digit suffixes.
	var source strings.Builder
	for _, b := range []byte(span.Source) {
		fmt.Fprintf(&source, "\\%03o", b)
	}
	return fmt.Sprintf("\"%s\", UINT32_C(%d), UINT64_C(%d), UINT64_C(%d)",
		source.String(), len(span.Source), span.StartByte, span.EndByte), true
}

// PackageTrapSource names a package-local file within one compilation. It is a
// private receipt key, not a portable package identity or a filesystem path.
func PackageTrapSource(packageID hir.PackageID, source string) string {
	if packageID == hir.DefaultPackage || source == "" {
		return ""
	}
	return fmt.Sprintf("p%d/%s", packageID, source)
}

func (e *emitter) trapLocationArguments(span hir.Span) (string, bool) {
	if e.analysis.profile.PackageTrapSources {
		span.Source = PackageTrapSource(span.Package, span.Source)
	}
	return trapLocationArguments(span)
}

func (e *emitter) trapCall(category string, span hir.Span) string {
	if arguments, valid := e.trapLocationArguments(span); valid {
		return "fy_runtime_trap_at_v0(" + category + ", " + arguments + ")"
	}
	return "fy_runtime_trap(" + category + ")"
}
