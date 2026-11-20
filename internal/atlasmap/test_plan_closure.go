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

type testPlanClosureGraph[E comparable] struct {
	incoming  map[E][]E
	aliases   map[E]E
	recovered int
}

func (index *QueryIndex) testPlanClosures(program mir.Program, packageID hir.PackageID) testPlanClosureGraph[string] {
	for _, record := range index.completeness {
		if record.Relation == "calls" && slices.Contains(record.Reasons, indirectClosureOmission) {
			return index.buildTestPlanClosures(program, packageID)
		}
	}
	return testPlanClosureGraph[string]{}
}

func (index *QueryIndex) buildTestPlanClosures(program mir.Program, packageID hir.PackageID) testPlanClosureGraph[string] {
	byName := make(map[string]string)
	addTestPlanClosureSources(index, packageID, byName, func(symbol string) string { return symbol })
	graph := buildTestPlanClosureGraph(program, byName, func(pkg hir.PackageID) bool { return pkg == packageID },
		func(_ string, id int) string { return "\x00" + strconv.Itoa(id) })
	for target, sources := range graph.incoming {
		slices.Sort(sources)
		graph.incoming[target] = slices.Compact(sources)
	}
	return graph
}

func addTestPlanClosureSources[E comparable](index *QueryIndex, packageID hir.PackageID, byName map[string]E, endpoint func(string) E) {
	for _, node := range index.nodes {
		if node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction && len(node.declaration.TypeParameters) == 0 {
			byName[hir.QualifiedModulePath(packageID, testPlanLabel(node.declaration.Key))] = endpoint(node.symbol.Visible)
		}
	}
}

func buildTestPlanClosureGraph[E comparable](program mir.Program, byName map[string]E, include func(hir.PackageID) bool, alias func(E, int) E) testPlanClosureGraph[E] {
	targets := make(map[hir.SymbolID]E)
	sources := make(map[hir.SymbolID]E)
	for _, fn := range program.Functions {
		source, found := byName[fn.Name]
		if !found || !include(fn.Span.Package) {
			continue
		}
		sources[fn.Symbol] = source
		if fn.ClosureTarget {
			targets[fn.Symbol] = source
		}
	}

	graph := testPlanClosureGraph[E]{incoming: make(map[E][]E), aliases: make(map[E]E)}
	trace := newClosureTrace(program, include)
	hasSource := func(symbol hir.SymbolID) bool {
		_, found := sources[symbol]
		return found
	}
	for _, fn := range program.Functions {
		source, found := sources[fn.Symbol]
		if !found ||
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
			resolved := trace.invocation(fn.Symbol, call.Arguments[0], hasSource, func(target hir.SymbolID, frame *closureFrame) bool {
				to, found := targets[target]
				if !found {
					return false
				}
				graph.addInvocationPath(to, source, frame, sources, alias)
				return true
			})
			if trace.work < 0 {
				return testPlanClosureGraph[E]{}
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
	return graph
}

// Keep intermediate nodes private to the proven call path. A plain edge
// from apply to its closure target would also select callers of apply that
// supply a different closure.
func (graph *testPlanClosureGraph[E]) addInvocationPath(target, source E, frame *closureFrame, sources map[hir.SymbolID]E, alias func(E, int) E) {
	for frame != nil {
		step := alias(source, len(graph.aliases))
		graph.aliases[step] = source
		graph.incoming[target] = append(graph.incoming[target], step)
		source = sources[frame.call.caller]
		target = step
		frame = frame.parent
	}
	graph.incoming[target] = append(graph.incoming[target], source)
}
