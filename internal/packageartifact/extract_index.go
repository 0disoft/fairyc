package packageartifact

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

type functionMetadata struct {
	id         FunctionID
	descriptor FunctionDescriptor
	signature  FunctionSignature
	body       mir.Function
}

// extractionIndex belongs to one extraction operation with a fixed program and
// coordinate map. Initialize after owner checks to preserve diagnostic order.
type extractionIndex struct {
	bySymbol map[hir.SymbolID]functionMetadata
}

func (index *extractionIndex) load(program mir.Program, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.SymbolID]functionMetadata, error) {
	if index.bySymbol != nil {
		return index.bySymbol, nil
	}
	bySymbol := make(map[hir.SymbolID]functionMetadata, len(program.Functions))
	for _, function := range program.Functions {
		descriptor, err := DescribeFunction(function, coordinates)
		if err != nil {
			return nil, err
		}
		id, err := ComputeFunctionID(descriptor)
		if err != nil {
			return nil, fmt.Errorf("compute FunctionID for %q: %w", function.Name, err)
		}
		if _, duplicate := bySymbol[function.Symbol]; duplicate {
			return nil, fmt.Errorf("whole program function symbol %d is duplicated", function.Symbol)
		}
		bySymbol[function.Symbol] = functionMetadata{
			id: id, descriptor: cloneDescriptor(descriptor), signature: signatureOf(function), body: function,
		}
	}
	index.bySymbol = bySymbol
	return bySymbol, nil
}
