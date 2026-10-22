package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (a *analyzer) prepareCEnumBase(module *moduleScope, enumeration *ast.Enum) string {
	base := ""
	seen := false
	for _, attribute := range enumeration.Attributes {
		if attribute.Name != "repr" {
			continue
		}
		if seen || len(attribute.Arguments) != 1 {
			a.addUnsupported(module.file.Path, attribute.Span, "C enum requires one @repr(c::integer) attribute")
			continue
		}
		seen = true
		candidate := strings.ReplaceAll(attribute.Arguments[0].Raw, "::", ".")
		capacity := hir.CEnumVariantCapacity(candidate)
		if capacity == 0 || len(enumeration.Variants) == 0 || uint64(len(enumeration.Variants)) > capacity {
			a.addUnsupported(module.file.Path, attribute.Span, "C enum requires a nonempty variant set fitting its C integer base")
			continue
		}
		base = candidate
		for _, variant := range enumeration.Variants {
			if len(variant.Fields) != 0 {
				a.addUnsupported(module.file.Path, variant.Span, "C enum payload fields")
			}
		}
	}
	return base
}
