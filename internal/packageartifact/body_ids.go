package packageartifact

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// bodyGraphShape records the exact dense namespaces carried by a normalized function body.
// Function symbols and call targets are deliberately outside both function-local namespaces.
type bodyGraphShape struct {
	LocalCount int
	BlockCount int
}

// canonicalizeFunctionGraphIDs removes build-local local and block numbering while preserving
// declaration order, operation order, CFG edges, and the entity strings that encode region locals.
func canonicalizeFunctionGraphIDs(function mir.Function) (mir.Function, bodyGraphShape, error) {
	result := cloneFunction(function)
	locals := make(map[mir.LocalID]mir.LocalID)
	localOrder := make([]mir.LocalID, 0)
	addLocal := func(value mir.LocalID) error {
		if value == mir.NoLocal {
			return nil
		}
		if value < 0 {
			return fmt.Errorf("function %q contains negative local %d", function.Name, value)
		}
		if _, found := locals[value]; found {
			return nil
		}
		locals[value] = mir.LocalID(len(localOrder))
		localOrder = append(localOrder, value)
		return nil
	}
	addSymbol := func(value hir.SymbolID) error {
		if value == hir.NoSymbol {
			return nil
		}
		return addLocal(mir.LocalID(value))
	}
	addOperand := func(value mir.Operand) error {
		if value.Kind != mir.OperandLocal && value.Kind != mir.OperandRegion {
			return nil
		}
		return addLocal(value.Local)
	}

	for _, parameter := range result.Parameters {
		if err := addLocal(parameter); err != nil {
			return mir.Function{}, bodyGraphShape{}, err
		}
	}
	for _, resource := range result.Resources {
		if err := addSymbol(resource.Symbol); err != nil {
			return mir.Function{}, bodyGraphShape{}, err
		}
	}
	for _, flag := range result.DropFlags {
		if err := addSymbol(flag.Symbol); err != nil {
			return mir.Function{}, bodyGraphShape{}, err
		}
	}
	for _, block := range result.Blocks {
		for _, operation := range block.Operations {
			if operationCarriesLocalSymbol(operation.Kind) {
				if err := addSymbol(operation.Symbol); err != nil {
					return mir.Function{}, bodyGraphShape{}, err
				}
			}
			for _, operand := range operation.Operands {
				if err := addOperand(operand); err != nil {
					return mir.Function{}, bodyGraphShape{}, err
				}
			}
			if err := addLocal(operation.DestinationLocal); err != nil {
				return mir.Function{}, bodyGraphShape{}, err
			}
		}
		terminator := block.Terminator
		if err := addOperand(terminator.ReturnValue); err != nil {
			return mir.Function{}, bodyGraphShape{}, err
		}
		for _, operand := range terminator.Arguments {
			if err := addOperand(operand); err != nil {
				return mir.Function{}, bodyGraphShape{}, err
			}
		}
		for _, operand := range terminator.SwitchOperands {
			if err := addOperand(operand); err != nil {
				return mir.Function{}, bodyGraphShape{}, err
			}
		}
		if err := addLocal(terminator.DestinationLocal); err != nil {
			return mir.Function{}, bodyGraphShape{}, err
		}
	}

	blocks := make(map[mir.BlockID]mir.BlockID, len(result.Blocks))
	for index, block := range result.Blocks {
		if block.ID < 0 {
			return mir.Function{}, bodyGraphShape{}, fmt.Errorf("function %q contains negative block %d", function.Name, block.ID)
		}
		if _, duplicate := blocks[block.ID]; duplicate {
			return mir.Function{}, bodyGraphShape{}, fmt.Errorf("function %q contains duplicate block %d", function.Name, block.ID)
		}
		blocks[block.ID] = mir.BlockID(index)
	}
	if _, found := blocks[result.Entry]; !found {
		return mir.Function{}, bodyGraphShape{}, fmt.Errorf("function %q entry block %d is missing", function.Name, result.Entry)
	}

	shape := bodyGraphShape{LocalCount: len(localOrder), BlockCount: len(result.Blocks)}
	if err := rewriteFunctionGraphIDs(&result, locals, blocks); err != nil {
		return mir.Function{}, bodyGraphShape{}, err
	}
	return result, shape, nil
}

// rebindFunctionGraphIDs moves one canonical dense function graph into a fresh local range. Block
// IDs remain dense zero-based indexes because the verified MIR model uses them to index Blocks.
func rebindFunctionGraphIDs(function mir.Function, shape bodyGraphShape, firstLocal mir.LocalID) (mir.Function, error) {
	if shape.LocalCount < 0 || shape.BlockCount <= 0 || firstLocal < 0 || len(function.Blocks) != shape.BlockCount {
		return mir.Function{}, fmt.Errorf("function graph rebind has invalid namespace bounds")
	}
	locals := make(map[mir.LocalID]mir.LocalID, shape.LocalCount)
	for index := 0; index < shape.LocalCount; index++ {
		old := mir.LocalID(index)
		fresh := firstLocal + old
		if fresh < firstLocal {
			return mir.Function{}, fmt.Errorf("function graph local namespace overflows")
		}
		locals[old] = fresh
	}
	blocks := make(map[mir.BlockID]mir.BlockID, shape.BlockCount)
	for index := 0; index < shape.BlockCount; index++ {
		id := mir.BlockID(index)
		if function.Blocks[index].ID != id {
			return mir.Function{}, fmt.Errorf("function graph blocks are not canonical dense indexes")
		}
		blocks[id] = id
	}
	result := cloneFunction(function)
	if err := rewriteFunctionGraphIDs(&result, locals, blocks); err != nil {
		return mir.Function{}, fmt.Errorf("rebind function graph: %w", err)
	}
	return result, nil
}

func rewriteFunctionGraphIDs(function *mir.Function, locals map[mir.LocalID]mir.LocalID, blocks map[mir.BlockID]mir.BlockID) error {
	local := func(value mir.LocalID) (mir.LocalID, error) {
		if value == mir.NoLocal {
			return value, nil
		}
		rewritten, found := locals[value]
		if !found {
			return mir.NoLocal, fmt.Errorf("local %d is outside the declared function namespace", value)
		}
		return rewritten, nil
	}
	symbol := func(value hir.SymbolID) (hir.SymbolID, error) {
		if value == hir.NoSymbol {
			return value, nil
		}
		rewritten, err := local(mir.LocalID(value))
		return hir.SymbolID(rewritten), err
	}
	operand := func(value *mir.Operand) error {
		if value.Kind != mir.OperandLocal && value.Kind != mir.OperandRegion {
			return nil
		}
		rewritten, err := local(value.Local)
		if err != nil {
			return err
		}
		value.Local = rewritten
		return nil
	}
	block := func(value mir.BlockID) (mir.BlockID, error) {
		if value == mir.NoBlock {
			return value, nil
		}
		rewritten, found := blocks[value]
		if !found {
			return mir.NoBlock, fmt.Errorf("block %d is outside the declared function namespace", value)
		}
		return rewritten, nil
	}

	var err error
	function.Entry, err = block(function.Entry)
	if err != nil {
		return err
	}
	for index := range function.Parameters {
		function.Parameters[index], err = local(function.Parameters[index])
		if err != nil {
			return err
		}
	}
	for index := range function.Resources {
		function.Resources[index].Symbol, err = symbol(function.Resources[index].Symbol)
		if err != nil {
			return err
		}
	}
	for index := range function.DropFlags {
		function.DropFlags[index].Symbol, err = symbol(function.DropFlags[index].Symbol)
		if err != nil {
			return err
		}
	}
	for index := range function.RegionValues {
		function.RegionValues[index].Entity, err = rewriteLocalEntity(function.RegionValues[index].Entity, locals)
		if err != nil {
			return err
		}
		function.RegionValues[index].RegionEntity, err = rewriteLocalEntity(function.RegionValues[index].RegionEntity, locals)
		if err != nil {
			return err
		}
	}
	for blockIndex := range function.Blocks {
		current := &function.Blocks[blockIndex]
		current.ID, err = block(current.ID)
		if err != nil {
			return err
		}
		for operationIndex := range current.Operations {
			operation := &current.Operations[operationIndex]
			if operationCarriesLocalSymbol(operation.Kind) {
				operation.Symbol, err = symbol(operation.Symbol)
				if err != nil {
					return err
				}
			}
			operation.DestinationLocal, err = local(operation.DestinationLocal)
			if err != nil {
				return err
			}
			for operandIndex := range operation.Operands {
				if err := operand(&operation.Operands[operandIndex]); err != nil {
					return err
				}
			}
		}
		terminator := &current.Terminator
		for successorIndex := range terminator.Successors {
			terminator.Successors[successorIndex], err = block(terminator.Successors[successorIndex])
			if err != nil {
				return err
			}
		}
		terminator.UnwindSuccessor, err = block(terminator.UnwindSuccessor)
		if err != nil {
			return err
		}
		terminator.DestinationLocal, err = local(terminator.DestinationLocal)
		if err != nil {
			return err
		}
		if err := operand(&terminator.ReturnValue); err != nil {
			return err
		}
		for operandIndex := range terminator.Arguments {
			if err := operand(&terminator.Arguments[operandIndex]); err != nil {
				return err
			}
		}
		for operandIndex := range terminator.SwitchOperands {
			if err := operand(&terminator.SwitchOperands[operandIndex]); err != nil {
				return err
			}
		}
	}
	return nil
}

func operationCarriesLocalSymbol(kind mir.OperationKind) bool {
	switch kind {
	case mir.OperationInitialize, mir.OperationMove, mir.OperationDrop, mir.OperationBurn,
		mir.OperationRegionEnter, mir.OperationRegionPlace, mir.OperationRegionReset:
		return true
	default:
		return false
	}
}

func rewriteLocalEntity(value string, locals map[mir.LocalID]mir.LocalID) (string, error) {
	for _, prefix := range []string{"region:", "local:region-value:"} {
		if !strings.HasPrefix(value, prefix) {
			continue
		}
		parsed, err := strconv.ParseInt(strings.TrimPrefix(value, prefix), 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid local entity %q", value)
		}
		rewritten, found := locals[mir.LocalID(parsed)]
		if !found {
			return "", fmt.Errorf("local entity %q is outside the declared function namespace", value)
		}
		return prefix + strconv.FormatInt(int64(rewritten), 10), nil
	}
	return value, nil
}
