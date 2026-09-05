package contractstatenative

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// PrepareCompiled checks the published compiler result before creating detached native authority.
// Result is caller-mutable, so neither its Verified flag nor its state side table is trusted alone.
func PrepareCompiled(result compiler.Result, packageID hir.PackageID, function hir.SymbolID, coordinate packagegraph.CoordinateID) (Handoff, error) {
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow ||
		result.Semantic.FailedPhase != semantic.PhaseNone || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		return Handoff{}, fmt.Errorf("native handoff requires a successful compiler result")
	}
	if failures := mir.Verify(result.MIR); len(failures) != 0 {
		return Handoff{}, fmt.Errorf("native handoff requires verified ordinary MIR: %v", failures[0])
	}
	ordinary, err := mir.Lower(result.Semantic.HIR)
	if err != nil || !reflect.DeepEqual(ordinary, result.MIR) {
		return Handoff{}, fmt.Errorf("native handoff ordinary MIR differs from HIR")
	}
	expected, err := contractstatemir.LowerHIRTransactions(result.Semantic.HIR)
	if err != nil {
		return Handoff{}, err
	}
	if len(expected) != len(result.StateTransactions) {
		return Handoff{}, fmt.Errorf("native handoff state publication count differs from HIR")
	}
	for index, transaction := range result.StateTransactions {
		want := expected[index]
		if failures := contractstatemir.Verify(transaction.Program); len(failures) != 0 {
			return Handoff{}, fmt.Errorf("native handoff requires verified state MIR: %v", failures[0])
		}
		if transaction.Package != want.Package || transaction.Function != want.Function ||
			!sameStateFacts(transaction.Program, want.Program) {
			return Handoff{}, fmt.Errorf("native handoff state publication differs from HIR at transaction %d", index)
		}
		found := false
		for _, owner := range result.MIR.Functions {
			if owner.Symbol == transaction.Function {
				found = true
				break
			}
		}
		if !found {
			return Handoff{}, fmt.Errorf("native handoff state owner is absent from ordinary MIR")
		}
	}
	return Prepare(result.Semantic.HIR, packageID, function, coordinate)
}

// Fresh lowering creates distinct private capabilities. Compare every other represented fact;
// verification above separately checks each program's internal capability coherence.
func sameStateFacts(left, right contractstatemir.Program) bool {
	right.Capability = left.Capability
	normalize := func(operations []contractstatemir.Operation) []contractstatemir.Operation {
		copy := slices.Clone(operations)
		for index := range copy {
			copy[index].Capability = left.Capability
		}
		return copy
	}
	blocks := func(input []contractstatemir.Block) []contractstatemir.Block {
		copy := slices.Clone(input)
		for index := range copy {
			copy[index].Operations = normalize(copy[index].Operations)
		}
		return copy
	}
	right.Operations = normalize(right.Operations)
	right.Blocks = blocks(right.Blocks)
	right.Helpers = slices.Clone(right.Helpers)
	for index := range right.Helpers {
		right.Helpers[index].Operations = normalize(right.Helpers[index].Operations)
		right.Helpers[index].Blocks = blocks(right.Helpers[index].Blocks)
	}
	return reflect.DeepEqual(left, right)
}
