package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const ordinaryCleanupOmission = "ordinary-cleanup-call-not-retained"
const maxTestPlanCleanupWork = 65536

type testPlanCleanupGraph[E comparable] struct {
	incoming  map[E][]E
	recovered int
	complete  bool
}

// The map has no edges from drop obligations to their cleanup bodies. Those
// bindings come from the same verified MIR used to produce the query index.
func (index *QueryIndex) testPlanCleanups(result compiler.Result, packageID hir.PackageID, wantCleanup bool) testPlanCleanupGraph[string] {
	if wantCleanup {
		return index.buildTestPlanCleanups(result, packageID)
	}
	for _, node := range index.nodes {
		for _, facts := range node.specializations {
			if facts.Cleanup != nil {
				return index.buildTestPlanCleanups(result, packageID)
			}
		}
	}
	for _, record := range index.completeness {
		if record.Relation == "calls" && slices.Contains(record.Reasons, ordinaryCleanupOmission) {
			return index.buildTestPlanCleanups(result, packageID)
		}
	}
	return testPlanCleanupGraph[string]{}
}

func (index *QueryIndex) buildTestPlanCleanups(result compiler.Result, packageID hir.PackageID) testPlanCleanupGraph[string] {
	bindings := testPlanCleanupBindings[string]{
		functions: make(map[string]string), cleanups: make(map[hir.Type]string),
	}
	bindings.addPackage(index, packageID, testPlanEndpoint)
	graph := buildTestPlanCleanupGraph(result, bindings)
	for target, sources := range graph.incoming {
		slices.Sort(sources)
		graph.incoming[target] = slices.Compact(sources)
	}
	return graph
}

func buildTestPlanCleanupGraph[E comparable](result compiler.Result, bindings testPlanCleanupBindings[E]) testPlanCleanupGraph[E] {
	program := result.MIR
	genericCleanupTypes := bindings.bindSpecializations(result.GenericProvenance)
	names := make(map[hir.SymbolID]E)
	matchedCleanups := 0
	for _, fn := range program.Functions {
		if fn.DropGlueFor == (hir.Type{}) {
			if endpoint, found := bindings.functions[fn.Name]; found {
				names[fn.Symbol] = endpoint
			}
		} else if endpoint, found := bindings.cleanups[fn.DropGlueFor]; found {
			names[fn.Symbol] = endpoint
			matchedCleanups++
		}
	}
	resources := make(map[hir.Type]*mir.ResourceType, len(program.ResourceTypes))
	for i := range program.ResourceTypes {
		resource := &program.ResourceTypes[i]
		resources[resource.Type] = resource
	}

	type callSite struct {
		span   hir.Span
		target hir.SymbolID
	}
	graph := testPlanCleanupGraph[E]{
		incoming: make(map[E][]E),
		complete: matchedCleanups == bindings.expectedCleanups,
	}
	work := 0
	for _, fn := range program.Functions {
		source, found := names[fn.Symbol]
		if !found {
			continue
		}
		locals := make(map[hir.SymbolID]hir.Type, len(fn.Resources))
		for _, local := range fn.Resources {
			locals[local.Symbol] = local.Type
		}
		seenTypes := make(map[hir.Type]struct{})
		seenCalls := make(map[callSite]struct{})
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if op.Kind != mir.OperationDrop {
					continue
				}
				root, found := locals[op.Symbol]
				if !found || !graph.addDropEdges(source, root, resources, names, seenTypes, &work) {
					return testPlanCleanupGraph[E]{}
				}
			}
			term := block.Terminator
			if fn.DropGlueFor == (hir.Type{}) || genericCleanupTypes[fn.DropGlueFor] || term.Kind != mir.TerminatorCall || term.ForeignABI != "" {
				continue
			}
			target, found := names[term.Callee]
			if !found {
				continue
			}
			site := callSite{span: term.Span, target: term.Callee}
			if _, seen := seenCalls[site]; seen {
				continue
			}
			work++
			if work > maxTestPlanCleanupWork {
				return testPlanCleanupGraph[E]{}
			}
			seenCalls[site] = struct{}{}
			graph.recovered++
			graph.incoming[target] = append(graph.incoming[target], source)
		}
	}
	return graph
}

type testPlanCleanupBinding[E comparable] struct {
	module   string
	endpoint E
	cleanup  bool
}

type testPlanCleanupBindings[E comparable] struct {
	functions        map[string]E
	cleanups         map[hir.Type]E
	implementations  map[string]testPlanCleanupBinding[E]
	expectedCleanups int
}

func (bindings *testPlanCleanupBindings[E]) addPackage(index *QueryIndex, packageID hir.PackageID, endpoint func(string, string) E) {
	for _, node := range index.nodes {
		key := node.declaration.Key
		name := hir.QualifiedModulePath(packageID, testPlanLabel(key))
		if len(node.declaration.TypeParameters) == 0 {
			switch key.Kind {
			case semanticsnapshot.DeclarationFunction:
				bindings.functions[name] = endpoint(node.symbol.Visible, "")
			case semanticsnapshot.DeclarationResource:
				if node.declaration.DropKind == string(hir.ResourceDropNonEmpty) {
					bindings.cleanups[hir.ParseType(name)] = endpoint(testPlanLabel(key)+testPlanCleanupSuffix, "")
					bindings.expectedCleanups++
				}
			}
		}
		for _, facts := range node.specializations {
			if facts.Function == nil && facts.Cleanup == nil {
				continue
			}
			if bindings.implementations == nil {
				bindings.implementations = make(map[string]testPlanCleanupBinding[E])
			}
			bindings.implementations[facts.ID] = testPlanCleanupBinding[E]{
				module: hir.QualifiedModulePath(packageID, key.Module), endpoint: endpoint(node.symbol.Visible, facts.ID),
				cleanup: facts.Cleanup != nil,
			}
			if facts.Cleanup != nil {
				bindings.expectedCleanups++
			}
		}
	}
}

// Join the exact implementation IDs recorded by Atlas to the compiler-owned
// generated names. Display labels and resource type spellings are independent.
func (bindings testPlanCleanupBindings[E]) bindSpecializations(provenance compiler.GenericProvenance) map[hir.Type]bool {
	if len(bindings.implementations) == 0 {
		return nil
	}
	types := make(map[hir.Type]bool)
	for _, origin := range provenance.Specializations() {
		facts, found := bindings.implementations[string(origin.Specialization)]
		if !found {
			continue
		}
		name := facts.module + "::" + origin.GeneratedName
		if facts.cleanup {
			typ := hir.ParseType(name)
			bindings.cleanups[typ], types[typ] = facts.endpoint, true
		} else {
			bindings.functions[name] = facts.endpoint
		}
	}
	return types
}

func (graph testPlanCleanupGraph[E]) addDropEdges(source E, root hir.Type, resources map[hir.Type]*mir.ResourceType, names map[hir.SymbolID]E, seen map[hir.Type]struct{}, work *int) bool {
	pending := []hir.Type{root}
	for len(pending) != 0 {
		typ := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		*work++
		if *work > maxTestPlanCleanupWork {
			return false
		}
		if _, found := seen[typ]; found {
			continue
		}
		seen[typ] = struct{}{}
		if resource := resources[typ]; resource != nil {
			if cleanup, found := names[resource.DropSymbol]; found {
				graph.incoming[cleanup] = append(graph.incoming[cleanup], source)
			}
			for _, field := range resource.Fields {
				if resources[field.Type] != nil {
					pending = append(pending, field.Type)
				}
			}
		} else if element, ok := hir.OptionElement(typ); ok {
			pending = append(pending, element)
		} else if success, failure, ok := hir.ResultElements(typ); ok {
			pending = append(pending, success, failure)
		}
	}
	return true
}
