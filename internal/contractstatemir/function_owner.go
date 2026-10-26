package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

type transactionOwnerKey struct {
	packageID hir.PackageID
	symbol    hir.SymbolID
}

type transactionOwner struct {
	module    *hir.Module
	function  *hir.Function
	ambiguous bool
}

// transactionOwnerIndex belongs to one fixed HIR input. Only transaction owners
// are retained; unrelated duplicate functions remain outside this check's scope.
// Building on first lookup preserves earlier layout/transaction diagnostics and
// avoids scanning ordinary programs with no state transactions.
type transactionOwnerIndex struct {
	entries map[transactionOwnerKey]transactionOwner
}

func (index *transactionOwnerIndex) find(input hir.Package, packageID hir.PackageID, symbol hir.SymbolID) (hir.Module, hir.Function, error) {
	if index.entries == nil {
		entries := make(map[transactionOwnerKey]transactionOwner, len(input.StateTransactions))
		for _, transaction := range input.StateTransactions {
			entries[transactionOwnerKey{transaction.Package, transaction.Function}] = transactionOwner{}
		}
		var single transactionOwnerKey
		if len(input.StateTransactions) == 1 {
			transaction := input.StateTransactions[0]
			single = transactionOwnerKey{transaction.Package, transaction.Function}
		}
		for mi := range input.Modules {
			module := &input.Modules[mi]
			for fi := range module.Functions {
				function := &module.Functions[fi]
				key := transactionOwnerKey{module.Package, function.Symbol}
				if len(input.StateTransactions) == 1 && key != single {
					continue
				}
				owner, requested := entries[key]
				if !requested {
					continue
				}
				if owner.function != nil {
					owner.ambiguous = true
				} else {
					owner.module, owner.function = module, function
				}
				entries[key] = owner
			}
		}
		index.entries = entries
	}
	owner := index.entries[transactionOwnerKey{packageID, symbol}]
	if owner.ambiguous {
		return hir.Module{}, hir.Function{}, fmt.Errorf("package %d function symbol %d is duplicated", packageID, symbol)
	}
	if owner.function == nil {
		return hir.Module{}, hir.Function{}, fmt.Errorf("package %d state transaction names missing function %d", packageID, symbol)
	}
	return *owner.module, *owner.function, nil
}
