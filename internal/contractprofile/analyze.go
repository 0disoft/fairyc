package contractprofile

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

var deterministicProfileCScalars = hir.CScalarTypes()

type functionAnalysis struct {
	function            mir.Function
	callTargets         []int
	internalCallNames   []string
	cfgCycle            bool
	cycleEdges          []CycleEdge
	cfgMalformed        bool
	recursive           bool
	unknownInternalCall bool
	externalCall        bool
	localCost           uint64
	localCostOverflow   bool
	transitiveCost      uint64
	transitiveBounded   bool
	transitiveOverflow  bool
}

// Analyze applies the deterministic contract profile to an already verified MIR program.
// Callers handing unverified MIR to this function violate its input contract; checks that are cheap
// and security-relevant still fail closed rather than trusting malformed graph references.
func Analyze(program mir.Program, loopBudgets map[string]uint64) Report {
	report := NewReport()

	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	nodes := make([]functionAnalysis, len(functions))
	symbolToIndex := make(map[hir.SymbolID]int, len(functions))
	nameToIndex := make(map[string]int, len(functions))
	for index, function := range functions {
		nodes[index].function = function
		if previous, duplicate := symbolToIndex[function.Symbol]; duplicate {
			report.Violations = append(report.Violations, violationAt(
				"CP1012",
				function.Name,
				function.Span,
				fmt.Sprintf("verified MIR reuses function symbol %d already owned by %s", function.Symbol, nodes[previous].function.Name),
			))
		} else {
			symbolToIndex[function.Symbol] = index
		}
		if previous, duplicate := nameToIndex[function.Name]; duplicate {
			report.Violations = append(report.Violations, violationAt(
				"CP1012",
				function.Name,
				function.Span,
				"verified MIR function name is not unique; previous owner is "+nodes[previous].function.Name,
			))
		} else {
			nameToIndex[function.Name] = index
		}
	}

	for _, foreign := range program.ForeignFunctions {
		report.Violations = append(report.Violations, violationAt(
			"CP1001",
			foreign.Name,
			foreign.Span,
			"raw foreign declarations are unavailable in the deterministic contract profile",
		))
		if containsRawType(foreign.ResultType) || anyRawType(foreign.ParameterTypes) {
			report.Violations = append(report.Violations, violationAt(
				"CP1003",
				foreign.Name,
				foreign.Span,
				"foreign declaration exposes a raw-pointer representation",
			))
		}
		if containsFloatType(foreign.ResultType) || anyFloatType(foreign.ParameterTypes) {
			report.Violations = append(report.Violations, violationAt(
				"CP1013",
				foreign.Name,
				foreign.Span,
				"floating-point transport is unavailable until every contract backend shares exact semantics",
			))
		}
		foreignTypes := slices.Clone(foreign.ParameterTypes)
		foreignTypes = append(foreignTypes, foreign.ResultType)
		appendTargetTypeViolations(&report, foreign.Name, foreign.Span, "foreign signature", foreignTypes...)
	}

	for _, structure := range program.StructTypes {
		if structure.ReprC {
			report.Violations = append(report.Violations, violationAt(
				"CP1016",
				structure.Name,
				structure.Span,
				"@repr(C) aggregate layout is target-dependent and unavailable in the contract profile",
			))
		}
		for _, field := range structure.Fields {
			if containsRawType(field.Type) {
				report.Violations = append(report.Violations, violationAt(
					"CP1003",
					structure.Name,
					field.Span,
					"contract-visible aggregate field contains a raw-pointer type",
				))
			}
			if containsFloatType(field.Type) {
				report.Violations = append(report.Violations, violationAt(
					"CP1013",
					structure.Name,
					field.Span,
					"contract-visible aggregate field contains a floating-point type",
				))
			}
			appendTargetTypeViolations(&report, structure.Name, field.Span, "aggregate field", field.Type)
		}
	}
	for _, enumeration := range program.EnumTypes {
		for _, variant := range enumeration.Variants {
			for _, field := range variant.Fields {
				if containsRawType(field.Type) {
					report.Violations = append(report.Violations, violationAt(
						"CP1003",
						enumeration.Name,
						field.Span,
						"contract-visible enum payload contains a raw-pointer type",
					))
				}
				if containsFloatType(field.Type) {
					report.Violations = append(report.Violations, violationAt(
						"CP1013",
						enumeration.Name,
						field.Span,
						"contract-visible enum payload contains a floating-point type",
					))
				}
				appendTargetTypeViolations(&report, enumeration.Name, field.Span, "enum payload", field.Type)
			}
		}
	}
	for _, resource := range program.ResourceTypes {
		for _, field := range resource.Fields {
			if containsRawType(field.Type) {
				report.Violations = append(report.Violations, violationAt(
					"CP1003",
					resource.Type.String(),
					field.Span,
					"contract resource field contains a raw-pointer type",
				))
			}
			if containsFloatType(field.Type) {
				report.Violations = append(report.Violations, violationAt(
					"CP1013",
					resource.Type.String(),
					field.Span,
					"contract resource field contains a floating-point type",
				))
			}
			appendTargetTypeViolations(&report, resource.Type.String(), field.Span, "resource field", field.Type)
		}
		if len(resource.DropEffects) != 0 {
			report.Violations = append(report.Violations, violationAt(
				"CP1002",
				resource.Type.String(),
				resource.DropSpan,
				"resource cleanup requires external effects: "+joinEffects(resource.DropEffects),
			))
		}
	}

	for index := range nodes {
		node := &nodes[index]
		function := node.function
		if containsRawType(function.ResultType) || anyRawType(function.ParameterTypes) {
			report.Violations = append(report.Violations, violationAt(
				"CP1003",
				function.Name,
				function.Span,
				"function signature contains a raw-pointer type",
			))
		}
		if containsFloatType(function.ResultType) || anyFloatType(function.ParameterTypes) {
			report.Violations = append(report.Violations, violationAt(
				"CP1013",
				function.Name,
				function.Span,
				"function signature contains a floating-point type",
			))
		}
		functionTypes := slices.Clone(function.ParameterTypes)
		functionTypes = append(functionTypes, function.ResultType)
		appendTargetTypeViolations(&report, function.Name, function.Span, "function signature", functionTypes...)
		if len(function.Effects) != 0 {
			report.Violations = append(report.Violations, violationAt(
				"CP1002",
				function.Name,
				function.Span,
				"function requires external effects: "+joinEffects(function.Effects),
			))
		}
		if len(function.UnsafeOperations) != 0 {
			report.Violations = append(report.Violations, violationAt(
				"CP1004",
				function.Name,
				function.Span,
				"unsafe MIR operations are unavailable in the deterministic contract profile",
			))
		}

		node.cfgCycle, node.cfgMalformed, node.cycleEdges = controlFlowCycleInfo(function)
		if node.cfgMalformed {
			report.Violations = append(report.Violations, violationAt(
				"CP1011",
				function.Name,
				function.Span,
				"verified MIR control-flow graph has an invalid block identity or successor",
			))
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				weight, overflow := operationCost(operation)
				node.localCost, overflow = saturatingAdd(node.localCost, weight, overflow)
				node.localCostOverflow = node.localCostOverflow || overflow
				if rawPointerOperation(operation) || operationContainsRawType(operation) {
					report.Violations = append(report.Violations, violationAt(
						"CP1003",
						function.Name,
						operation.Span,
						"raw-pointer MIR operation is unavailable in the deterministic contract profile",
					))
				}
				if operationContainsFloatType(operation) {
					report.Violations = append(report.Violations, violationAt(
						"CP1013",
						function.Name,
						operation.Span,
						"floating-point MIR operation is unavailable until every backend shares exact semantics",
					))
				}
				appendTargetTypeViolations(&report, function.Name, operation.Span, "MIR operation", operationTypes(operation)...)
				if regionAllocationOperation(operation.Kind) {
					report.Violations = append(report.Violations, violationAt(
						"CP1005",
						function.Name,
						operation.Span,
						"Region allocation is not admitted until a backend meters memory growth explicitly",
					))
				}
			}

			weight, overflow := terminatorCost(block.Terminator)
			node.localCost, overflow = saturatingAdd(node.localCost, weight, overflow)
			node.localCostOverflow = node.localCostOverflow || overflow
			if terminatorContainsRawType(block.Terminator) {
				report.Violations = append(report.Violations, violationAt(
					"CP1003",
					function.Name,
					block.Terminator.Span,
					"raw-pointer call or return transport is unavailable in the deterministic contract profile",
				))
			}
			if terminatorContainsFloatType(block.Terminator) {
				report.Violations = append(report.Violations, violationAt(
					"CP1013",
					function.Name,
					block.Terminator.Span,
					"floating-point call or return transport is unavailable until every backend shares exact semantics",
				))
			}
			appendTargetTypeViolations(&report, function.Name, block.Terminator.Span, "MIR terminator", terminatorTypes(block.Terminator)...)
			if block.Terminator.Kind != mir.TerminatorCall {
				continue
			}
			if block.Terminator.ForeignABI != "" {
				node.externalCall = true
				report.Violations = append(report.Violations, violationAt(
					"CP1001",
					function.Name,
					block.Terminator.Span,
					"foreign calls are unavailable in the deterministic contract profile",
				))
				continue
			}
			targetIndex, found := symbolToIndex[block.Terminator.Callee]
			if !found {
				node.unknownInternalCall = true
				report.Violations = append(report.Violations, violationAt(
					"CP1006",
					function.Name,
					block.Terminator.Span,
					fmt.Sprintf("internal call target %d is absent from verified MIR", block.Terminator.Callee),
				))
				continue
			}
			node.callTargets = append(node.callTargets, targetIndex)
			node.internalCallNames = append(node.internalCallNames, nodes[targetIndex].function.Name)
		}
		if node.cfgCycle && loopBudgets[function.Name] == 0 {
			report.Violations = append(report.Violations, violationAt(
				"CP1007",
				function.Name,
				function.Span,
				"cyclic control flow requires @max_iter with a positive aggregate metered cycle-edge budget",
			))
		}
		if bound := loopBudgets[function.Name]; bound > MaxLoopBudget {
			report.Violations = append(report.Violations, violationAt(
				"CP0102",
				function.Name,
				function.Span,
				fmt.Sprintf("loop budget %d exceeds the profile maximum %d", bound, MaxLoopBudget),
			))
		}
	}

	markRecursiveFunctions(nodes)
	for index := range nodes {
		if !nodes[index].recursive {
			continue
		}
		report.Violations = append(report.Violations, violationAt(
			"CP1008",
			nodes[index].function.Name,
			nodes[index].function.Span,
			"recursive call cycles are unavailable in the deterministic contract profile",
		))
	}

	for name := range loopBudgets {
		if _, found := nameToIndex[name]; found {
			continue
		}
		report.Violations = append(report.Violations, Violation{
			Code:     "CP1010",
			Function: name,
			Message:  "@max_iter did not match a verified MIR function",
		})
	}

	costState := make([]uint8, len(nodes))
	for index := range nodes {
		computeTransitiveCost(index, nodes, loopBudgets, costState)
		if nodes[index].localCostOverflow || nodes[index].transitiveOverflow {
			report.Violations = append(report.Violations, violationAt(
				"CP1009",
				nodes[index].function.Name,
				nodes[index].function.Span,
				"abstract execution cost overflows uint64 and cannot be certified",
			))
		}
		bounded := nodes[index].transitiveBounded && !nodes[index].transitiveOverflow
		transitiveCost := uint64(0)
		if bounded {
			transitiveCost = nodes[index].transitiveCost
		}
		report.Functions = append(report.Functions, FunctionReport{
			Name:                         nodes[index].function.Name,
			HasControlFlowCycle:          nodes[index].cfgCycle,
			MeteredCycleEdges:            len(nodes[index].cycleEdges),
			DeclaredLoopBudget:           loopBudgets[nodes[index].function.Name],
			LocalAbstractCost:            nodes[index].localCost,
			TransitiveCostUpperBound:     transitiveCost,
			StorageGrowthUpperBoundBytes: 0,
			Bounded:                      bounded,
			InternalCalls:                slices.Clone(nodes[index].internalCallNames),
		})
	}

	report.Finalize()
	return report
}
