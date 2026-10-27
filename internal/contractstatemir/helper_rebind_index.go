package contractstatemir

import "github.com/0disoft/fairyc/internal/hir"

// currentHelperHIRIndex belongs to one rebind operation and one fixed HIR package.
// Build only when helper validation is reached, then share across transactions.
// Empty payloads and earlier validation failures do not scan the current HIR.
type currentHelperHIRIndex struct {
	bySymbol  map[hir.SymbolID]string
	byName    map[string]hir.Function
	functions map[hir.SymbolID]hir.Function
}

func (index *currentHelperHIRIndex) prepare(input hir.Package, packageID hir.PackageID) {
	if index.functions != nil {
		return
	}
	index.bySymbol = make(map[hir.SymbolID]string)
	index.byName = make(map[string]hir.Function)
	index.functions = make(map[hir.SymbolID]hir.Function)
	for _, module := range input.Modules {
		if module.Package != packageID {
			continue
		}
		for _, function := range module.Functions {
			name := module.Path + "::" + function.Name
			index.bySymbol[function.Symbol] = name
			index.byName[name] = function
			index.functions[function.Symbol] = function
		}
	}
}
