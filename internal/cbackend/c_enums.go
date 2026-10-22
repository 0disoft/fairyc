package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

func (e *emitter) emitCEnumType(output *strings.Builder, index int) error {
	enumeration := e.analysis.enumTypes[index]
	transport, ok := cScalarTransportForType(hir.ParseType(strings.ReplaceAll(enumeration.definition.CBase, ".", "::")))
	if !ok {
		return fmt.Errorf("unsupported C enum base %q", enumeration.definition.CBase)
	}
	fmt.Fprintf(output, "typedef %s fy_enum_%d;\n", transport.cType, index)
	fmt.Fprintf(output, "_Static_assert(sizeof(fy_enum_%d) == UINT64_C(%d), \"fairyc C enum %d size\");\n", index, enumeration.layout.SizeBytes, index)
	fmt.Fprintf(output, "_Static_assert(_Alignof(fy_enum_%d) == UINT64_C(%d), \"fairyc C enum %d alignment\");\n\n", index, enumeration.layout.AlignmentBytes, index)
	return nil
}
