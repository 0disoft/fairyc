package genericsource

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/syntax"
)

// PrepareStorageModule enables compiler-private typed layout queries. Neither
// ordinary generic expansion nor the stable package lane calls this entry point.
func PrepareStorageModule(module *ast.Module, owner genericplan.PackageID, limits genericplan.PlanLimits) (PackageExpansion, error) {
	if module == nil || len(module.Imports) != 0 {
		return PackageExpansion{}, fmt.Errorf("private storage queries require an import-free module")
	}
	return prepareModule(module, owner, limits, true, true, false)
}

// PrepareStorageTransferModule also enables unsafe source ownership transfers.
func PrepareStorageTransferModule(module *ast.Module, owner genericplan.PackageID, limits genericplan.PlanLimits) (PackageExpansion, error) {
	if module == nil || len(module.Imports) != 0 {
		return PackageExpansion{}, fmt.Errorf("private storage transfers require an import-free module")
	}
	return prepareModule(module, owner, limits, true, true, true)
}

func (x *expander) storageSelector(name string) string {
	if x.allowStorageTransfers && name == "storage_take" {
		return "take"
	}
	if x.allowStorageTransfers && name == "storage_put" {
		return "put"
	}
	if x.allowStorageTransfers && name == "storage_borrow" {
		return "borrow"
	}
	if x.allowStorageTransfers && name == "storage_borrow_mut" {
		return "borrow-mut"
	}
	if x.allowStorageTransfers && name == "storage_element_address" {
		return "element-address"
	}
	if !x.allowStorage {
		return ""
	}
	switch name {
	case "storage_size":
		return "size"
	case "storage_alignment":
		return "alignment"
	}
	return ""
}

const storageType syntax.Kind = "private-storage-type"

func (x *expander) rewriteStorageQuery(e *ast.Expression) (bool, error) {
	if e.Kind == syntax.KindSpecialize && len(e.Children) == 1 {
		selector := x.storageSelector(e.Children[0].Text)
		if selector != "" {
			if len(e.TypeArguments) != 1 {
				return true, fmt.Errorf("private storage query requires one type argument")
			}
			typ := e.TypeArguments[0]
			*e = ast.Expression{Kind: storageType, Span: e.Span, Text: selector, Type: &typ}
			return true, nil
		}
	}
	if e.Kind == syntax.KindCall && len(e.Children) != 0 && e.Children[0].Kind == storageType {
		callee := e.Children[0]
		if callee.Text == "borrow" || callee.Text == "borrow-mut" {
			if len(e.Children) != 3 {
				return true, fmt.Errorf("private storage borrow requires an owner borrow and a pointer")
			}
			*e = ast.Expression{Kind: ast.PrivateStorageBorrow, Span: e.Span, Text: callee.Text, Type: callee.Type, Children: e.Children[1:]}
			return true, nil
		}
		if callee.Text == "element-address" {
			if len(e.Children) != 3 {
				return true, fmt.Errorf("private storage element address requires a mutable void pointer and index")
			}
			*e = ast.Expression{Kind: ast.PrivateStorageElementAddress, Span: e.Span, Text: callee.Text, Type: callee.Type, Children: e.Children[1:]}
			return true, nil
		}
		if callee.Text == "put" {
			if len(e.Children) != 3 {
				return true, fmt.Errorf("private storage put requires a pointer and a value")
			}
			*e = ast.Expression{Kind: ast.PrivateStoragePut, Span: e.Span, Text: "put", Type: callee.Type, Children: e.Children[1:]}
			return true, nil
		}
		if callee.Text == "take" {
			if len(e.Children) != 2 {
				return true, fmt.Errorf("private storage take requires one pointer argument")
			}
			*e = ast.Expression{Kind: ast.PrivateStorageTake, Span: e.Span, Text: "take", Type: callee.Type, Children: e.Children[1:]}
			return true, nil
		}
		if len(e.Children) != 1 {
			return true, fmt.Errorf("private storage query takes no value arguments")
		}
		*e = ast.Expression{Kind: ast.PrivateStorageLayout, Span: e.Span, Text: callee.Text, Type: callee.Type}
		return true, nil
	}
	return false, nil
}
