package contractstatemir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

// PackageLayout binds one build-local package to its exact compiler-private
// persistent-state layout.
type PackageLayout struct {
	Package hir.PackageID
	Layout  Layout
}

// LowerHIRLayoutForPackage lowers only one package-owned layout. Malformed
// layouts owned by other packages remain the responsibility of their own
// interface extraction.
func LowerHIRLayoutForPackage(input hir.Package, packageID hir.PackageID) (Layout, bool, error) {
	selected := hir.Package{}
	for _, definition := range input.StateLayouts {
		if definition.Package == packageID {
			selected.StateLayouts = append(selected.StateLayouts, definition)
		}
	}
	lowered, err := LowerHIRLayouts(selected)
	if err != nil {
		return Layout{}, false, err
	}
	if len(lowered) == 0 {
		return Layout{}, false, nil
	}
	return lowered[0].Layout, true, nil
}

// LowerHIRLayouts validates and lowers every source-independent HIR layout in
// package order. At most one contract layout may belong to each package.
func LowerHIRLayouts(input hir.Package) ([]PackageLayout, error) {
	definitions := slices.Clone(input.StateLayouts)
	slices.SortFunc(definitions, func(left, right hir.StateLayout) int {
		if left.Package < right.Package {
			return -1
		}
		if left.Package > right.Package {
			return 1
		}
		return 0
	})
	result := make([]PackageLayout, 0, len(definitions))
	for index, definition := range definitions {
		if index > 0 && definitions[index-1].Package == definition.Package {
			return nil, fmt.Errorf("package %d declares multiple contract state layouts", definition.Package)
		}
		declarations := make([]Declaration, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			kind, err := lowerFieldKind(field.Kind)
			if err != nil {
				return nil, fmt.Errorf("package %d state field %q: %w", definition.Package, field.Name, err)
			}
			declarations = append(declarations, Declaration{
				Name: field.Name, Kind: kind, KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
			})
		}
		layout, err := BuildLayout(definition.ContractUID, declarations)
		if err != nil {
			return nil, fmt.Errorf("package %d contract state layout: %w", definition.Package, err)
		}
		result = append(result, PackageLayout{Package: definition.Package, Layout: layout})
	}
	return result, nil
}

func lowerFieldKind(kind hir.StateFieldKind) (contractstate.FieldKind, error) {
	switch kind {
	case hir.StateFieldCell:
		return contractstate.FieldCell, nil
	case hir.StateFieldMap:
		return contractstate.FieldMap, nil
	default:
		return 0, fmt.Errorf("unsupported field kind %q", kind)
	}
}
