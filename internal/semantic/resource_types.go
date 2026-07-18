package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

type resourceInfo struct {
	symbol *symbol
	fields map[string]resourceField
}

type resourceField struct {
	name        string
	typ         hir.Type
	declaration hir.Span
}

func (a *analyzer) prepareAggregateIdentities() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Enums {
			enumeration := &module.file.Module.Enums[index]
			if value := a.enums[enumeration]; value != nil {
				value.typ = hir.Type(module.file.Module.Path + "::" + enumeration.Name)
			}
		}
		for index := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[index]
			if value := a.structs[structure]; value != nil {
				value.typ = hir.Type(module.file.Module.Path + "::" + structure.Name)
			}
		}
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			value := a.resources[resource]
			if value == nil {
				continue
			}
			value.typ = hir.Type(module.file.Module.Path + "::" + resource.Name)
			a.resourceTypes[value.typ] = &resourceInfo{symbol: value, fields: make(map[string]resourceField)}
		}
	}
}

func (a *analyzer) prepareResourceTypes() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			value := a.resources[resource]
			if value == nil {
				continue
			}
			info := a.resourceTypes[value.typ]
			for fieldIndex := range resource.Fields {
				field := &resource.Fields[fieldIndex]
				if previous, duplicate := info.fields[field.Name]; duplicate {
					a.addDiagnostic("E0102", span(module.file.Path, field.NameSpan), RelatedSpan{
						Role: "previous-declaration",
						Span: previous.declaration,
					})
					continue
				}
				fieldType := a.canonicalType(module, field.Type)
				if fieldType == hir.TypeRegion {
					a.addDiagnostic("E3103", span(module.file.Path, field.Type.Span), RelatedSpan{
						Role: "capability",
						Span: span(module.file.Path, field.NameSpan),
					})
				}
				if _, _, borrow := hir.BorrowElement(fieldType); borrow {
					a.addDiagnostic("E1003", span(module.file.Path, field.Type.Span))
				}
				info.fields[field.Name] = resourceField{
					name:        field.Name,
					typ:         fieldType,
					declaration: span(module.file.Path, field.NameSpan),
				}
			}
		}
	}
}

func (a *analyzer) prepareStructTypes() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[index]
			value := a.structs[structure]
			if value == nil {
				continue
			}
			seen := make(map[string]hir.Span, len(structure.Fields))
			for fieldIndex := range structure.Fields {
				field := &structure.Fields[fieldIndex]
				if previous, duplicate := seen[field.Name]; duplicate {
					a.addDiagnostic("E0102", span(module.file.Path, field.NameSpan), RelatedSpan{Role: "previous-declaration", Span: previous})
					continue
				}
				seen[field.Name] = span(module.file.Path, field.NameSpan)
				if a.canonicalType(module, field.Type) == hir.TypeRegion {
					a.addDiagnostic("E3103", span(module.file.Path, field.Type.Span), RelatedSpan{
						Role: "capability",
						Span: span(module.file.Path, field.NameSpan),
					})
				}
			}
		}
	}
}

func (a *analyzer) canonicalType(module *moduleScope, reference ast.TypeReference) hir.Type {
	value, ok := a.canonicalTypeText(module, reference.Text)
	if ok {
		return value
	}
	a.addUnsupported(module.file.Path, reference.Span, "type "+strings.TrimSpace(reference.Text))
	return hir.TypeInvalid
}

func (a *analyzer) canonicalTypeText(module *moduleScope, text string) (hir.Type, bool) {
	trimmed := strings.TrimSpace(text)
	if option, ok := parseOptionType(trimmed); ok {
		element, elementOK := a.canonicalTypeText(module, option)
		if !elementOK {
			return hir.TypeInvalid, false
		}
		return hir.Option(element), true
	}
	if result, ok := parseResultType(trimmed); ok {
		success, successOK := a.canonicalTypeText(module, result.success)
		failure, failureOK := a.canonicalTypeText(module, result.failure)
		if !successOK || !failureOK {
			return hir.TypeInvalid, false
		}
		return hir.Result(success, failure), true
	}
	switch trimmed {
	case "bool":
		return hir.TypeBool, true
	case "i32":
		return hir.TypeI32, true
	case "Region":
		return hir.TypeRegion, true
	}
	if strings.HasPrefix(trimmed, "&") {
		remainder := strings.TrimSpace(trimmed[1:])
		mutable := strings.HasPrefix(remainder, "mut") &&
			(len(remainder) == len("mut") || remainder[len("mut")] == ' ' || remainder[len("mut")] == '&')
		if mutable {
			remainder = strings.TrimSpace(remainder[len("mut"):])
		}
		element, ok := a.canonicalTypeText(module, remainder)
		if !ok {
			return hir.TypeInvalid, false
		}
		if mutable {
			return hir.MutableBorrow(element), true
		}
		return hir.SharedBorrow(element), true
	}
	parts := strings.Split(trimmed, "::")
	var target *symbol
	switch len(parts) {
	case 1:
		target = module.item(parts[0])
	case 2:
		if imported := module.imports[parts[0]]; imported != nil {
			target = imported.item(parts[1])
		}
	}
	if target == nil || (target.resource == nil && target.structure == nil && target.enumeration == nil) {
		return hir.TypeInvalid, false
	}
	return target.typ, true
}

func (a *analyzer) resourceType(value hir.Type) (*resourceInfo, bool) {
	info, found := a.resourceTypes[value]
	return info, found
}
