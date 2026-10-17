package contractprofile

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAnalyzeAcceptsBoundedControlFlowCycleAndReportsCost(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{{
		Symbol:     1,
		Name:       "main::main",
		Span:       hir.Span{Source: "main.fy", StartByte: 0, EndByte: 80},
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []mir.Block{
			{
				ID:         0,
				Operations: []mir.Operation{{Kind: mir.OperationAssign}},
				Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}},
			},
			{
				ID:         1,
				Operations: []mir.Operation{{Kind: mir.OperationArithmeticCheck}},
				Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{1, 2}},
			},
			{
				ID:         2,
				Terminator: mir.Terminator{Kind: mir.TerminatorReturn},
			},
		},
	}}}
	report := Analyze(program, map[string]uint64{"main::main": 8})
	if !report.Accepted || report.Status != "accepted" || len(report.Violations) != 0 {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("functions = %+v", report.Functions)
	}
	function := report.Functions[0]
	if !function.HasControlFlowCycle || function.MeteredCycleEdges == 0 || function.DeclaredLoopBudget != 8 || !function.Bounded {
		t.Fatalf("function bound = %+v", function)
	}
	if function.LocalAbstractCost == 0 || function.TransitiveCostUpperBound <= function.LocalAbstractCost {
		t.Fatalf("function cost = %+v", function)
	}
	if function.StorageGrowthUpperBoundBytes != 0 {
		t.Fatalf("storage growth = %d", function.StorageGrowthUpperBoundBytes)
	}
}

func TestAnalyzeRejectsCycleWithoutLoopBudget(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{{
		Symbol:     1,
		Name:       "main::main",
		Span:       hir.Span{Source: "main.fy", StartByte: 0, EndByte: 40},
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []mir.Block{{
			ID:         0,
			Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{0}},
		}},
	}}}
	report := Analyze(program, nil)
	if report.Accepted || !hasViolationCode(report, "CP1007") {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Functions) != 1 || report.Functions[0].Bounded {
		t.Fatalf("functions = %+v", report.Functions)
	}
}

func TestAnalyzeRejectsAmbientAuthorityRawPointersAndRegionAllocation(t *testing.T) {
	t.Parallel()
	program := mir.Program{
		StructTypes: []mir.StructType{{
			Type:  hir.ParseType("main::FloatBox"),
			Name:  "main::FloatBox",
			ReprC: true,
			Fields: []mir.StructField{
				{
					Name: "value",
					Type: hir.ParseType("f64"),
					Span: hir.Span{Source: "main.fy", StartByte: 0, EndByte: 4},
				},
				{
					Name: "target_long",
					Type: hir.TypeCLong,
					Span: hir.Span{Source: "main.fy", StartByte: 5, EndByte: 12},
				},
				{
					Name: "target_size",
					Type: hir.TypeUSize,
					Span: hir.Span{Source: "main.fy", StartByte: 13, EndByte: 20},
				},
			},
		}},
		ForeignFunctions: []mir.ForeignFunction{{
			Symbol:     2,
			Name:       "main::host",
			Span:       hir.Span{Source: "main.fy", StartByte: 0, EndByte: 10},
			ABI:        "C",
			LinkName:   "host",
			ResultType: hir.TypeI32,
		}},
		Functions: []mir.Function{{
			Symbol:           1,
			Name:             "main::main",
			Span:             hir.Span{Source: "main.fy", StartByte: 11, EndByte: 100},
			ResultType:       hir.RawPointer(hir.TypeU8, false),
			Effects:          []hir.Effect{hir.EffectClock},
			UnsafeOperations: []mir.UnsafeOperation{{Entity: "unsafe:1"}},
			Entry:            0,
			Blocks: []mir.Block{{
				ID: 0,
				Operations: []mir.Operation{
					{Kind: mir.OperationRawAddress, Span: hir.Span{Source: "main.fy", StartByte: 20, EndByte: 25}},
					{Kind: mir.OperationRegionPlace, Span: hir.Span{Source: "main.fy", StartByte: 26, EndByte: 31}},
				},
				Terminator: mir.Terminator{
					Kind:       mir.TerminatorCall,
					ForeignABI: "C",
					Span:       hir.Span{Source: "main.fy", StartByte: 32, EndByte: 38},
				},
			}},
		}},
	}
	report := Analyze(program, nil)
	for _, code := range []string{"CP1001", "CP1002", "CP1003", "CP1004", "CP1005", "CP1013", "CP1014", "CP1015", "CP1016"} {
		if !hasViolationCode(report, code) {
			t.Fatalf("missing %s in %+v", code, report.Violations)
		}
	}
	if report.Accepted {
		t.Fatalf("report unexpectedly accepted: %+v", report)
	}
}

func TestAnalyzeRejectsDirectAndMutualRecursion(t *testing.T) {
	t.Parallel()
	callFunction := func(symbol hir.SymbolID, name string, target hir.SymbolID) mir.Function {
		return mir.Function{
			Symbol:     symbol,
			Name:       name,
			Span:       hir.Span{Source: name + ".fy", StartByte: 0, EndByte: 20},
			ResultType: hir.TypeI32,
			Entry:      0,
			Blocks: []mir.Block{
				{
					ID: 0,
					Terminator: mir.Terminator{
						Kind:       mir.TerminatorCall,
						Callee:     target,
						Successors: []mir.BlockID{1},
					},
				},
				{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}},
			},
		}
	}
	program := mir.Program{Functions: []mir.Function{
		callFunction(1, "a::run", 2),
		callFunction(2, "b::run", 1),
		callFunction(3, "self::run", 3),
	}}
	report := Analyze(program, nil)
	count := 0
	for _, violation := range report.Violations {
		if violation.Code == "CP1008" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("recursion violations = %d, report = %+v", count, report)
	}
	for _, function := range report.Functions {
		if function.Bounded {
			t.Fatalf("recursive function is bounded: %+v", function)
		}
	}
}

func TestReportJSONIsDeterministic(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{
		{
			Symbol:     2,
			Name:       "z::run",
			ResultType: hir.TypeI32,
			Entry:      0,
			Blocks:     []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}}},
		},
		{
			Symbol:     1,
			Name:       "a::run",
			ResultType: hir.TypeI32,
			Entry:      0,
			Blocks:     []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}}},
		},
	}}
	first, err := Analyze(program, map[string]uint64{"z::run": 3, "a::run": 2}).JSON()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Analyze(program, map[string]uint64{"a::run": 2, "z::run": 3}).JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first, second) {
		t.Fatalf("JSON differs:\nfirst=%s\nsecond=%s", first, second)
	}
}

func hasViolationCode(report Report, code string) bool {
	for _, violation := range report.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
