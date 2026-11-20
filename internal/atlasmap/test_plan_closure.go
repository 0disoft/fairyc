package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const indirectClosureOmission = "indirect-closure-call"
const maxTestPlanClosureWork = 65536

type testPlanClosureGraph struct {
	incoming  map[string][]string
	recovered int
}

func (index *QueryIndex) testPlanClosures(program mir.Program, packageID hir.PackageID) testPlanClosureGraph {
	for _, record := range index.completeness {
		if record.Relation == "calls" && slices.Contains(record.Reasons, indirectClosureOmission) {
			return index.buildTestPlanClosures(program, packageID)
		}
	}
	return testPlanClosureGraph{}
}

func (index *QueryIndex) buildTestPlanClosures(program mir.Program, packageID hir.PackageID) testPlanClosureGraph {
	byName := make(map[string]string)
	for _, node := range index.nodes {
		if node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction && len(node.declaration.TypeParameters) == 0 {
			byName[hir.QualifiedModulePath(packageID, testPlanLabel(node.declaration.Key))] = node.symbol.Visible
		}
	}
	targets := make(map[hir.SymbolID]string)
	for _, fn := range program.Functions {
		if fn.Span.Package == packageID && fn.ClosureTarget {
			targets[fn.Symbol] = byName[fn.Name]
		}
	}

	graph := testPlanClosureGraph{incoming: make(map[string][]string)}
	work := maxTestPlanClosureWork
	for _, fn := range program.Functions {
		source := byName[fn.Name]
		if source == "" || fn.Span.Package != packageID ||
			!slices.ContainsFunc(fn.Blocks, func(block mir.Block) bool {
				return block.Terminator.Kind == mir.TerminatorClosureCall
			}) {
			continue
		}
		definitions := closureLocalDefinitions(fn)
		// Copies of one site count as recovered only if every copy is resolved.
		sites := make(map[hir.Span]bool)
		for _, block := range fn.Blocks {
			call := block.Terminator
			if call.Kind != mir.TerminatorClosureCall {
				continue
			}
			symbol := resolveLocalClosure(call.Arguments[0], definitions, &work)
			if work < 0 {
				return testPlanClosureGraph{}
			}
			target := targets[symbol]
			if _, seen := sites[call.Span]; !seen {
				sites[call.Span] = true
			}
			if target == "" {
				sites[call.Span] = false
				continue
			}
			graph.incoming[target] = append(graph.incoming[target], source)
		}
		for _, resolved := range sites {
			if resolved {
				graph.recovered++
			}
		}
	}
	for target, sources := range graph.incoming {
		slices.Sort(sources)
		graph.incoming[target] = slices.Compact(sources)
	}
	return graph
}

// The MIR verifier enforces a single definition for closure carriers. Mark
// ambiguous destinations anyway so ordinary multi-write locals cannot become
// accidental evidence when tracing a value.
func closureLocalDefinitions(fn mir.Function) map[mir.LocalID]*mir.Operation {
	definitions := make(map[mir.LocalID]*mir.Operation)
	for _, local := range fn.Parameters {
		definitions[local] = nil
	}
	for _, block := range fn.Blocks {
		if term := block.Terminator; term.Kind == mir.TerminatorCall || term.Kind == mir.TerminatorClosureCall {
			definitions[term.DestinationLocal] = nil
		}
		for i := range block.Operations {
			op := &block.Operations[i]
			switch op.Kind {
			case mir.OperationClosureCreate, mir.OperationAssign, mir.OperationFieldRead, mir.OperationStructConstruct:
				if _, found := definitions[op.DestinationLocal]; found {
					definitions[op.DestinationLocal] = nil
				} else {
					definitions[op.DestinationLocal] = op
				}
			}
		}
	}
	return definitions
}

func resolveLocalClosure(value mir.Operand, definitions map[mir.LocalID]*mir.Operation, work *int) hir.SymbolID {
	field := ""
	for value.Kind == mir.OperandLocal {
		*work--
		if *work < 0 {
			break
		}
		op := definitions[value.Local]
		if op == nil {
			break
		}
		switch op.Kind {
		case mir.OperationClosureCreate:
			if field == "" {
				return op.Symbol
			}
			return hir.NoSymbol
		case mir.OperationAssign:
			value = op.Operands[0]
		case mir.OperationFieldRead:
			if field != "" || op.Type != hir.TypeClosureI32 {
				return hir.NoSymbol
			}
			field, value = op.Field, op.Operands[0]
		case mir.OperationStructConstruct:
			*work -= len(op.Fields)
			if field == "" || *work < 0 {
				return hir.NoSymbol
			}
			position := slices.Index(op.Fields, field)
			if position < 0 {
				return hir.NoSymbol
			}
			field, value = "", op.Operands[position]
		}
	}
	return hir.NoSymbol
}
