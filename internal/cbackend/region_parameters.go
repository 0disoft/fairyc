package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// Input capabilities refer to the caller's existing handle. Lexical handles
// remain stack objects; forwarding does not allocate, copy, begin, or reset one.
func (e *emitter) regionPointer(functionIndex int, symbol hir.SymbolID) string {
	function := &e.analysis.functions[functionIndex]
	for index, local := range function.Parameters {
		if hir.SymbolID(local) == symbol && function.ParameterTypes[index] == hir.TypeRegion {
			return fmt.Sprintf("fy_l_%d", local)
		}
	}
	return fmt.Sprintf("&fy_region_%d", symbol)
}
