package semanticsnapshot

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type omittedCallCoverage struct {
	cleanups int
	closures int
}

// Count source observations in verified MIR that the map cannot represent.
// A deferred call may appear in multiple blocks for the same source site.
func countOmittedCalls(program mir.Program, summarizedCleanups map[hir.SymbolID]DeclarationKey) omittedCallCoverage {
	type site struct {
		span   hir.Span
		callee hir.SymbolID
		kind   mir.TerminatorKind
	}
	var seen map[site]struct{}
	var coverage omittedCallCoverage
	for _, function := range program.Functions {
		_, summarized := summarizedCleanups[function.Symbol]
		omittedCleanup := function.DropGlueFor != hir.ParseType("") && !summarized
		clear(seen)
		for _, block := range function.Blocks {
			term := block.Terminator
			if term.Kind != mir.TerminatorClosureCall && !(omittedCleanup && term.Kind == mir.TerminatorCall) {
				continue
			}
			key := site{span: term.Span, callee: term.Callee, kind: term.Kind}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			if seen == nil {
				seen = make(map[site]struct{})
			}
			seen[key] = struct{}{}
			if term.Kind == mir.TerminatorClosureCall {
				coverage.closures++
			} else {
				coverage.cleanups++
			}
		}
	}
	return coverage
}
