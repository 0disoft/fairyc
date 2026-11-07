package mir

import "github.com/0disoft/fairyc/internal/hir"

// HasCopiedClosureValues selects explicit private format gates. It examines
// structured types, including dormant declarations; no source text is reparsed.
func HasCopiedClosureValues(program Program) bool {
	seen := make(map[hir.Type]bool)
	var contains func(hir.Type) bool
	contains = func(typ hir.Type) bool {
		if typ == hir.TypeClosureI32 {
			return true
		}
		if typ == (hir.Type{}) || seen[typ] {
			return false
		}
		seen[typ] = true
		d := typ.Descriptor()
		return contains(d.Element) || contains(d.Failure)
	}
	for _, record := range program.StructTypes {
		if contains(record.Type) {
			return true
		}
		for _, field := range record.Fields {
			if contains(field.Type) {
				return true
			}
		}
	}
	for _, record := range program.ResourceTypes {
		if contains(record.Type) {
			return true
		}
		for _, field := range record.Fields {
			if contains(field.Type) {
				return true
			}
		}
	}
	for _, record := range program.EnumTypes {
		if contains(record.Type) {
			return true
		}
		for _, variant := range record.Variants {
			for _, field := range variant.Fields {
				if contains(field.Type) {
					return true
				}
			}
		}
	}
	for _, fn := range program.ForeignFunctions {
		if contains(fn.ResultType) {
			return true
		}
		for _, typ := range fn.ParameterTypes {
			if contains(typ) {
				return true
			}
		}
	}
	for _, fn := range program.Functions {
		if fn.ClosureTarget || fn.ClosureCaptures != 0 || contains(fn.ResultType) || contains(fn.DropGlueFor) {
			return true
		}
		for _, typ := range fn.ParameterTypes {
			if contains(typ) {
				return true
			}
		}
		for _, resource := range fn.Resources {
			if contains(resource.Type) {
				return true
			}
		}
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if op.Kind == OperationClosureCreate || contains(op.Type) || contains(op.OwnerType) || contains(op.IndexType) {
					return true
				}
				for _, value := range op.Operands {
					if contains(value.Type) {
						return true
					}
				}
			}
			term := block.Terminator
			if term.Kind == TerminatorClosureCall || contains(term.ReturnValue.Type) || contains(term.SwitchType) {
				return true
			}
			for _, list := range [][]Operand{term.Arguments, term.SwitchOperands} {
				for _, value := range list {
					if contains(value.Type) {
						return true
					}
				}
			}
		}
	}
	return false
}
