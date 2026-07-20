package semantic

import (
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

type resourceInfo struct {
	symbol        *symbol
	fields        map[string]resourceField
	orderedFields []resourceField
}

type structInfo struct {
	symbol        *symbol
	fields        map[string]resourceField
	orderedFields []resourceField
	reprC         bool
}

type enumInfo struct {
	symbol          *symbol
	variants        map[string]enumVariant
	orderedVariants []enumVariant
}

type enumVariant struct {
	name          string
	span          hir.Span
	declaration   hir.Span
	fields        map[string]resourceField
	orderedFields []resourceField
}

type resourceField struct {
	name        string
	typ         hir.Type
	span        hir.Span
	declaration hir.Span
	typeSpan    hir.Span
}

func (a *analyzer) prepareAggregateIdentities() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Enums {
			enumeration := &module.file.Module.Enums[index]
			if value := a.enums[enumeration]; value != nil {
				value.typ = hir.Type(module.file.Module.Path + "::" + enumeration.Name)
				a.enumTypes[value.typ] = &enumInfo{symbol: value, variants: make(map[string]enumVariant)}
			}
		}
		for index := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[index]
			if value := a.structs[structure]; value != nil {
				value.typ = hir.Type(module.file.Module.Path + "::" + structure.Name)
				a.structTypes[value.typ] = &structInfo{symbol: value, fields: make(map[string]resourceField)}
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

func (a *analyzer) prepareEnumTypes() {
	for _, module := range a.moduleOrder {
		for enumIndex := range module.file.Module.Enums {
			enumeration := &module.file.Module.Enums[enumIndex]
			value := a.enums[enumeration]
			if value == nil {
				continue
			}
			info := a.enumTypes[value.typ]
			for variantIndex := range enumeration.Variants {
				variantDeclaration := &enumeration.Variants[variantIndex]
				if previous, duplicate := info.variants[variantDeclaration.Name]; duplicate {
					a.addDiagnostic("E0102", span(module.file.Path, variantDeclaration.NameSpan), RelatedSpan{
						Role: "previous-declaration", Span: previous.declaration,
					})
					continue
				}
				variant := enumVariant{
					name: variantDeclaration.Name, span: span(module.file.Path, variantDeclaration.Span),
					declaration: span(module.file.Path, variantDeclaration.NameSpan), fields: make(map[string]resourceField),
				}
				for fieldIndex := range variantDeclaration.Fields {
					field := &variantDeclaration.Fields[fieldIndex]
					if previous, duplicate := variant.fields[field.Name]; duplicate {
						a.addDiagnostic("E0102", span(module.file.Path, field.NameSpan), RelatedSpan{
							Role: "previous-declaration", Span: previous.declaration,
						})
						continue
					}
					fieldType := a.canonicalType(module, field.Type)
					if _, _, array := hir.FixedArrayParts(fieldType); array {
						a.addUnsupported(module.file.Path, field.Type.Span, "fixed-array enum payload")
					}
					resolved := resourceField{
						name: field.Name, typ: fieldType, span: span(module.file.Path, field.Span),
						declaration: span(module.file.Path, field.NameSpan), typeSpan: span(module.file.Path, field.Type.Span),
					}
					variant.fields[field.Name] = resolved
					variant.orderedFields = append(variant.orderedFields, resolved)
				}
				info.variants[variant.name] = variant
				info.orderedVariants = append(info.orderedVariants, variant)
			}
		}
	}

	for _, info := range a.enumTypes {
		for _, variant := range info.orderedVariants {
			for _, field := range variant.orderedFields {
				if field.typ == hir.TypeInvalid {
					continue
				}
				if field.typ == hir.TypeRegion {
					a.addDiagnostic("E3103", field.typeSpan, RelatedSpan{Role: "capability", Span: field.declaration})
					continue
				}
				if _, _, array := hir.FixedArrayParts(field.typ); array {
					continue
				}
				if !a.inertCopyable(field.typ, make(map[hir.Type]uint8)) {
					a.addDiagnostic("E1004", field.typeSpan, RelatedSpan{Role: "variant", Span: variant.declaration})
				}
			}
		}
	}
}

func (a *analyzer) inertCopyable(value hir.Type, state map[hir.Type]uint8) bool {
	if hir.IsCScalar(value) {
		return true
	}
	switch value {
	case hir.TypeBool, hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return true
	case hir.TypeInvalid, hir.TypeVoid, hir.TypeRegion:
		return false
	}
	if _, _, pointer := hir.RawPointerElement(value); pointer {
		return true
	}
	if _, _, borrow := hir.BorrowElement(value); borrow {
		return false
	}
	if a.isResource(value) {
		return false
	}
	if element, option := hir.OptionElement(value); option {
		return a.inertCopyable(element, state)
	}
	if success, failure, result := hir.ResultElements(value); result {
		return a.inertCopyable(success, state) && a.inertCopyable(failure, state)
	}
	if state[value] == 1 {
		return false
	}
	if state[value] == 2 {
		return true
	}
	state[value] = 1
	defer func() { state[value] = 2 }()
	if info := a.structTypes[value]; info != nil {
		for _, field := range info.orderedFields {
			if !a.inertCopyable(field.typ, state) {
				return false
			}
		}
		return true
	}
	if info := a.enumTypes[value]; info != nil {
		for _, variant := range info.orderedVariants {
			for _, field := range variant.orderedFields {
				if !a.inertCopyable(field.typ, state) {
					return false
				}
			}
		}
		return true
	}
	return false
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
				if _, _, array := hir.FixedArrayParts(fieldType); array {
					a.addUnsupported(module.file.Path, field.Type.Span, "fixed-array resource field")
				}
				if fieldType == hir.TypeRegion {
					a.addDiagnostic("E3103", span(module.file.Path, field.Type.Span), RelatedSpan{
						Role: "capability",
						Span: span(module.file.Path, field.NameSpan),
					})
				}
				if _, _, borrow := hir.BorrowElement(fieldType); borrow {
					a.addDiagnostic("E1003", span(module.file.Path, field.Type.Span))
				}
				resolved := resourceField{
					name:        field.Name,
					typ:         fieldType,
					span:        span(module.file.Path, field.Span),
					declaration: span(module.file.Path, field.NameSpan),
				}
				info.fields[field.Name] = resolved
				info.orderedFields = append(info.orderedFields, resolved)
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
			info := a.structTypes[value.typ]
			for _, attribute := range structure.Attributes {
				if attribute.Name == "repr" && len(attribute.Arguments) == 1 && attribute.Arguments[0].Raw == "C" && !info.reprC {
					info.reprC = true
					continue
				}
				a.addUnsupported(module.file.Path, attribute.Span, "struct attribute @"+attribute.Name)
			}
			if info.reprC && len(structure.Fields) == 0 {
				a.addUnsupported(module.file.Path, structure.Span, "empty @repr(C) struct")
			}
			for fieldIndex := range structure.Fields {
				field := &structure.Fields[fieldIndex]
				if previous, duplicate := info.fields[field.Name]; duplicate {
					a.addDiagnostic("E0102", span(module.file.Path, field.NameSpan), RelatedSpan{Role: "previous-declaration", Span: previous.declaration})
					continue
				}
				fieldType := a.canonicalType(module, field.Type)
				if _, _, array := hir.FixedArrayParts(fieldType); array {
					a.addUnsupported(module.file.Path, field.Type.Span, "fixed-array struct field")
				}
				if fieldType == hir.TypeRegion {
					a.addDiagnostic("E3103", span(module.file.Path, field.Type.Span), RelatedSpan{
						Role: "capability",
						Span: span(module.file.Path, field.NameSpan),
					})
				}
				if _, _, borrow := hir.BorrowElement(fieldType); borrow {
					a.addDiagnostic("E1003", span(module.file.Path, field.Type.Span))
				}
				if a.isResource(fieldType) {
					a.addDiagnostic("E1002", span(module.file.Path, field.Type.Span))
				}
				resolved := resourceField{
					name: field.Name, typ: fieldType,
					span: span(module.file.Path, field.Span), declaration: span(module.file.Path, field.NameSpan),
				}
				info.fields[field.Name] = resolved
				info.orderedFields = append(info.orderedFields, resolved)
			}
		}
	}
}

func (a *analyzer) structType(value hir.Type) (*structInfo, bool) {
	info, found := a.structTypes[value]
	return info, found
}

func (a *analyzer) canonicalType(module *moduleScope, reference ast.TypeReference) hir.Type {
	value, ok := a.canonicalTypeText(module, reference.Text)
	if ok && validUnsizedTypeUse(value) {
		return value
	}
	a.addUnsupported(module.file.Path, reference.Span, "type "+strings.TrimSpace(reference.Text))
	return hir.TypeInvalid
}

func (a *analyzer) canonicalTypeText(module *moduleScope, text string) (hir.Type, bool) {
	return a.canonicalTypeTextMode(module, text, true)
}

func (a *analyzer) canonicalTypeTextMode(module *moduleScope, text string, allowFixedArray bool) (hir.Type, bool) {
	trimmed := strings.TrimSpace(text)
	if allowFixedArray {
		if elementText, length, ok := parseFixedArrayTypeText(trimmed); ok {
			element, elementOK := a.canonicalTypeTextMode(module, elementText, false)
			if !elementOK || !hir.ExecutableFixedArrayShape(element, length) {
				return hir.TypeInvalid, false
			}
			return hir.FixedArray(element, length), true
		}
	}
	if option, ok := parseOptionType(trimmed); ok {
		element, elementOK := a.canonicalTypeTextMode(module, option, false)
		if !elementOK {
			return hir.TypeInvalid, false
		}
		return hir.Option(element), true
	}
	if result, ok := parseResultType(trimmed); ok {
		success, successOK := a.canonicalTypeTextMode(module, result.success, false)
		failure, failureOK := a.canonicalTypeTextMode(module, result.failure, false)
		if !successOK || !failureOK {
			return hir.TypeInvalid, false
		}
		return hir.Result(success, failure), true
	}
	switch trimmed {
	case "bool":
		return hir.TypeBool, true
	case "u8":
		return hir.TypeU8, true
	case "i32":
		return hir.TypeI32, true
	case "u32":
		return hir.TypeU32, true
	case "usize":
		return hir.TypeUSize, true
	case "[u8]":
		return hir.TypeByteSlice, true
	case "str":
		return hir.TypeStr, true
	case "CStr":
		return hir.TypeCStr, true
	case "Region":
		return hir.TypeRegion, true
	}
	candidate := hir.Type(trimmed)
	if hir.IsCScalar(candidate) {
		return candidate, true
	}
	if strings.HasPrefix(trimmed, "*") {
		remainder := strings.TrimSpace(trimmed[1:])
		mutable := strings.HasPrefix(remainder, "mut") &&
			(len(remainder) == len("mut") || remainder[len("mut")] == ' ' || remainder[len("mut")] == '&' || remainder[len("mut")] == '*')
		if mutable {
			remainder = strings.TrimSpace(remainder[len("mut"):])
		}
		element, ok := hir.TypeInvalid, false
		if remainder == "void" {
			element, ok = hir.TypeVoid, true
		} else {
			element, ok = a.canonicalTypeTextMode(module, remainder, false)
		}
		if !ok {
			return hir.TypeInvalid, false
		}
		return hir.RawPointer(element, mutable), true
	}
	if strings.HasPrefix(trimmed, "&") {
		remainder := strings.TrimSpace(trimmed[1:])
		mutable := strings.HasPrefix(remainder, "mut") &&
			(len(remainder) == len("mut") || remainder[len("mut")] == ' ' || remainder[len("mut")] == '&')
		if mutable {
			remainder = strings.TrimSpace(remainder[len("mut"):])
		}
		// A safe borrow may name one executable fixed array. The array parser still
		// rejects nested fixed arrays because it canonicalizes its element with
		// allowFixedArray=false.
		element, ok := a.canonicalTypeTextMode(module, remainder, true)
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

func parseFixedArrayTypeText(text string) (string, uint64, bool) {
	if len(text) < 5 || text[0] != '[' || text[len(text)-1] != ']' {
		return "", 0, false
	}
	inside := text[1 : len(text)-1]
	depth := 0
	separator := -1
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '[', '<':
			depth++
		case ']', '>':
			depth--
			if depth < 0 {
				return "", 0, false
			}
		case ';':
			if depth == 0 {
				if separator >= 0 {
					return "", 0, false
				}
				separator = index
			}
		}
	}
	if depth != 0 || separator <= 0 || separator == len(inside)-1 {
		return "", 0, false
	}
	element := strings.TrimSpace(inside[:separator])
	length, ok := parsePositiveArrayLength(strings.TrimSpace(inside[separator+1:]))
	if element == "" || !ok {
		return "", 0, false
	}
	return element, length, true
}

func parsePositiveArrayLength(raw string) (uint64, bool) {
	clean := strings.ReplaceAll(raw, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	if digits == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(digits, base, 64)
	return value, err == nil && value > 0
}

func validUnsizedTypeUse(value hir.Type) bool {
	if value == hir.TypeByteSlice || value == hir.TypeStr || value == hir.TypeCStr {
		return false
	}
	if element, mutable, borrow := hir.BorrowElement(value); borrow {
		if element == hir.TypeByteSlice {
			return true
		}
		if element == hir.TypeStr || element == hir.TypeCStr {
			return !mutable
		}
		return validUnsizedTypeUse(element)
	}
	if element, option := hir.OptionElement(value); option {
		return validUnsizedTypeUse(element)
	}
	if success, failure, result := hir.ResultElements(value); result {
		return validUnsizedTypeUse(success) && validUnsizedTypeUse(failure)
	}
	if element, _, pointer := hir.RawPointerElement(value); pointer {
		return validUnsizedTypeUse(element)
	}
	return true
}

func (a *analyzer) resourceType(value hir.Type) (*resourceInfo, bool) {
	info, found := a.resourceTypes[value]
	return info, found
}
