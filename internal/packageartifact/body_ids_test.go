package packageartifact

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestFunctionGraphIDsCanonicalizeAfterFreshRebindAndPreserveExecution(t *testing.T) {
	t.Parallel()
	workspace, program, _, _ := checkedWorkspaceProgram(t)
	reboundProgram := mir.Clone(program)
	for index, function := range program.Functions {
		canonical, shape, err := canonicalizeFunctionGraphIDs(function)
		if err != nil {
			t.Fatalf("canonicalize %q: %v", function.Name, err)
		}
		rebound, err := rebindFunctionGraphIDs(canonical, shape, mir.LocalID(1000+index*1000))
		if err != nil {
			t.Fatalf("rebind %q: %v", function.Name, err)
		}
		roundTrip, roundTripShape, err := canonicalizeFunctionGraphIDs(rebound)
		if err != nil {
			t.Fatalf("recanonicalize %q: %v", function.Name, err)
		}
		if shape != roundTripShape || !reflect.DeepEqual(canonical, roundTrip) {
			t.Fatalf("function %q canonical graph changed after fresh rebind", function.Name)
		}
		reboundProgram.Functions[index] = rebound
	}
	if failures := mir.Verify(reboundProgram); len(failures) != 0 {
		t.Fatalf("freshly rebound program failed MIR verification: %v", failures[0])
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(reboundProgram, entry)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("freshly rebound execution = %+v", outcome)
	}
}

func TestFunctionGraphIDsRewriteRegionEntities(t *testing.T) {
	t.Parallel()
	function := mir.Function{
		Name: "main::region", Entry: 40,
		Blocks: []mir.Block{{
			ID: 40,
			Operations: []mir.Operation{
				{Kind: mir.OperationRegionEnter, Symbol: 71, DestinationLocal: mir.NoLocal},
				{Kind: mir.OperationRegionPlace, Symbol: 71, DestinationLocal: 92, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 81}}},
			},
			Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal},
		}},
		RegionValues: []mir.RegionValue{{Entity: "local:region-value:92", RegionEntity: "region:71"}},
	}
	canonical, shape, err := canonicalizeFunctionGraphIDs(function)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.RegionValues[0].Entity != "local:region-value:2" || canonical.RegionValues[0].RegionEntity != "region:0" {
		t.Fatalf("canonical region entities = %+v", canonical.RegionValues[0])
	}
	rebound, err := rebindFunctionGraphIDs(canonical, shape, 500)
	if err != nil {
		t.Fatal(err)
	}
	if rebound.RegionValues[0].Entity != "local:region-value:502" || rebound.RegionValues[0].RegionEntity != "region:500" {
		t.Fatalf("rebound region entities = %+v", rebound.RegionValues[0])
	}
}

func TestFunctionGraphIDsRewriteBurnLocalSymbol(t *testing.T) {
	t.Parallel()
	function := mir.Function{
		Name: "main::burn", Entry: 0,
		Resources: []mir.ResourceLocal{{Symbol: 71, Type: hir.Type("main::Coin"), Parameter: true}},
		Blocks: []mir.Block{{
			ID: 0,
			Operations: []mir.Operation{{
				Kind: mir.OperationBurn, Symbol: 71, DestinationLocal: mir.NoLocal, Type: hir.Type("main::Coin"),
			}},
			Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal},
		}},
	}
	canonical, shape, err := canonicalizeFunctionGraphIDs(function)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.Resources[0].Symbol != 0 || canonical.Blocks[0].Operations[0].Symbol != 0 {
		t.Fatalf("canonical burn locals = resource %d, operation %d", canonical.Resources[0].Symbol, canonical.Blocks[0].Operations[0].Symbol)
	}
	rebound, err := rebindFunctionGraphIDs(canonical, shape, 500)
	if err != nil {
		t.Fatal(err)
	}
	if rebound.Resources[0].Symbol != 500 || rebound.Blocks[0].Operations[0].Symbol != 500 {
		t.Fatalf("rebound burn locals = resource %d, operation %d", rebound.Resources[0].Symbol, rebound.Blocks[0].Operations[0].Symbol)
	}
}

func TestFunctionGraphIDsRejectMalformedNamespaces(t *testing.T) {
	t.Parallel()
	function := mir.Function{
		Name: "main::bad", Entry: 4,
		Blocks: []mir.Block{
			{ID: 4, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{9}, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal}},
			{ID: 4, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal}},
		},
	}
	if _, _, err := canonicalizeFunctionGraphIDs(function); err == nil || !strings.Contains(err.Error(), "duplicate block") {
		t.Fatalf("duplicate block error = %v", err)
	}
	valid := mir.Function{Name: "main::valid", Entry: 0, Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal}}}}
	if _, err := rebindFunctionGraphIDs(valid, bodyGraphShape{BlockCount: 1}, -1); err == nil {
		t.Fatal("negative local base accepted")
	}
}
