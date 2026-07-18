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

func (a *analyzer) prepareResourceTypes() {
	for _, module := range a.moduleOrder {
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
	switch trimmed {
	case "bool":
		return hir.TypeBool, true
	case "i32":
		return hir.TypeI32, true
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
		target = module.resources[parts[0]]
	case 2:
		if imported := module.imports[parts[0]]; imported != nil {
			target = imported.resources[parts[1]]
		}
	}
	if target == nil || target.resource == nil {
		return hir.TypeInvalid, false
	}
	return target.typ, true
}

func (a *analyzer) resourceType(value hir.Type) (*resourceInfo, bool) {
	info, found := a.resourceTypes[value]
	return info, found
}
