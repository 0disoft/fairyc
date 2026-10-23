package interpreter

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func regionParameter(function *mir.Function, symbol hir.SymbolID) bool {
	for index, local := range function.Parameters {
		if hir.SymbolID(local) == symbol && index < len(function.ParameterTypes) && function.ParameterTypes[index] == hir.TypeRegion {
			return true
		}
	}
	return false
}
