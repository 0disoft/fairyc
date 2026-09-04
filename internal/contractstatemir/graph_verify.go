package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

const (
	maxGraphBlocks     = 64
	maxGraphOperations = 1024
	canonicalStepCost  = 1
)

func verifyGraph(program Program, bound boundLayout) ([]VerificationError, []hir.Effect) {
	failures := make([]VerificationError, 0)
	requiredEffects := make([]hir.Effect, 0, 2)
	if len(program.Blocks) > maxGraphBlocks {
		return []VerificationError{{Code: "CSM1006", Entity: "graph", Detail: "block count exceeds 64"}}, nil
	}
	if int(program.Entry) >= len(program.Blocks) {
		return []VerificationError{{Code: "CSM1006", Entity: "entry", Detail: "entry block is outside the graph"}}, nil
	}
	valueTypes := make(map[ValueID]string)
	operationCount := 0
	for index, block := range program.Blocks {
		entity := fmt.Sprintf("block %d", index)
		if block.ID != BlockID(index) {
			failures = append(failures, VerificationError{Code: "CSM1006", Entity: entity, Detail: "block IDs must be dense and source ordered"})
		}
		if block.ID == program.Entry && len(block.Parameters) != 0 {
			failures = append(failures, VerificationError{Code: "CSM1007", Entity: entity, Detail: "entry block cannot declare join parameters"})
		}
		for _, parameter := range block.Parameters {
			if detail := defineGraphValue(valueTypes, parameter.ID, parameter.TypeID); detail != "" {
				failures = append(failures, VerificationError{Code: "CSM1007", Entity: entity, Detail: detail})
			}
		}
		operationCount += len(block.Operations)
		for opIndex, operation := range block.Operations {
			opEntity := fmt.Sprintf("block %d operation %d", index, opIndex)
			if operation.Result != 0 {
				typeID := graphOperationResultType(operation, bound)
				if detail := defineGraphValue(valueTypes, operation.Result, typeID); detail != "" {
					failures = append(failures, VerificationError{Code: "CSM1007", Entity: opEntity, Detail: detail})
				}
			}
			if operation.Found != 0 {
				if detail := defineGraphValue(valueTypes, operation.Found, contractstate.Bool.TypeID()); detail != "" {
					failures = append(failures, VerificationError{Code: "CSM1007", Entity: opEntity, Detail: detail})
				}
			}
		}
	}
	if operationCount > maxGraphOperations {
		failures = append(failures, VerificationError{Code: "CSM1006", Entity: "graph", Detail: "operation count exceeds 1024"})
	}

	edges := make([][]BlockID, len(program.Blocks))
	indegree := make([]int, len(program.Blocks))
	for index, block := range program.Blocks {
		available := make(map[ValueID]string)
		for _, parameter := range block.Parameters {
			available[parameter.ID] = parameter.TypeID
		}
		for opIndex, operation := range block.Operations {
			entity := fmt.Sprintf("block %d operation %d", index, opIndex)
			if operation.Capability.token == nil || operation.Capability != program.Capability {
				failures = append(failures, VerificationError{Code: "CSM1004", Entity: entity, Detail: "operation does not use the program transaction capability"})
			}
			if operation.Cost != canonicalStepCost {
				failures = append(failures, VerificationError{Code: "CSM1008", Entity: entity, Detail: "operation requires explicit canonical cost 1"})
			}
			requiredEffects = append(requiredEffects, requiredStorageEffects(operation.Kind)...)
			if int(operation.FieldIndex) >= len(bound.fields) {
				failures = append(failures, VerificationError{Code: "CSM1002", Entity: entity, Detail: "field index is outside the verified layout"})
				continue
			}
			field := bound.fields[operation.FieldIndex].descriptor
			if detail := verifyGraphOperation(operation, field, available); detail != "" {
				failures = append(failures, VerificationError{Code: "CSM1002", Entity: entity, Detail: detail})
			}
			if operation.Result != 0 {
				available[operation.Result] = field.ValueTypeID
			}
			if operation.Found != 0 {
				available[operation.Found] = contractstate.Bool.TypeID()
			}
		}
		termEntity := fmt.Sprintf("block %d terminator", index)
		if block.Terminator.Cost != canonicalStepCost {
			failures = append(failures, VerificationError{Code: "CSM1008", Entity: termEntity, Detail: "terminator requires explicit canonical cost 1"})
		}
		blockEdges, termFailures := verifyGraphTerminator(block.Terminator, program.Blocks, available, index)
		failures = append(failures, termFailures...)
		for _, target := range blockEdges {
			edges[index] = append(edges[index], target)
			if int(target) < len(indegree) {
				indegree[target]++
			}
		}
	}
	if indegree[program.Entry] != 0 {
		failures = append(failures, VerificationError{Code: "CSM1006", Entity: "entry", Detail: "entry block has a predecessor"})
	}
	seen := make([]bool, len(program.Blocks))
	active := make([]bool, len(program.Blocks))
	var visit func(BlockID)
	visit = func(id BlockID) {
		if int(id) >= len(program.Blocks) || active[id] {
			if int(id) < len(program.Blocks) && active[id] {
				failures = append(failures, VerificationError{Code: "CSM1006", Entity: fmt.Sprintf("block %d", id), Detail: "graph contains a cycle"})
			}
			return
		}
		if seen[id] {
			return
		}
		seen[id], active[id] = true, true
		for _, target := range edges[id] {
			visit(target)
		}
		active[id] = false
	}
	visit(program.Entry)
	for index, reachable := range seen {
		if !reachable {
			failures = append(failures, VerificationError{Code: "CSM1006", Entity: fmt.Sprintf("block %d", index), Detail: "block is unreachable from entry"})
		}
	}
	return failures, requiredEffects
}

func defineGraphValue(types map[ValueID]string, id ValueID, typeID string) string {
	if id == 0 || typeID == "" {
		return "value identity and type must be nonzero"
	}
	if _, found := types[id]; found {
		return fmt.Sprintf("value %d is defined more than once", id)
	}
	if !knownGraphType(typeID) {
		return fmt.Sprintf("value %d has unsupported type %q", id, typeID)
	}
	types[id] = typeID
	return ""
}

func knownGraphType(typeID string) bool {
	return typeID == contractstate.Bool.TypeID() || typeID == contractstate.U32.TypeID() || typeID == contractstate.U64.TypeID() ||
		typeID == contractstate.Bytes.TypeID() || typeID == contractstate.Text.TypeID() ||
		typeID == contractstate.Byte20.TypeID() || typeID == contractstate.Byte32.TypeID()
}

func graphOperationResultType(operation Operation, bound boundLayout) string {
	if int(operation.FieldIndex) >= len(bound.fields) {
		return ""
	}
	return bound.fields[operation.FieldIndex].descriptor.ValueTypeID
}

func verifyGraphOperation(operation Operation, field contractstate.Field, available map[ValueID]string) string {
	isRead := operation.Kind == OperationCellRead || operation.Kind == OperationMapRead
	if isRead != (operation.Result != 0 && operation.Found != 0) {
		return "reads must define both value and found results; non-reads define neither"
	}
	if operation.Result != 0 && operation.Result == operation.Found {
		return "read value and found results must differ"
	}
	keyType := ""
	if operation.KeyInput != 0 {
		keyType = available[operation.KeyInput]
		if keyType == "" {
			return fmt.Sprintf("key value %d is not available in this block", operation.KeyInput)
		}
		if operation.Key.TypeID != "" || operation.Key.Encoded != nil {
			return "dynamic key also carries a literal"
		}
	}
	valueType := ""
	if operation.ValueInput != 0 {
		valueType = available[operation.ValueInput]
		if valueType == "" {
			return fmt.Sprintf("write value %d is not available in this block", operation.ValueInput)
		}
		if operation.Value.TypeID != "" || operation.Value.Encoded != nil {
			return "dynamic write value also carries a literal"
		}
	}
	switch operation.Kind {
	case OperationCellRead, OperationCellDelete:
		if field.Kind != contractstate.FieldCell || operation.KeyInput != 0 || operation.Key.TypeID != "" || operation.Key.Encoded != nil || operation.ValueInput != 0 || operation.Value.TypeID != "" || operation.Value.Encoded != nil {
			return "cell read/delete has an invalid field or operand"
		}
	case OperationCellWrite:
		if field.Kind != contractstate.FieldCell || operation.KeyInput != 0 || operation.Key.TypeID != "" || operation.Key.Encoded != nil {
			return "cell write has an invalid field or key"
		}
		if valueType != "" {
			if valueType != field.ValueTypeID {
				return "dynamic value type does not match field type"
			}
		} else if detail := verifyValue("value", operation.Value, field.ValueTypeID); detail != "" {
			return detail
		}
	case OperationMapRead, OperationMapDelete:
		if field.Kind != contractstate.FieldMap || operation.ValueInput != 0 || operation.Value.TypeID != "" || operation.Value.Encoded != nil {
			return "map read/delete has an invalid field or value"
		}
		if keyType != "" {
			if keyType != field.KeyTypeID {
				return "dynamic key type does not match field type"
			}
		} else if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
	case OperationMapWrite:
		if field.Kind != contractstate.FieldMap {
			return "map write names a non-map field"
		}
		if keyType != "" {
			if keyType != field.KeyTypeID {
				return "dynamic key type does not match field type"
			}
		} else if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
		if valueType != "" {
			if valueType != field.ValueTypeID {
				return "dynamic value type does not match field type"
			}
		} else if detail := verifyValue("value", operation.Value, field.ValueTypeID); detail != "" {
			return detail
		}
	default:
		return fmt.Sprintf("operation kind %q is unsupported", operation.Kind)
	}
	return ""
}

func verifyGraphTerminator(term Terminator, blocks []Block, available map[ValueID]string, source int) ([]BlockID, []VerificationError) {
	entity := fmt.Sprintf("block %d terminator", source)
	fail := func(detail string) ([]BlockID, []VerificationError) {
		return nil, []VerificationError{{Code: "CSM1006", Entity: entity, Detail: detail}}
	}
	verifyEdge := func(target BlockID, arguments []ValueID) string {
		if int(target) >= len(blocks) {
			return "target block is outside the graph"
		}
		parameters := blocks[target].Parameters
		if len(arguments) != len(parameters) {
			return "edge argument count does not match target parameters"
		}
		for index, argument := range arguments {
			typeID := available[argument]
			if typeID == "" {
				return fmt.Sprintf("edge value %d is not available in this block", argument)
			}
			if typeID != parameters[index].TypeID {
				return "edge argument type does not match target parameter"
			}
		}
		return ""
	}
	switch term.Kind {
	case TerminatorJump:
		if term.Condition != 0 || len(term.TrueArguments) != 0 || len(term.FalseArguments) != 0 || term.TrapCategory != "" {
			return fail("jump carries unrelated fields")
		}
		if detail := verifyEdge(term.Target, term.Arguments); detail != "" {
			return fail(detail)
		}
		return []BlockID{term.Target}, nil
	case TerminatorBranch:
		if available[term.Condition] != contractstate.Bool.TypeID() || len(term.Arguments) != 0 || term.TrapCategory != "" {
			return fail("branch requires one available bool condition and no unrelated fields")
		}
		if detail := verifyEdge(term.TrueTarget, term.TrueArguments); detail != "" {
			return fail("true edge: " + detail)
		}
		if detail := verifyEdge(term.FalseTarget, term.FalseArguments); detail != "" {
			return fail("false edge: " + detail)
		}
		return []BlockID{term.TrueTarget, term.FalseTarget}, nil
	case TerminatorCommit, TerminatorReject:
		if term.Condition != 0 || term.Target != 0 || len(term.Arguments) != 0 || term.TrueTarget != 0 || term.FalseTarget != 0 || len(term.TrueArguments) != 0 || len(term.FalseArguments) != 0 || term.TrapCategory != "" {
			return fail("exit carries unrelated fields")
		}
		return nil, nil
	case TerminatorTrap:
		if !validTrapCategory(term.TrapCategory) {
			return fail("trap requires a canonical category")
		}
		if term.Condition != 0 || term.Target != 0 || len(term.Arguments) != 0 || term.TrueTarget != 0 || term.FalseTarget != 0 || len(term.TrueArguments) != 0 || len(term.FalseArguments) != 0 {
			return fail("trap carries unrelated control fields")
		}
		return nil, nil
	default:
		return fail(fmt.Sprintf("terminator kind %q is unsupported", term.Kind))
	}
}
