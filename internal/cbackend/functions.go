package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
	"slices"
	"strconv"
	"strings"
)

func (e *emitter) emitFunction(output *strings.Builder, functionIndex int) error {
	function := &e.analysis.functions[functionIndex]
	resultType, err := e.analysis.emittedFunctionResultType(function)
	if err != nil {
		return fmt.Errorf("function %s: %w", function.Name, err)
	}
	output.WriteString("static ")
	if e.analysis.nonReturning[function.Symbol] {
		output.WriteString("_Noreturn ")
	}
	fmt.Fprintf(output, "%s fy_fn_%d(", resultType, functionIndex)
	if err := e.emitParameters(output, function.Parameters, function.ParameterTypes); err != nil {
		return fmt.Errorf("function %s: %w", function.Name, err)
	}
	output.WriteString(") {\n")
	parameters := make(map[mir.LocalID]struct{}, len(function.Parameters))
	for _, parameter := range function.Parameters {
		parameters[parameter] = struct{}{}
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", parameter)
	}
	for _, local := range collectLocals(*function) {
		if _, parameter := parameters[local]; parameter {
			continue
		}
		localType, found := e.analysis.localTypes[functionIndex][local]
		if !found {
			return fmt.Errorf("function %s local %d has no C type", function.Name, local)
		}
		cType, err := e.analysis.cValueType(localType)
		if err != nil {
			return fmt.Errorf("function %s local %d: %w", function.Name, local, err)
		}
		fmt.Fprintf(output, "    %s fy_l_%d;\n", cType, local)
		fmt.Fprintf(output, "    (void)&fy_l_%d;\n", local)
	}
	for _, symbol := range functionRegionSymbols(*function) {
		fmt.Fprintf(output, "    fy_region fy_region_%d = {0};\n", symbol)
		fmt.Fprintf(output, "    (void)&fy_region_%d;\n", symbol)
	}
	for _, divisorType := range functionCheckedDivisorTypes(*function) {
		cType, err := e.analysis.cValueType(divisorType)
		if err != nil {
			return fmt.Errorf("function %s checked divisor type: %w", function.Name, err)
		}
		name, err := checkedDivisorName(divisorType)
		if err != nil {
			return fmt.Errorf("function %s checked divisor type: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    volatile %s %s;\n", cType, name)
	}
	dynamicTargets := e.analysis.dynamicTrapTargets[functionIndex]
	if len(dynamicTargets) != 0 {
		output.WriteString("    uint32_t fy_pending_trap = FY_TRAP_UNREACHABLE;\n")
		output.WriteString("    struct fy_runtime_location_v0 fy_pending_trap_location = {0};\n")
	}
	fmt.Fprintf(output, "    goto fy_b_%d;\n", function.Entry)
	blocks := slices.Clone(function.Blocks)
	slices.SortFunc(blocks, func(left, right mir.Block) int { return int(left.ID - right.ID) })
	reachable := e.analysis.reachableBlocks[functionIndex]
	fixedTrapGroups := make(map[string][]mir.BlockID)
	for _, block := range blocks {
		// Collect fixed-category candidates; sharing also requires equal source spans.
		// Dynamic-category targets retain state set by their incoming edges.
		if !reachable[int(block.ID)] || len(block.Operations) != 0 ||
			block.Terminator.Kind != mir.TerminatorTrap || len(block.Terminator.TrapCategories) != 1 ||
			len(block.Terminator.Successors) != 0 || block.Terminator.UnwindSuccessor != mir.NoBlock {
			continue
		}
		if _, dynamic := dynamicTargets[block.ID]; dynamic {
			continue
		}
		category := block.Terminator.TrapCategories[0]
		fixedTrapGroups[category] = append(fixedTrapGroups[category], block.ID)
	}
	groupedTrapBlocks := make(map[mir.BlockID]struct{})
	var groupedCategories []string
	for category, ids := range fixedTrapGroups {
		if len(ids) < 2 {
			continue
		}
		// A shared category is no longer enough: the source payload is observable.
		sharedSpan := true
		for _, id := range ids[1:] {
			if function.Blocks[int(id)].Terminator.Span != function.Blocks[int(ids[0])].Terminator.Span {
				sharedSpan = false
				break
			}
		}
		if !sharedSpan {
			continue
		}
		groupedCategories = append(groupedCategories, category)
		for _, id := range ids {
			groupedTrapBlocks[id] = struct{}{}
		}
	}
	slices.Sort(groupedCategories)
	for blockIndex := range blocks {
		block := &blocks[blockIndex]
		if !reachable[int(block.ID)] {
			continue
		}
		if _, grouped := groupedTrapBlocks[block.ID]; grouped {
			continue
		}
		fmt.Fprintf(output, "fy_b_%d:\n", block.ID)
		for operationIndex := range block.Operations {
			if err := e.emitOperation(output, functionIndex, block.Operations[operationIndex]); err != nil {
				return err
			}
		}
		if err := e.emitTerminator(output, functionIndex, block.ID, block.Terminator); err != nil {
			return err
		}
	}
	for _, category := range groupedCategories {
		name, _, ok := cTrapSpec(category)
		if !ok {
			return CapabilityError{Function: function.Name, Feature: "trap category " + strconv.Quote(category)}
		}
		for _, id := range fixedTrapGroups[category] {
			fmt.Fprintf(output, "fy_b_%d:\n", id)
		}
		fmt.Fprintf(output, "    %s;\n", trapCall(name, function.Blocks[int(fixedTrapGroups[category][0])].Terminator.Span))
	}
	output.WriteString("}\n\n")
	return nil
}
