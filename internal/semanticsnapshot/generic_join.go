package semanticsnapshot

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func joinGenericImplementations(provenance compiler.GenericProvenance, originals, concrete []Declaration, metadata map[hir.PackageID]packageMetadata) ([]Declaration, map[DeclarationKey]SpecializationSummary, error) {
	byKey := map[DeclarationKey]Declaration{}
	byID := map[genericplan.DeclarationID]Declaration{}
	for _, declaration := range originals {
		byKey[declaration.Key] = declaration
	}
	for _, origin := range provenance.Declarations() {
		key := DeclarationKey{PackageInstance: metadata[origin.Package].instance, Module: origin.Module, Kind: DeclarationKind(origin.Kind), Name: origin.Name}
		declaration, ok := byKey[key]
		if !ok {
			return nil, nil, fmt.Errorf("generic origin has no source declaration")
		}
		if _, duplicate := byID[origin.ID]; duplicate {
			return nil, nil, fmt.Errorf("generic origin identity is duplicated")
		}
		byID[origin.ID] = declaration
	}
	concreteByKey := map[DeclarationKey]Declaration{}
	for _, declaration := range concrete {
		concreteByKey[declaration.Key] = declaration
	}
	implementations := map[DeclarationKey]SpecializationSummary{}
	typeNames := map[string]TypeRef{}
	ids := map[string]bool{}
	for _, origin := range provenance.Specializations() {
		original, ok := byID[origin.Declaration]
		if !ok || origin.GeneratedName != "_fy_generic_"+string(origin.Specialization) || ids[string(origin.Specialization)] {
			return nil, nil, fmt.Errorf("specialization has an invalid or duplicated source binding")
		}
		ids[string(origin.Specialization)] = true
		key := original.Key
		key.Name = origin.GeneratedName
		actual, ok := concreteByKey[key]
		if !ok || actual.Span != original.Span || actual.Foreign || actual.Owner != nil || actual.Unsafe != original.Unsafe || !slices.Equal(actual.Effects, original.Effects) {
			return nil, nil, fmt.Errorf("specialization has no matching typed declaration")
		}
		arguments := make([]TypeRef, len(original.TypeParameters))
		bound := make([]bool, len(arguments))
		if len(origin.Bindings) != len(arguments) {
			return nil, nil, fmt.Errorf("specialization binding arity mismatch")
		}
		for _, binding := range origin.Bindings {
			index := slices.Index(original.TypeParameters, binding.Parameter)
			if index < 0 || bound[index] {
				return nil, nil, fmt.Errorf("specialization binding does not match an original parameter")
			}
			argument, err := normalizeType(hir.ParseType(string(binding.Type)), metadata)
			if err != nil || argument.Kind != TypeBuiltin {
				return nil, nil, fmt.Errorf("specialization argument has no admitted concrete type")
			}
			arguments[index], bound[index] = argument, true
		}
		if original.Key.Kind != DeclarationFunction {
			typ, err := substituteGenericType(original.Type, arguments)
			if err != nil || actual.Type.Kind != TypeNominal {
				return nil, nil, fmt.Errorf("aggregate specialization has no nominal type")
			}
			typeNames[genericNominalKey(actual.Type)] = typ
		}
		implementations[key] = SpecializationSummary{ID: string(origin.Specialization), Declaration: original.Key, TypeArguments: arguments}
	}
	for i := range concrete {
		value, err := remapGenericDeclaration(concrete[i], typeNames)
		if err != nil {
			return nil, nil, err
		}
		concreteByKey[value.Key] = value
	}
	for key, implementation := range implementations {
		original := byKey[implementation.Declaration]
		actual := concreteByKey[key]
		arguments := implementation.TypeArguments
		expected, err := substituteGenericType(original.Type, arguments)
		if err != nil || !sameSnapshotType(expected, actual.Type) || len(original.Parameters) != len(actual.Parameters) || len(original.Fields) != len(actual.Fields) {
			return nil, nil, fmt.Errorf("specialization result or parameter arity contradicts its source signature")
		}
		for i, parameter := range original.Parameters {
			expected, err := substituteGenericType(parameter.Type, arguments)
			if err != nil || !sameSnapshotType(expected, actual.Parameters[i].Type) || parameter.Name != actual.Parameters[i].Name || parameter.Span != actual.Parameters[i].Span {
				return nil, nil, fmt.Errorf("specialization parameter contradicts its source signature")
			}
		}
		for i, field := range original.Fields {
			expected, err := substituteGenericType(field.Type, arguments)
			if err != nil || !sameSnapshotType(expected, actual.Fields[i].Type) || field.Name != actual.Fields[i].Name || !containsSpan(actual.Fields[i].Span, field.Span) {
				return nil, nil, fmt.Errorf("aggregate field contradicts its source signature")
			}
		}
		if original.Key.Kind != DeclarationFunction {
			actual.Key = original.Key
			actual.Provenance = ProvenanceCompilerObserved
			implementation.Aggregate = &actual
			implementations[key] = implementation
		}
	}
	declarations := slices.Clone(originals)
	for _, declaration := range concrete {
		if _, generated := implementations[declaration.Key]; generated {
			continue
		}
		if strings.HasPrefix(declaration.Key.Name, "_fy_generic_") {
			return nil, nil, fmt.Errorf("typed generic implementation has no compiler provenance")
		}
		if _, collision := byKey[declaration.Key]; collision {
			return nil, nil, fmt.Errorf("original generic declaration collides with a typed source declaration")
		}
		declarations = append(declarations, concreteByKey[declaration.Key])
	}
	slices.SortFunc(declarations, func(a, b Declaration) int { return compareDeclarationKey(a.Key, b.Key) })
	return declarations, implementations, nil
}

func substituteGenericType(value TypeRef, arguments []TypeRef) (TypeRef, error) {
	if value.Kind == TypeParameter {
		index, err := strconv.Atoi(value.Name)
		if err != nil || index < 0 || index >= len(arguments) {
			return TypeRef{}, fmt.Errorf("unbound generic type parameter")
		}
		return cloneType(arguments[index]), nil
	}
	result := value
	result.Arguments = make([]TypeRef, len(value.Arguments))
	for i, child := range value.Arguments {
		var err error
		result.Arguments[i], err = substituteGenericType(child, arguments)
		if err != nil {
			return TypeRef{}, err
		}
	}
	return result, nil
}

func sameSnapshotType(a, b TypeRef) bool {
	return a.Kind == b.Kind && a.Name == b.Name && a.PackageCoordinate == b.PackageCoordinate && a.Mutable == b.Mutable &&
		a.Length == b.Length && slices.EqualFunc(a.Arguments, b.Arguments, sameSnapshotType)
}
