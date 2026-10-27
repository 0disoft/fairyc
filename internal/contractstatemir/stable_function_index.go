package contractstatemir

import "github.com/0disoft/fairyc/internal/hir"

type stableFunctionKey struct {
	modulePath   string
	functionName string
}

// currentStableFunctionIndex is local to one rebind and one fixed package.
// Defer ambiguity errors until lookup so transaction diagnostic order stays intact.
// A nil index scans directly, avoiding map allocation for a single transaction.
type currentStableFunctionIndex struct {
	entries map[stableFunctionKey]transactionOwner
}

func (index *currentStableFunctionIndex) find(input hir.Package, packageID hir.PackageID, key stableFunctionKey) transactionOwner {
	if index == nil {
		return scanStableFunctionOwner(input, packageID, key)
	}
	if index.entries == nil {
		index.prepare(input, packageID)
	}
	return index.entries[key]
}

func (index *currentStableFunctionIndex) prepare(input hir.Package, packageID hir.PackageID) {
	count := 0
	for _, module := range input.Modules {
		if module.Package == packageID {
			count += len(module.Functions)
		}
	}
	index.entries = make(map[stableFunctionKey]transactionOwner, count)
	for mi := range input.Modules {
		module := &input.Modules[mi]
		if module.Package != packageID {
			continue
		}
		for fi := range module.Functions {
			function := &module.Functions[fi]
			key := stableFunctionKey{module.Path, function.Name}
			owner := index.entries[key]
			if owner.function != nil {
				owner.ambiguous = true
			} else {
				owner.module, owner.function = module, function
			}
			index.entries[key] = owner
		}
	}
}

func scanStableFunctionOwner(input hir.Package, packageID hir.PackageID, key stableFunctionKey) transactionOwner {
	var owner transactionOwner
	for mi := range input.Modules {
		module := &input.Modules[mi]
		if module.Package != packageID || module.Path != key.modulePath {
			continue
		}
		for fi := range module.Functions {
			function := &module.Functions[fi]
			if function.Name != key.functionName {
				continue
			}
			if owner.function != nil {
				owner.ambiguous = true
				return owner
			}
			owner.module, owner.function = module, function
		}
	}
	return owner
}
