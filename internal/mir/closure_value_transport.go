package mir

import "github.com/0disoft/fairyc/internal/hir"

// Ordinary direct fields and monomorphic value transport are the complete
// admission boundary. A forged wrapper/ABI/operation cannot widen that boundary.
func verifyClosureValueTransport(program Program, functions map[hir.SymbolID]*Function, foreign map[hir.SymbolID]*ForeignFunction) []VerificationError {
	if !HasCopiedClosureValues(program) {
		return nil
	}
	carriers := map[hir.Type]bool{hir.TypeClosureI32: true}
	structs := make(map[hir.Type]*StructType, len(program.StructTypes))
	for i := range program.StructTypes {
		record := &program.StructTypes[i]
		structs[record.Type] = record
		for _, field := range record.Fields {
			if field.Type == hir.TypeClosureI32 {
				carriers[record.Type] = true
			}
		}
	}
	memo := make(map[hir.Type]bool)
	var contains func(hir.Type) bool
	contains = func(typ hir.Type) bool {
		if carriers[typ] {
			return true
		}
		if result, found := memo[typ]; found {
			return result
		}
		d := typ.Descriptor()
		result := false
		if d.Element != (hir.Type{}) {
			result = contains(d.Element)
		}
		if d.Failure != (hir.Type{}) {
			result = result || contains(d.Failure)
		}
		memo[typ] = result
		return result
	}
	reject := func(entity, detail string) []VerificationError {
		return []VerificationError{{Code: CodeOperationType, Entity: entity, Detail: "copied closure transport: " + detail}}
	}
	for _, record := range program.StructTypes {
		if contains(record.Type) && record.Type.Descriptor().Kind != hir.TypeKindNominal {
			return reject(record.Name, "aggregate identity cannot shadow a closure type")
		}
		for _, field := range record.Fields {
			if contains(field.Type) && (record.ReprC || field.Type != hir.TypeClosureI32) {
				return reject(record.Name, "only direct fields of ordinary structs are admitted")
			}
		}
	}
	for _, record := range program.ResourceTypes {
		if contains(record.Type) && record.Type.Descriptor().Kind != hir.TypeKindNominal {
			return reject(record.Type.String(), "resource identity cannot shadow a closure type")
		}
		for _, field := range record.Fields {
			if contains(field.Type) {
				return reject(record.Type.String(), "resource storage is closed")
			}
		}
	}
	for _, record := range program.EnumTypes {
		if contains(record.Type) && record.Type.Descriptor().Kind != hir.TypeKindNominal {
			return reject(record.Name, "enum identity cannot shadow a closure type")
		}
		for _, variant := range record.Variants {
			for _, field := range variant.Fields {
				if contains(field.Type) {
					return reject(record.Name, "enum storage is closed")
				}
			}
		}
	}
	for _, fn := range program.ForeignFunctions {
		if contains(fn.ResultType) {
			return reject(fn.Name, "foreign result is closed")
		}
		for _, typ := range fn.ParameterTypes {
			if contains(typ) {
				return reject(fn.Name, "foreign parameter is closed")
			}
		}
	}
	for _, fn := range program.Functions {
		badType := func(typ hir.Type) bool { return contains(typ) && !carriers[typ] }
		if badType(fn.ResultType) {
			return reject(fn.Name, "wrapped result is closed")
		}
		for _, typ := range fn.ParameterTypes {
			if badType(typ) {
				return reject(fn.Name, "wrapped parameter is closed")
			}
		}
		types, err := fieldStoreLocalTypesMode(&fn, functions, foreign, true)
		if err != nil {
			return reject(fn.Name, err.Error())
		}
		definitions := make(map[LocalID]bool)
		for _, local := range fn.Parameters {
			definitions[local] = true
		}
		define := func(local LocalID) bool {
			if !carriers[types[local]] {
				return true
			}
			if definitions[local] {
				return false
			}
			definitions[local] = true
			return true
		}
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if badType(op.Type) || badType(op.OwnerType) || contains(op.IndexType) {
					return reject(fn.Name, "wrapped operation type is closed")
				}
				if op.OwnerType == hir.TypeClosureI32 || carriers[op.OwnerType] && op.Kind != OperationFieldRead {
					return reject(fn.Name, "closure owner operation is closed")
				}
				if carriers[op.Type] {
					switch op.Kind {
					case OperationClosureCreate, OperationAssign, OperationStructConstruct, OperationFieldRead:
					default:
						return reject(fn.Name, "closure-producing operation is closed")
					}
				}
				for i, value := range op.Operands {
					actual := fieldStoreOperandType(value, types)
					if badType(actual) || badType(value.Type) {
						return reject(fn.Name, "wrapped operand is closed")
					}
					expected := hir.Type{}
					switch op.Kind {
					case OperationAssign:
						if len(op.Operands) == 1 {
							expected = types[op.DestinationLocal]
						}
					case OperationFieldRead:
						if i == 0 {
							expected = op.OwnerType
						}
					case OperationStructConstruct:
						if record := structs[op.Type]; record != nil && i < len(record.Fields) {
							expected = record.Fields[i].Type
						}
					}
					if (carriers[actual] || carriers[expected]) && (actual != expected || value.Kind != OperandLocal || value.Type != (hir.Type{}) && value.Type != expected) {
						return reject(fn.Name, "value is consumed by an unrelated operation or field")
					}
				}
				switch op.Kind {
				case OperationClosureCreate, OperationAssign, OperationFieldRead, OperationStructConstruct:
					if !define(op.DestinationLocal) {
						return reject(fn.Name, "immutable closure value is overwritten")
					}
				}
			}
			term := block.Terminator
			if badType(term.ReturnValue.Type) || contains(term.SwitchType) {
				return reject(fn.Name, "wrapped return or closure switch is closed")
			}
			for _, value := range term.SwitchOperands {
				if contains(fieldStoreOperandType(value, types)) || contains(value.Type) {
					return reject(fn.Name, "closure switch operand is closed")
				}
			}
			if term.Kind == TerminatorReturn {
				actual := fieldStoreOperandType(term.ReturnValue, types)
				if (carriers[actual] || carriers[fn.ResultType]) && actual != fn.ResultType {
					return reject(fn.Name, "return type differs from the declared closure carrier")
				}
			}
			if term.Kind == TerminatorCall {
				callee := functions[term.Callee]
				for i, value := range term.Arguments {
					actual := fieldStoreOperandType(value, types)
					expected := hir.Type{}
					if callee != nil && i < len(callee.ParameterTypes) && term.ForeignABI == "" {
						expected = callee.ParameterTypes[i]
					}
					if badType(actual) || badType(value.Type) || (carriers[actual] || carriers[expected]) && (actual != expected || value.Kind != OperandLocal || value.Type != (hir.Type{}) && value.Type != expected) {
						return reject(fn.Name, "call argument differs from its monomorphic closure parameter")
					}
				}
				if !define(term.DestinationLocal) {
					return reject(fn.Name, "call overwrites an immutable closure value")
				}
			}
		}
	}
	return nil
}
