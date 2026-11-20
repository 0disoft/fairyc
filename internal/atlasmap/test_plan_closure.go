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
	trace := newClosureTrace(program, packageID)
	for _, fn := range program.Functions {
		source := byName[fn.Name]
		if source == "" || fn.Span.Package != packageID ||
			!slices.ContainsFunc(fn.Blocks, func(block mir.Block) bool {
				return block.Terminator.Kind == mir.TerminatorClosureCall
			}) {
			continue
		}
		// Copies of one site count as recovered only if every copy is resolved.
		sites := make(map[hir.Span]bool)
		for _, block := range fn.Blocks {
			call := block.Terminator
			if call.Kind != mir.TerminatorClosureCall {
				continue
			}
			symbol := trace.resolve(fn.Symbol, call.Arguments[0], "", nil, 0)
			if trace.work < 0 {
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
