package symbolidentity

import (
	"fmt"
	"strconv"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
)

func validateGenericDeclaration(declaration semanticsnapshot.Declaration) error {
	arity := uint64(len(declaration.TypeParameters))
	if arity != 0 {
		kind := declaration.Key.Kind
		if (kind != semanticsnapshot.DeclarationFunction && kind != semanticsnapshot.DeclarationStruct && kind != semanticsnapshot.DeclarationResource) || declaration.Foreign || declaration.Owner != nil {
			return fmt.Errorf("generic identity requires a module function or aggregate")
		}
		if kind != semanticsnapshot.DeclarationFunction {
			if len(declaration.Parameters) != 0 || declaration.Type.Kind != semanticsnapshot.TypeNominal || declaration.Type.Name != declaration.Key.Module+"::"+declaration.Key.Name || uint64(len(declaration.Type.Arguments)) != arity || declaration.Type.PackageCoordinate == "" {
				return fmt.Errorf("generic aggregate identity has an invalid self type")
			}
			for i, argument := range declaration.Type.Arguments {
				if argument.Kind != semanticsnapshot.TypeParameter || argument.Name != strconv.Itoa(i) {
					return fmt.Errorf("generic aggregate self type has invalid binders")
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, name := range declaration.TypeParameters {
		if !source.ValidIdentifier(name) || seen[name] {
			return fmt.Errorf("generic identity has an invalid or duplicate parameter")
		}
		seen[name] = true
	}
	var validate func(semanticsnapshot.TypeRef) error
	validate = func(value semanticsnapshot.TypeRef) error {
		if value.Kind == semanticsnapshot.TypeParameter {
			index, err := strconv.ParseUint(value.Name, 10, 64)
			if err != nil || strconv.FormatUint(index, 10) != value.Name || index >= arity || value.PackageCoordinate != "" || value.Mutable || value.Length != 0 || len(value.Arguments) != 0 {
				return fmt.Errorf("generic identity has an invalid type-parameter reference")
			}
		}
		for _, argument := range value.Arguments {
			if err := validate(argument); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validate(declaration.Type); err != nil {
		return err
	}
	if declaration.Owner != nil {
		if err := validate(*declaration.Owner); err != nil {
			return err
		}
	}
	for _, parameter := range declaration.Parameters {
		if err := validate(parameter.Type); err != nil {
			return err
		}
	}
	for _, field := range declaration.Fields {
		if err := validate(field.Type); err != nil {
			return err
		}
	}
	for _, variant := range declaration.Variants {
		for _, field := range variant.Fields {
			if err := validate(field.Type); err != nil {
				return err
			}
		}
	}
	return nil
}
