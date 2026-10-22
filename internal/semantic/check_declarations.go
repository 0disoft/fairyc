package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (a *analyzer) checkEnum(module *moduleScope, enumeration *ast.Enum) hir.Enum {
	value := a.enums[enumeration]
	for _, attribute := range enumeration.Attributes {
		if attribute.Name == "repr" {
			continue // Validated while preparing enum identities and fields.
		}
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == mustUseAttribute {
			a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, enumeration.NameSpan)})
			continue
		}
		a.addUnsupported(module.file.Path, attribute.Span, "enum attribute @"+attribute.Name)
	}
	result := hir.Enum{
		Name: enumeration.Name, Public: enumeration.Public, Type: value.typ,
		Span:              span(module.file.Path, enumeration.Span),
		DocumentationSpan: optionalSpan(module.file.Path, enumeration.DocumentationSpan),
	}
	if info := a.enumTypes[value.typ]; info != nil {
		result.CBase = info.cBase
		for _, variant := range info.orderedVariants {
			hirVariant := hir.EnumVariant{Name: variant.name, Span: variant.span}
			for _, field := range variant.orderedFields {
				hirVariant.Fields = append(hirVariant.Fields, hir.EnumField{Name: field.name, Type: field.typ, Span: field.span})
			}
			result.Variants = append(result.Variants, hirVariant)
		}
	}
	return result
}

func (a *analyzer) checkStruct(module *moduleScope, structure *ast.Struct) hir.Struct {
	value := a.structs[structure]
	result := hir.Struct{
		Name: structure.Name, Public: structure.Public, Type: value.typ,
		Span:              span(module.file.Path, structure.Span),
		DocumentationSpan: optionalSpan(module.file.Path, structure.DocumentationSpan),
	}
	if info := a.structTypes[value.typ]; info != nil {
		result.ReprC = info.reprC
		for _, field := range info.orderedFields {
			result.Fields = append(result.Fields, hir.StructField{Name: field.name, Type: field.typ, Span: field.span})
		}
	}
	return result
}

func (a *analyzer) checkForeignFunction(module *moduleScope, function *ast.ForeignFunction) hir.ForeignFunction {
	symbol := a.foreignFunctions[function]
	linkName := function.Name
	seenLinkName := false
	for _, attribute := range function.Attributes {
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == effectsAttribute || attribute.Name == mustUseAttribute {
			continue
		}
		if attribute.Name != "link_name" {
			a.addUnsupported(module.file.Path, attribute.Span, "foreign function attribute @"+attribute.Name)
			continue
		}
		if seenLinkName || len(attribute.Arguments) != 1 || len(attribute.Arguments[0].Raw) < 2 || attribute.Arguments[0].Raw[0] != '"' || attribute.Arguments[0].Cooked == "" {
			a.addDiagnostic("E5106", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
			seenLinkName = true
			continue
		}
		seenLinkName = true
		linkName = attribute.Arguments[0].Cooked
	}
	result := hir.ForeignFunction{
		Symbol: symbol.id, Name: function.Name, Public: function.Public, LinkName: linkName, ABI: function.ABI,
		Span:              span(module.file.Path, function.Span),
		DocumentationSpan: optionalSpan(module.file.Path, function.DocumentationSpan),
		Effects:           slices.Clone(symbol.effects),
		ResultType:        a.canonicalType(module, function.ResultType),
	}
	symbol.typ = result.ResultType
	result.MustUse = a.finalizeMustUse(module.file.Path, function.ResultType.Span, symbol, result.ResultType)
	if containsSafeBorrow(result.ResultType) {
		a.addDiagnostic("E5102", span(module.file.Path, function.ResultType.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
	}
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		parameterType := a.canonicalType(module, parameter.Type)
		if containsSafeBorrow(parameterType) {
			a.addDiagnostic("E5102", span(module.file.Path, parameter.Type.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
		}
		parameterSpan := span(module.file.Path, parameter.NameSpan)
		result.Parameters = append(result.Parameters, hir.ForeignParameter{Name: parameter.Name, Type: parameterType, Span: parameterSpan})
		if parameterType != hir.TypeInvalid {
			a.types = append(a.types, TypeFact{Expression: parameterSpan, CanonicalType: parameterType})
		}
	}
	if result.ResultType != hir.TypeInvalid {
		a.types = append(a.types, TypeFact{Expression: span(module.file.Path, function.ResultType.Span), CanonicalType: result.ResultType})
	}
	return result
}

func (a *analyzer) checkResource(module *moduleScope, resource *ast.Resource) hir.Resource {
	value := a.resources[resource]
	for _, attribute := range resource.Attributes {
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == effectsAttribute {
			continue
		}
		if attribute.Name == mustUseAttribute {
			a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, resource.NameSpan)})
			continue
		}
		a.addUnsupported(module.file.Path, attribute.Span, "resource attribute @"+attribute.Name)
	}
	if resource.Cleanup != nil {
		for _, attribute := range resource.Cleanup.Attributes {
			if attribute.Name == constEvalAttribute {
				continue
			}
			if attribute.Name == effectsAttribute {
				continue
			}
			if attribute.Name == mustUseAttribute {
				a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "cleanup-declaration", Span: span(module.file.Path, resource.Cleanup.Span)})
			}
		}
	}
	result := hir.Resource{
		Name:              resource.Name,
		Public:            resource.Public,
		Type:              value.typ,
		Span:              span(module.file.Path, resource.Span),
		DocumentationSpan: optionalSpan(module.file.Path, resource.DocumentationSpan),
		DropKind:          hir.ResourceDropMissing,
		DropSymbol:        hir.NoSymbol,
		DropEffects:       slices.Clone(value.effects),
	}
	if info := a.resourceTypes[value.typ]; info != nil {
		result.Abilities = info.abilities.canonicalHIR()
		result.Fields = make([]hir.ResourceField, 0, len(info.orderedFields))
		for _, field := range info.orderedFields {
			result.Fields = append(result.Fields, hir.ResourceField{Name: field.name, Type: field.typ, Span: field.span})
		}
	}
	if resource.Cleanup == nil || resource.Cleanup.Body == nil {
		a.addUnsupported(module.file.Path, resource.Span, "resource without a cleanup block")
		return result
	}
	result.DropSpan = span(module.file.Path, resource.Cleanup.Span)
	if len(resource.Cleanup.Body.Statements) == 0 && resource.Cleanup.Body.Tail == nil {
		if len(value.effects) > 0 && value.effectsValid {
			a.addDiagnostic("E7101", span(module.file.Path, value.effectSpan), RelatedSpan{
				Role: "cleanup-declaration", Span: span(module.file.Path, resource.Cleanup.Span),
			})
		}
		result.DropKind = hir.ResourceDropEmpty
		return result
	}
	result.DropKind = hir.ResourceDropNonEmpty
	if function := a.dropFunctions[resource]; function != nil {
		if dropSymbol := a.functions[function]; dropSymbol != nil {
			result.DropSymbol = dropSymbol.id
		}
	}
	return result
}
