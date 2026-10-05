package mir

import "github.com/0disoft/fairyc/internal/hir"

func (p typeProperties) plainViewStruct(element hir.Type) bool {
	return hir.PlainViewStruct(element, p.plainStructFields)
}

func (p typeProperties) inlineArrayField(value hir.Type) bool {
	return hir.InlineArrayField(value, p.plainStructFields)
}

func (p typeProperties) plainStructFields(value hir.Type) ([]hir.Type, bool) {
	info := p.structs[value]
	if info == nil {
		return nil, false
	}
	fields := make([]hir.Type, len(info.Fields))
	for i, field := range info.Fields {
		fields[i] = field.Type
	}
	return fields, true
}

func (p typeProperties) validStructViewType(value hir.Type) bool {
	if element, length, array := hir.FixedArrayParts(value); array {
		return hir.FixedArrayScalarElement(element) || hir.ExecutableFixedArrayShape(element, length) && p.plainViewStruct(element)
	}
	if slice, _, borrowed := hir.BorrowElement(value); borrowed {
		if element, nominal := hir.StructSliceElement(slice); nominal {
			return p.plainViewStruct(element)
		}
		return p.validStructViewType(slice)
	}
	if element, ok := hir.OptionElement(value); ok {
		return p.validStructViewType(element)
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		return p.validStructViewType(success) && p.validStructViewType(failure)
	}
	if element, _, ok := hir.RawPointerElement(value); ok {
		return p.validStructViewType(element)
	}
	_, slice := hir.StructSliceElement(value)
	return !slice
}

func verifyStructViewTypes(function *Function, p typeProperties) []VerificationError {
	var failures []VerificationError
	check := func(value hir.Type) {
		if !p.validStructViewType(value) {
			failures = append(failures, VerificationError{Code: "V1001", Entity: function.Name, Detail: "struct view requires an actual nonempty plain scalar-field struct"})
		}
	}
	check(function.ResultType)
	for _, value := range function.ParameterTypes {
		check(value)
	}
	for _, block := range function.Blocks {
		for _, op := range block.Operations {
			check(op.Type)
			check(op.OwnerType)
			check(op.IndexType)
			for _, operand := range op.Operands {
				check(operand.Type)
			}
		}
	}
	return failures
}
