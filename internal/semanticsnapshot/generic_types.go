package semanticsnapshot

import (
	"fmt"
	"strings"
)

func genericNominalKey(value TypeRef) string { return value.PackageCoordinate + "\x00" + value.Name }

func remapGenericDeclaration(value Declaration, names map[string]TypeRef) (Declaration, error) {
	value = cloneDeclaration(value)
	var convert func(TypeRef) (TypeRef, error)
	convert = func(typ TypeRef) (TypeRef, error) {
		if typ.Kind == TypeNominal {
			if original, found := names[genericNominalKey(typ)]; found {
				return cloneType(original), nil
			}
			if strings.Contains(typ.Name, "_fy_generic_") {
				return TypeRef{}, fmt.Errorf("generated nominal type has no source provenance")
			}
		}
		for i, child := range typ.Arguments {
			converted, err := convert(child)
			if err != nil {
				return TypeRef{}, err
			}
			typ.Arguments[i] = converted
		}
		return typ, nil
	}
	var err error
	value.Type, err = convert(value.Type)
	if err != nil {
		return Declaration{}, err
	}
	if value.Owner != nil {
		typ, err := convert(*value.Owner)
		if err != nil {
			return Declaration{}, err
		}
		value.Owner = &typ
	}
	for i := range value.Parameters {
		value.Parameters[i].Type, err = convert(value.Parameters[i].Type)
		if err != nil {
			return Declaration{}, err
		}
	}
	for i := range value.Fields {
		value.Fields[i].Type, err = convert(value.Fields[i].Type)
		if err != nil {
			return Declaration{}, err
		}
	}
	for i := range value.Variants {
		for j := range value.Variants[i].Fields {
			value.Variants[i].Fields[j].Type, err = convert(value.Variants[i].Fields[j].Type)
			if err != nil {
				return Declaration{}, err
			}
		}
	}
	return value, nil
}
