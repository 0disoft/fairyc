package packageartifact

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// RebindContextDescriptor reconstructs process-local MIR type and foreign declarations solely
// from a stable descriptor. Internal drop functions and foreign declarations receive explicit,
// collision-free caller-owned symbols; no source span or entity is invented as semantic truth.
func RebindContextDescriptor(descriptor ContextDescriptor, identities map[hir.PackageID]packagegraph.InstanceID, functionSymbols map[FunctionID]hir.SymbolID, firstForeignSymbol hir.SymbolID) (mir.Program, error) {
	if err := ValidateContextDescriptor(descriptor); err != nil {
		return mir.Program{}, err
	}
	if firstForeignSymbol == hir.NoSymbol {
		return mir.Program{}, fmt.Errorf("stable type-context rebind requires a foreign symbol base")
	}
	usedSymbols := make(map[hir.SymbolID]string, len(functionSymbols)+len(descriptor.ForeignFunctions))
	for id, symbol := range functionSymbols {
		if id == (FunctionID{}) || symbol == hir.NoSymbol {
			return mir.Program{}, fmt.Errorf("stable type-context rebind has an invalid function symbol binding")
		}
		if previous, duplicate := usedSymbols[symbol]; duplicate {
			return mir.Program{}, fmt.Errorf("function symbol %d is duplicated by %s", symbol, previous)
		}
		usedSymbols[symbol] = id.String()
	}

	program := mir.Program{}
	for _, item := range descriptor.Structs {
		typeRef, err := rebindStableType(item.Type, identities)
		if err != nil {
			return mir.Program{}, fmt.Errorf("rebind struct %q type: %w", item.Type, err)
		}
		owner, err := packageForStableNominal(item.Type, identities)
		if err != nil {
			return mir.Program{}, err
		}
		structure := mir.StructType{Type: typeRef, Name: item.Name, Span: hir.Span{Package: owner}, ReprC: item.ReprC}
		for _, field := range item.Fields {
			fieldType, fieldErr := rebindStableType(field.Type, identities)
			if fieldErr != nil {
				return mir.Program{}, fmt.Errorf("rebind struct %q field %q: %w", item.Type, field.Name, fieldErr)
			}
			structure.Fields = append(structure.Fields, mir.StructField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
		}
		program.StructTypes = append(program.StructTypes, structure)
	}
	for _, item := range descriptor.Enums {
		typeRef, err := rebindStableType(item.Type, identities)
		if err != nil {
			return mir.Program{}, fmt.Errorf("rebind enum %q type: %w", item.Type, err)
		}
		owner, err := packageForStableNominal(item.Type, identities)
		if err != nil {
			return mir.Program{}, err
		}
		enumeration := mir.EnumType{Type: typeRef, Name: item.Name, Span: hir.Span{Package: owner}}
		for _, variant := range item.Variants {
			reboundVariant := mir.EnumVariant{Name: variant.Name, Span: hir.Span{Package: owner}}
			for _, field := range variant.Fields {
				fieldType, fieldErr := rebindStableType(field.Type, identities)
				if fieldErr != nil {
					return mir.Program{}, fmt.Errorf("rebind enum %q variant %q field %q: %w", item.Type, variant.Name, field.Name, fieldErr)
				}
				reboundVariant.Fields = append(reboundVariant.Fields, mir.EnumField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
			}
			enumeration.Variants = append(enumeration.Variants, reboundVariant)
		}
		program.EnumTypes = append(program.EnumTypes, enumeration)
	}
	for _, item := range descriptor.Resources {
		typeRef, err := rebindStableType(item.Type, identities)
		if err != nil {
			return mir.Program{}, fmt.Errorf("rebind resource %q type: %w", item.Type, err)
		}
		owner, err := packageForStableNominal(item.Type, identities)
		if err != nil {
			return mir.Program{}, err
		}
		resource := mir.ResourceType{Type: typeRef, Span: hir.Span{Package: owner}, DropKind: hir.ResourceDropKind(item.DropKind), DropSymbol: hir.NoSymbol}
		for _, field := range item.Fields {
			fieldType, fieldErr := rebindStableType(field.Type, identities)
			if fieldErr != nil {
				return mir.Program{}, fmt.Errorf("rebind resource %q field %q: %w", item.Type, field.Name, fieldErr)
			}
			resource.Fields = append(resource.Fields, mir.ResourceField{Name: field.Name, Type: fieldType, Span: hir.Span{Package: owner}})
		}
		for _, effect := range item.DropEffects {
			parsed, ok := hir.ParseEffect(effect)
			if !ok {
				return mir.Program{}, fmt.Errorf("rebind resource %q has unknown effect %q", item.Type, effect)
			}
			resource.DropEffects = append(resource.DropEffects, parsed)
		}
		if item.DropFunctionID != (FunctionID{}) {
			symbol, found := functionSymbols[item.DropFunctionID]
			if !found || symbol == hir.NoSymbol {
				return mir.Program{}, fmt.Errorf("stable resource %q drop target %s is unresolved", item.Type, item.DropFunctionID)
			}
			resource.DropSymbol = symbol
		}
		program.ResourceTypes = append(program.ResourceTypes, resource)
	}
	for index, item := range descriptor.ForeignFunctions {
		owner, err := packageForInstance(item.Package, identities)
		if err != nil {
			return mir.Program{}, err
		}
		symbol := firstForeignSymbol + hir.SymbolID(index)
		if symbol < firstForeignSymbol {
			return mir.Program{}, fmt.Errorf("stable foreign function symbol namespace overflows")
		}
		if previous, collision := usedSymbols[symbol]; collision {
			return mir.Program{}, fmt.Errorf("foreign symbol %d collides with %s", symbol, previous)
		}
		usedSymbols[symbol] = item.Name
		foreign := mir.ForeignFunction{
			Symbol: symbol, Name: hir.QualifiedModulePath(owner, item.Name), LinkName: item.LinkName,
			ABI: item.ABI, Span: hir.Span{Package: owner}, ResultType: hir.TypeInvalid,
		}
		for _, parameterType := range item.ParameterTypes {
			rebound, parameterErr := rebindStableType(parameterType, identities)
			if parameterErr != nil {
				return mir.Program{}, fmt.Errorf("rebind foreign function %q parameter: %w", item.Name, parameterErr)
			}
			foreign.ParameterTypes = append(foreign.ParameterTypes, rebound)
		}
		foreign.ResultType, err = rebindStableType(item.ResultType, identities)
		if err != nil {
			return mir.Program{}, fmt.Errorf("rebind foreign function %q result: %w", item.Name, err)
		}
		program.ForeignFunctions = append(program.ForeignFunctions, foreign)
	}

	bindings := make(map[hir.SymbolID]FunctionID, len(descriptor.Resources))
	for _, resource := range program.ResourceTypes {
		if resource.DropSymbol == hir.NoSymbol {
			continue
		}
		stableTypeName, err := stableType(resource.Type, identities)
		if err != nil {
			return mir.Program{}, err
		}
		for _, item := range descriptor.Resources {
			if item.Type == stableTypeName {
				bindings[resource.DropSymbol] = item.DropFunctionID
				break
			}
		}
	}
	roundTrip, err := DescribeContext(program, identities, bindings)
	if err != nil {
		return mir.Program{}, fmt.Errorf("describe rebound stable type context: %w", err)
	}
	if !reflect.DeepEqual(roundTrip, descriptor) {
		return mir.Program{}, fmt.Errorf("rebound stable type context does not round-trip")
	}
	return program, nil
}

func packageForStableNominal(value string, identities map[hir.PackageID]packagegraph.InstanceID) (hir.PackageID, error) {
	const prefix = "pkg["
	separator := strings.Index(value, "]::")
	if !strings.HasPrefix(value, prefix) || separator <= len(prefix) {
		return 0, fmt.Errorf("stable nominal type %q is invalid", value)
	}
	instance, err := packagegraph.ParseInstanceID(value[len(prefix):separator])
	if err != nil {
		return 0, fmt.Errorf("parse stable nominal type %q: %w", value, err)
	}
	return packageForInstance(instance, identities)
}
