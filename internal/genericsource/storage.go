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
	return prepareModule(module, owner, limits, true, true)
}

func storageSelector(name string) string {
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
		selector := storageSelector(e.Children[0].Text)
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
		if len(e.Children) != 1 {
			return true, fmt.Errorf("private storage query takes no value arguments")
		}
		callee := e.Children[0]
		*e = ast.Expression{Kind: ast.PrivateStorageLayout, Span: e.Span, Text: callee.Text, Type: callee.Type}
		return true, nil
	}
	return false, nil
}
