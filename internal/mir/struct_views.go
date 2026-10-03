package mir

import "github.com/0disoft/fairyc/internal/hir"

func (p typeProperties) plainViewStruct(element hir.Type) bool {
	info := p.structs[element]
	if info == nil || len(info.Fields) == 0 {
		return false
	}
	for _, field := range info.Fields {
		if !hir.PlainViewField(field.Type) {
			return false
		}
	}
	return true
}

func (p typeProperties) validStructViewType(value hir.Type) bool {
	if slice, mutable, borrowed := hir.BorrowElement(value); borrowed {
		if element, nominal := hir.StructSliceElement(slice); nominal {
			return !mutable && p.plainViewStruct(element)
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
			failures = append(failures, VerificationError{Code: "V1001", Entity: function.Name, Detail: "struct view requires an actual nonempty shared scalar-field struct"})
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
