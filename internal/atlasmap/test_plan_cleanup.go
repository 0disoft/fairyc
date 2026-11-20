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

type testPlanCleanupGraph struct {
	incoming  map[string][]string
	recovered int
	complete  bool
}

// The map has no edges from drop obligations to their cleanup bodies. Those
// bindings come from the same verified MIR used to produce the query index.
func (index *QueryIndex) testPlanCleanups(result compiler.Result, packageID hir.PackageID, wantCleanup bool) testPlanCleanupGraph {
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
	return testPlanCleanupGraph{}
}

func (index *QueryIndex) buildTestPlanCleanups(result compiler.Result, packageID hir.PackageID) testPlanCleanupGraph {
	program := result.MIR
	functions := make(map[string]string)
	cleanupNames := make(map[hir.Type]string)
	for _, node := range index.nodes {
		key := node.declaration.Key
		if len(node.declaration.TypeParameters) != 0 {
			continue
		}
		name := hir.QualifiedModulePath(packageID, testPlanLabel(key))
		switch key.Kind {
		case semanticsnapshot.DeclarationFunction:
			functions[name] = node.symbol.Visible
		case semanticsnapshot.DeclarationResource:
			if node.declaration.DropKind == string(hir.ResourceDropNonEmpty) {
				cleanupNames[hir.ParseType(name)] = testPlanLabel(key) + testPlanCleanupSuffix
			}
		}
	}
	ordinaryCleanups := len(cleanupNames)
	genericCleanupTypes, expectedCleanups := index.bindTestPlanSpecializations(result.GenericProvenance, packageID, functions, cleanupNames)
	names := make(map[hir.SymbolID]string)
	matchedCleanups := 0
	for _, fn := range program.Functions {
		if fn.Span.Package != packageID {
			continue
		}
		if fn.DropGlueFor == (hir.Type{}) {
			names[fn.Symbol] = functions[fn.Name]
		} else {
			names[fn.Symbol] = cleanupNames[fn.DropGlueFor]
			if names[fn.Symbol] != "" {
				matchedCleanups++
			}
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
	graph := testPlanCleanupGraph{
		incoming: make(map[string][]string),
		complete: matchedCleanups == ordinaryCleanups+expectedCleanups,
	}
	work := 0
	for _, fn := range program.Functions {
		source := names[fn.Symbol]
		if source == "" {
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
					return testPlanCleanupGraph{}
				}
			}
			term := block.Terminator
			if fn.DropGlueFor == (hir.Type{}) || genericCleanupTypes[fn.DropGlueFor] || term.Kind != mir.TerminatorCall || term.ForeignABI != "" {
				continue
			}
			target := names[term.Callee]
			if target == "" {
				continue
			}
			site := callSite{span: term.Span, target: term.Callee}
			if _, seen := seenCalls[site]; seen {
				continue
			}
			work++
			if work > maxTestPlanCleanupWork {
				return testPlanCleanupGraph{}
			}
			seenCalls[site] = struct{}{}
			graph.recovered++
			graph.incoming[target] = append(graph.incoming[target], source)
		}
	}
	for target, sources := range graph.incoming {
		slices.Sort(sources)
		graph.incoming[target] = slices.Compact(sources)
	}
	return graph
}

// Join the exact implementation IDs recorded by Atlas to the compiler-owned
// generated names. Display labels and resource type spellings are independent.
func (index *QueryIndex) bindTestPlanSpecializations(provenance compiler.GenericProvenance, packageID hir.PackageID, functions map[string]string, cleanups map[hir.Type]string) (map[hir.Type]bool, int) {
	type implementation struct {
		module   string
		endpoint string
		cleanup  bool
	}
	implementations := make(map[string]implementation)
	expectedCleanups := 0
	for _, node := range index.nodes {
		for _, facts := range node.specializations {
			if facts.Function == nil && facts.Cleanup == nil {
				continue
			}
			implementations[facts.ID] = implementation{
				module: node.declaration.Key.Module, endpoint: testPlanEndpoint(node.symbol.Visible, facts.ID),
				cleanup: facts.Cleanup != nil,
			}
			if facts.Cleanup != nil {
				expectedCleanups++
			}
		}
	}
	types := make(map[hir.Type]bool, expectedCleanups)
	if len(implementations) == 0 {
		return types, expectedCleanups
	}
	for _, origin := range provenance.Specializations() {
		facts, found := implementations[string(origin.Specialization)]
		if !found {
			continue
		}
		name := hir.QualifiedModulePath(packageID, facts.module) + "::" + origin.GeneratedName
		if facts.cleanup {
			typ := hir.ParseType(name)
			cleanups[typ], types[typ] = facts.endpoint, true
		} else {
			functions[name] = facts.endpoint
		}
	}
	return types, expectedCleanups
}

func (graph testPlanCleanupGraph) addDropEdges(source string, root hir.Type, resources map[hir.Type]*mir.ResourceType, names map[hir.SymbolID]string, seen map[hir.Type]struct{}, work *int) bool {
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
			if cleanup := names[resource.DropSymbol]; cleanup != "" {
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
