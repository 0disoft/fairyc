package mirverifyfixture

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

const (
	valueSymbol  hir.SymbolID = 1
	tokenSymbol  hir.SymbolID = 2
	regionSymbol hir.SymbolID = 3
	arenaSymbol  hir.SymbolID = 4
)

var seedArenaType = hir.Type("seed::Arena")

func constructSeed(id string) (mir.Program, error) {
	var function mir.Function
	var resourceTypes []mir.ResourceType
	switch id {
	case "scalar-return":
		function = scalarReturnSeed()
	case "resource-return":
		function = resourceReturnSeed()
		resourceTypes = []mir.ResourceType{{
			Type: hir.Type("seed::Token"), Entity: "resource-type:seed-token",
			Fields: []mir.ResourceField{{Name: "value", Type: hir.TypeI32}}, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
		}}
	case "borrow-return":
		function = borrowReturnSeed()
	case "plain-region":
		function = plainRegionSeed()
		resourceTypes = []mir.ResourceType{seedArenaResourceType()}
	case "break-region-control":
		function = breakRegionControlSeed()
		resourceTypes = []mir.ResourceType{seedArenaResourceType()}
	case "checked-division":
		function = checkedDivisionSeed()
	case "checked-arithmetic":
		function = checkedArithmeticSeed()
	case "normal-call":
		function = normalCallSeed()
	case "normalized-control":
		function = normalizedControlSeed()
	case "unsafe-operation":
		function = unsafeOperationSeed()
	default:
		return mir.Program{}, fmt.Errorf("unknown seed %q", id)
	}
	return mir.Program{ResourceTypes: resourceTypes, Functions: []mir.Function{function}}, nil
}

func seedArenaResourceType() mir.ResourceType {
	return mir.ResourceType{
		Type: seedArenaType, Entity: "resource-type:seed-arena",
		Fields:   []mir.ResourceField{{Name: "opaque", Type: hir.TypeI32}},
		DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
	}
}

func scalarReturnSeed() mir.Function {
	return mir.Function{
		Name:         "seed::scalar-return",
		Entry:        0,
		ReturnEntity: "terminator:return",
		Entities: []string{
			"block:entry", "local:value", "operation:return-value-use", "statement:init-value", "terminator:return",
		},
		Blocks: []mir.Block{{
			ID: 0, Entity: "block:entry",
			Operations: []mir.Operation{
				{Kind: mir.OperationInitialize, Entity: "statement:init-value", Symbol: valueSymbol},
				{Kind: mir.OperationUse, Entity: "operation:return-value-use", Symbol: valueSymbol},
			},
			Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock},
		}},
	}
}

func resourceReturnSeed() mir.Function {
	return mir.Function{
		Name:         "seed::resource-return",
		Entry:        0,
		ReturnEntity: "terminator:return",
		Entities: []string{
			"block:cleanup", "block:entry", "flag:token-live", "local:token", "place:token-field",
			"resource-type:seed-token",
			"statement:clear-token-flag", "statement:drop-token", "statement:init-token-flag", "terminator:return",
		},
		Resources: []mir.ResourceLocal{{Symbol: tokenSymbol, Type: hir.Type("seed::Token"), Parameter: true, Entity: "local:token"}},
		DropFlags: []mir.DropFlag{{Entity: "flag:token-live", Symbol: tokenSymbol}},
		Blocks: []mir.Block{
			{
				ID: 0, Entity: "block:entry",
				Operations: []mir.Operation{{Kind: mir.OperationDropFlagInit, Entity: "statement:init-token-flag", Symbol: tokenSymbol, DropFlagEntity: "flag:token-live"}},
				Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock},
			},
			{
				ID: 1, Entity: "block:cleanup",
				Operations: []mir.Operation{
					{Kind: mir.OperationDropFlagClear, Entity: "statement:clear-token-flag", Symbol: tokenSymbol, DropFlagEntity: "flag:token-live"},
					{Kind: mir.OperationDrop, Entity: "statement:drop-token", Symbol: tokenSymbol, DropFlagEntity: "flag:token-live"},
				},
				Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock},
			},
		},
	}
}

func borrowReturnSeed() mir.Function {
	return mir.Function{
		Name:         "seed::borrow-return",
		Entry:        0,
		ReturnEntity: "terminator:return",
		Entities: []string{
			"loan:param-a", "local:param-a", "local:param-b", "origin:param-a", "origin:param-b", "terminator:return",
		},
		Loans:         []mir.Loan{{Entity: "loan:param-a", OriginEntity: "origin:param-a"}},
		ReturnOrigins: []mir.ReturnOrigin{{Entity: "origin:param-a", Returned: true}, {Entity: "origin:param-b"}},
		Blocks: []mir.Block{{
			ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock},
		}},
	}
}

func plainRegionSeed() mir.Function {
	return mir.Function{
		Name:      "seed::plain-region",
		Entry:     0,
		Resources: []mir.ResourceLocal{{Symbol: arenaSymbol, Type: seedArenaType, Parameter: true, Entity: "local:arena"}},
		Entities: []string{
			"local:inert-value", "local:resource-value", "operation:place-inert", "operation:region-reset", "region:scratch", "terminator:return",
		},
		RegionValues: []mir.RegionValue{{
			Entity: "local:inert-value", OperationEntity: "operation:place-inert", RegionEntity: "region:scratch",
		}},
		Blocks: []mir.Block{{
			ID: 0,
			Operations: []mir.Operation{
				{Kind: mir.OperationRegionEnter, Entity: "region:scratch", Symbol: regionSymbol, DestinationLocal: mir.NoLocal,
					Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: mir.LocalID(arenaSymbol)}}, OwnerType: seedArenaType,
					Type: hir.MutableBorrow(seedArenaType), Operator: "&mut"},
				{Kind: mir.OperationRegionPlace, Entity: "operation:place-inert", Symbol: regionSymbol, DestinationLocal: mir.LocalID(valueSymbol), Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 1}}, Type: hir.TypeI32},
				{Kind: mir.OperationRegionReset, Entity: "operation:region-reset", Symbol: regionSymbol, DestinationLocal: mir.NoLocal, ExitKind: "fallthrough"},
				{Kind: mir.OperationMove, Symbol: arenaSymbol},
			},
			Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock, ReturnValue: mir.Operand{Kind: mir.OperandI32, I32: 0}},
		}},
	}
}

func breakRegionControlSeed() mir.Function {
	return mir.Function{
		Name:       "seed::break-region-control",
		ResultType: hir.TypeI32,
		Entry:      0,
		Resources:  []mir.ResourceLocal{{Symbol: arenaSymbol, Type: seedArenaType, Parameter: true, Entity: "local:arena"}},
		Entities: []string{
			"block:break-body", "block:loop-exit", "local:placed", "operation:discard-placement",
			"operation:place-inert", "operation:region-reset", "region:scratch", "terminator:return",
		},
		RegionValues: []mir.RegionValue{{
			Entity: "local:placed", OperationEntity: "operation:place-inert", RegionEntity: "region:scratch",
		}},
		Blocks: []mir.Block{
			{
				ID: 0, Entity: "block:break-body",
				Operations: []mir.Operation{
					{Kind: mir.OperationRegionEnter, Entity: "region:scratch", Symbol: regionSymbol, DestinationLocal: mir.NoLocal,
						Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: mir.LocalID(arenaSymbol)}}, OwnerType: seedArenaType,
						Type: hir.MutableBorrow(seedArenaType), Operator: "&mut"},
					{Kind: mir.OperationRegionPlace, Entity: "operation:place-inert", Symbol: regionSymbol, DestinationLocal: mir.LocalID(valueSymbol), Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 1}}, Type: hir.TypeI32},
					{Kind: mir.OperationMove, Entity: "operation:discard-placement", Symbol: valueSymbol},
					{Kind: mir.OperationRegionReset, Entity: "operation:region-reset", Symbol: regionSymbol, DestinationLocal: mir.NoLocal, ExitKind: "break"},
				},
				Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock},
			},
			{
				ID: 1, Entity: "block:loop-exit",
				Operations: []mir.Operation{{Kind: mir.OperationMove, Symbol: arenaSymbol}},
				Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock, ReturnValue: mir.Operand{Kind: mir.OperandI32, I32: 0}},
			},
		},
	}
}

func checkedDivisionSeed() mir.Function {
	operands := []mir.Operand{{Kind: mir.OperandI32, I32: 4}, {Kind: mir.OperandI32, I32: 2}}
	categories := []string{"division-by-zero", "signed-division-overflow"}
	return mir.Function{
		Name:       "seed::checked-division",
		ResultType: hir.TypeI32,
		Entry:      0,
		Entities: []string{
			"block:check", "block:continuation", "block:trap", "operation:division", "operation:division-result", "operation:min-neg-one-check", "operation:zero-check", "terminator:trap",
		},
		Blocks: []mir.Block{
			{
				ID: 0, Entity: "block:check",
				Operations: []mir.Operation{{
					Kind: mir.OperationCheckedDivision, Entity: "operation:division",
					Operands: operands, OwnerType: hir.TypeI32, Type: hir.TypeI32,
					Operator: "/", TrapCategories: categories,
				}},
				Terminator: mir.Terminator{
					Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{1, 2}, UnwindSuccessor: mir.NoBlock,
					SwitchKind: "checked-division", SwitchType: hir.TypeI32,
					SwitchOperands: operands, Operator: "/",
				},
			},
			{
				ID: 1, Entity: "block:continuation",
				Operations: []mir.Operation{{
					Kind: mir.OperationBinary, Entity: "operation:division-result", DestinationLocal: 1,
					Operands: operands, OwnerType: hir.TypeI32, Type: hir.TypeI32, Operator: "/",
				}},
				Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
					ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 1},
				},
			},
			{
				ID: 2, Entity: "block:trap",
				Terminator: mir.Terminator{
					Kind: mir.TerminatorTrap, Entity: "terminator:trap", UnwindSuccessor: mir.NoBlock,
					TrapCategories: categories,
				},
			},
		},
	}
}

func checkedArithmeticSeed() mir.Function {
	operands := []mir.Operand{{Kind: mir.OperandI32, I32: 2147483647}, {Kind: mir.OperandI32, I32: 1}}
	categories := []string{"integer-overflow"}
	return mir.Function{
		Name:       "seed::checked-arithmetic",
		ResultType: hir.TypeI32,
		Entry:      0,
		Entities: []string{
			"block:arithmetic-check", "block:arithmetic-continuation", "block:arithmetic-trap",
			"operation:add-check", "operation:add-result", "terminator:arithmetic-trap",
		},
		Blocks: []mir.Block{
			{
				ID: 0, Entity: "block:arithmetic-check",
				Operations: []mir.Operation{{
					Kind: mir.OperationArithmeticCheck, Entity: "operation:add-check",
					Operands: operands, OwnerType: hir.TypeI32, Type: hir.TypeI32,
					Operator: "+", TrapCategories: categories,
				}},
				Terminator: mir.Terminator{
					Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{1, 2}, UnwindSuccessor: mir.NoBlock,
					SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI32, SwitchOperands: operands, Operator: "+",
				},
			},
			{
				ID: 1, Entity: "block:arithmetic-continuation",
				Operations: []mir.Operation{{
					Kind: mir.OperationBinary, Entity: "operation:add-result", DestinationLocal: 1,
					Operands: operands, OwnerType: hir.TypeI32, Type: hir.TypeI32, Operator: "+",
				}},
				Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
					ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 1},
				},
			},
			{
				ID: 2, Entity: "block:arithmetic-trap",
				Terminator: mir.Terminator{
					Kind: mir.TerminatorTrap, Entity: "terminator:arithmetic-trap", UnwindSuccessor: mir.NoBlock,
					TrapCategories: categories,
				},
			},
		},
	}
}

func normalCallSeed() mir.Function {
	return mir.Function{
		Name:       "seed::normal-call",
		ResultType: hir.TypeVoid,
		Entry:      0,
		Entities:   []string{"block:continuation", "terminator:call"},
		Blocks: []mir.Block{
			{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Entity: "terminator:call", DestinationLocal: mir.NoLocal, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
			{ID: 1, Entity: "block:continuation", Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
		},
	}
}

func normalizedControlSeed() mir.Function {
	return mir.Function{
		Name:         "seed::normalized-control",
		Entry:        0,
		ReturnEntity: "terminator:return",
		Entities:     []string{"operation:result-construct", "operation:result-switch", "terminator:return"},
		Blocks: []mir.Block{{
			ID: 0,
			Operations: []mir.Operation{
				{Kind: mir.OperationResultConstruct, Entity: "operation:result-construct", DestinationLocal: 1, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 0}}, Variant: "ok", Type: hir.Result(hir.TypeI32, hir.TypeI32)},
				{Kind: mir.OperationResultSwitch, Entity: "operation:result-switch", Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 1}}},
			},
			Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock},
		}},
	}
}

func unsafeOperationSeed() mir.Function {
	return mir.Function{
		Name:     "seed::unsafe-operation",
		Entry:    0,
		Entities: []string{"scope:unsafe-block", "span:unsafe-source", "terminator:return", "unsafe:raw-address"},
		UnsafeOperations: []mir.UnsafeOperation{{
			Entity: "unsafe:raw-address", SourceSpanEntity: "span:unsafe-source", ScopeEntity: "scope:unsafe-block",
		}},
		Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, Entity: "terminator:return", UnwindSuccessor: mir.NoBlock}}},
	}
}

func seedEntities(program mir.Program) []string {
	if len(program.Functions) != 1 {
		return nil
	}
	return slices.Clone(program.Functions[0].Entities)
}
