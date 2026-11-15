package packageartifact

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func rebindContextDeclarations(descriptor ContextDescriptor, resolve packageResolver, functionSymbols map[FunctionID]hir.SymbolID, boundSymbols map[hir.SymbolID]string, firstForeignSymbol hir.SymbolID) (mir.Program, map[hir.SymbolID]FunctionID, error) {
	program := mir.Program{}
	for _, item := range descriptor.Structs {
		structure, err := rebindContextStruct(item, resolve)
		if err != nil {
			return mir.Program{}, nil, err
		}
		program.StructTypes = append(program.StructTypes, structure)
	}
	for _, item := range descriptor.Enums {
		enumeration, err := rebindContextEnum(item, resolve)
		if err != nil {
			return mir.Program{}, nil, err
		}
		program.EnumTypes = append(program.EnumTypes, enumeration)
	}
	bindings := make(map[hir.SymbolID]FunctionID, len(descriptor.Resources))
	for _, item := range descriptor.Resources {
		resource, err := rebindContextResource(item, resolve, functionSymbols)
		if err != nil {
			return mir.Program{}, nil, err
		}
		if resource.DropSymbol != hir.NoSymbol {
			bindings[resource.DropSymbol] = item.DropFunctionID
		}
		program.ResourceTypes = append(program.ResourceTypes, resource)
	}
	foreign, err := rebindContextForeignFunctions(descriptor.ForeignFunctions, resolve, boundSymbols, firstForeignSymbol)
	if err != nil {
		return mir.Program{}, nil, err
	}
	program.ForeignFunctions = foreign
	return program, bindings, nil
}

func rebindContextStruct(item ContextStruct, resolve packageResolver) (mir.StructType, error) {
	typeRef, err := rebindStableTypeWithResolver(item.Type, resolve)
	if err != nil {
		return mir.StructType{}, fmt.Errorf("rebind struct %q type: %w", item.Type, err)
	}
	owner, err := packageForStableNominal(item.Type, resolve)
	if err != nil {
		return mir.StructType{}, err
	}
	structure := mir.StructType{Type: typeRef, Name: item.Name, Span: hir.Span{Package: owner}, ReprC: item.ReprC}
	for _, field := range item.Fields {
		fieldType, err := rebindStableTypeWithResolver(field.Type, resolve)
		if err != nil {
			return mir.StructType{}, fmt.Errorf("rebind struct %q field %q: %w", item.Type, field.Name, err)
		}
		structure.Fields = append(structure.Fields, mir.StructField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
	}
	return structure, nil
}

func rebindContextEnum(item ContextEnum, resolve packageResolver) (mir.EnumType, error) {
	typeRef, err := rebindStableTypeWithResolver(item.Type, resolve)
	if err != nil {
		return mir.EnumType{}, fmt.Errorf("rebind enum %q type: %w", item.Type, err)
	}
	owner, err := packageForStableNominal(item.Type, resolve)
	if err != nil {
		return mir.EnumType{}, err
	}
	enumeration := mir.EnumType{Type: typeRef, Name: item.Name, CBase: item.CBase, Span: hir.Span{Package: owner}}
	for _, variant := range item.Variants {
		reboundVariant, err := rebindContextVariant(item.Type, variant, owner, resolve)
		if err != nil {
			return mir.EnumType{}, err
		}
		enumeration.Variants = append(enumeration.Variants, reboundVariant)
	}
	return enumeration, nil
}

func rebindContextVariant(enumType string, variant ContextVariant, owner hir.PackageID, resolve packageResolver) (mir.EnumVariant, error) {
	reboundVariant := mir.EnumVariant{Name: variant.Name, Span: hir.Span{Package: owner}}
	for _, field := range variant.Fields {
		fieldType, err := rebindStableTypeWithResolver(field.Type, resolve)
		if err != nil {
			return mir.EnumVariant{}, fmt.Errorf("rebind enum %q variant %q field %q: %w", enumType, variant.Name, field.Name, err)
		}
		reboundVariant.Fields = append(reboundVariant.Fields, mir.EnumField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
	}
	return reboundVariant, nil
}

func rebindContextResource(item ContextResource, resolve packageResolver, functionSymbols map[FunctionID]hir.SymbolID) (mir.ResourceType, error) {
	typeRef, err := rebindStableTypeWithResolver(item.Type, resolve)
	if err != nil {
		return mir.ResourceType{}, fmt.Errorf("rebind resource %q type: %w", item.Type, err)
	}
	owner, err := packageForStableNominal(item.Type, resolve)
	if err != nil {
		return mir.ResourceType{}, err
	}
	resource := mir.ResourceType{Type: typeRef, Span: hir.Span{Package: owner}, DropKind: hir.ResourceDropKind(item.DropKind), DropSymbol: hir.NoSymbol}
	for _, field := range item.Fields {
		fieldType, err := rebindStableTypeWithResolver(field.Type, resolve)
		if err != nil {
			return mir.ResourceType{}, fmt.Errorf("rebind resource %q field %q: %w", item.Type, field.Name, err)
		}
		resource.Fields = append(resource.Fields, mir.ResourceField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
	}
	resource.DropSymbol, resource.DropEffects, err = rebindContextResourceDrop(item, functionSymbols)
	if err != nil {
		return mir.ResourceType{}, err
	}
	return resource, nil
}

func rebindContextResourceDrop(item ContextResource, functionSymbols map[FunctionID]hir.SymbolID) (hir.SymbolID, []hir.Effect, error) {
	var effects []hir.Effect
	for _, effect := range item.DropEffects {
		parsed, ok := hir.ParseEffect(effect)
		if !ok {
			return hir.NoSymbol, nil, fmt.Errorf("rebind resource %q has unknown effect %q", item.Type, effect)
		}
		effects = append(effects, parsed)
	}
	if item.DropFunctionID == (FunctionID{}) {
		return hir.NoSymbol, effects, nil
	}
	symbol, found := functionSymbols[item.DropFunctionID]
	if !found || symbol == hir.NoSymbol {
		return hir.NoSymbol, nil, fmt.Errorf("stable resource %q drop target %s is unresolved", item.Type, item.DropFunctionID)
	}
	return symbol, effects, nil
}

func rebindContextForeignFunctions(items []ContextForeignFunction, resolve packageResolver, boundSymbols map[hir.SymbolID]string, firstSymbol hir.SymbolID) ([]mir.ForeignFunction, error) {
	foreignFunctions := slices.Grow([]mir.ForeignFunction(nil), len(items))
	for index, item := range items {
		owner, err := packageForCoordinateWithResolver(item.Package, resolve)
		if err != nil {
			return nil, err
		}
		symbol := firstSymbol + hir.SymbolID(index)
		if symbol < firstSymbol {
			return nil, fmt.Errorf("stable foreign function symbol namespace overflows")
		}
		// Non-overflowing consecutive symbols cannot collide with earlier foreigns;
		// only caller-supplied function bindings need collision tracking.
		if previous, collision := boundSymbols[symbol]; collision {
			return nil, fmt.Errorf("foreign symbol %d collides with %s", symbol, previous)
		}
		foreign, err := rebindContextForeign(item, owner, symbol, resolve)
		if err != nil {
			return nil, err
		}
		foreignFunctions = append(foreignFunctions, foreign)
	}
	return foreignFunctions, nil
}

func rebindContextForeign(item ContextForeignFunction, owner hir.PackageID, symbol hir.SymbolID, resolve packageResolver) (mir.ForeignFunction, error) {
	foreign := mir.ForeignFunction{
		Symbol: symbol, Name: hir.QualifiedModulePath(owner, item.Name), LinkName: item.LinkName,
		ABI: item.ABI, Span: hir.Span{Package: owner}, ResultType: hir.TypeInvalid,
	}
	for _, parameterType := range item.ParameterTypes {
		rebound, err := rebindStableTypeWithResolver(parameterType, resolve)
		if err != nil {
			return mir.ForeignFunction{}, fmt.Errorf("rebind foreign function %q parameter: %w", item.Name, err)
		}
		foreign.ParameterTypes = append(foreign.ParameterTypes, rebound)
	}
	result, err := rebindStableTypeWithResolver(item.ResultType, resolve)
	if err != nil {
		return mir.ForeignFunction{}, fmt.Errorf("rebind foreign function %q result: %w", item.Name, err)
	}
	foreign.ResultType = result
	return foreign, nil
}
