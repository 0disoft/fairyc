package atlasmap

import (
	"slices"
	"strconv"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const indirectClosureOmission = "indirect-closure-call"
const maxTestPlanClosureWork = 65536

type testPlanClosureGraph struct {
	incoming  map[string][]string
	labels    map[string]string
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
	labels := make(map[string]string)
	for _, node := range index.nodes {
		if node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction && len(node.declaration.TypeParameters) == 0 {
			byName[hir.QualifiedModulePath(packageID, testPlanLabel(node.declaration.Key))] = node.symbol.Visible
			labels[node.symbol.Visible] = testPlanLabel(node.declaration.Key)
		}
	}
	targets := make(map[hir.SymbolID]string)
	sources := make(map[hir.SymbolID]string)
	for _, fn := range program.Functions {
		if fn.Span.Package != packageID {
			continue
		}
		sources[fn.Symbol] = byName[fn.Name]
		if fn.ClosureTarget {
			targets[fn.Symbol] = byName[fn.Name]
		}
	}

	graph := testPlanClosureGraph{incoming: make(map[string][]string), labels: make(map[string]string)}
	trace := newClosureTrace(program, packageID)
	for _, fn := range program.Functions {
		if sources[fn.Symbol] == "" ||
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
			resolved := trace.invocation(fn.Symbol, call.Arguments[0], sources, func(target hir.SymbolID, frame *closureFrame) bool {
				to := targets[target]
				if to == "" {
					return false
				}
				graph.addInvocationPath(to, sources[fn.Symbol], frame, sources, labels)
				return true
			})
			if trace.work < 0 {
				return testPlanClosureGraph{}
			}
			if prior, seen := sites[call.Span]; !seen || prior {
				sites[call.Span] = resolved
			}
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

// Keep intermediate nodes private to the proven call path. A plain edge
// from apply to its closure target would also select callers of apply that
// supply a different closure.
func (graph *testPlanClosureGraph) addInvocationPath(target, source string, frame *closureFrame, sources map[hir.SymbolID]string, labels map[string]string) {
	for frame != nil {
		step := "\x00" + strconv.Itoa(len(graph.labels))
		graph.labels[step] = labels[source]
		graph.incoming[target] = append(graph.incoming[target], step)
		source = sources[frame.call.caller]
		target = step
		frame = frame.parent
	}
	graph.incoming[target] = append(graph.incoming[target], source)
}
