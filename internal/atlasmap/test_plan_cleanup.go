package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const ordinaryCleanupOmission = "ordinary-cleanup-call-not-retained"
const maxTestPlanCleanupWork = 65536

type testPlanCleanupGraph struct {
	incoming  map[string][]string
	recovered int
}

// The persisted map reports these calls as omitted. Only the CLI's fresh,
// verified MIR can attach executable drop obligations and private cleanup bodies.
func (index *QueryIndex) testPlanCleanups(program mir.Program, packageID hir.PackageID) testPlanCleanupGraph {
	for _, record := range index.completeness {
		if record.Relation == "calls" && slices.Contains(record.Reasons, ordinaryCleanupOmission) {
			return index.buildTestPlanCleanups(program, packageID)
		}
	}
	return testPlanCleanupGraph{}
}

func (index *QueryIndex) buildTestPlanCleanups(program mir.Program, packageID hir.PackageID) testPlanCleanupGraph {
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
			cleanupNames[hir.ParseType(name)] = testPlanLabel(key) + "::cleanup"
		}
	}
	names := make(map[hir.SymbolID]string)
	for _, fn := range program.Functions {
		if fn.Span.Package != packageID {
			continue
		}
		if fn.DropGlueFor == (hir.Type{}) {
			names[fn.Symbol] = functions[fn.Name]
		} else {
			names[fn.Symbol] = cleanupNames[fn.DropGlueFor]
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
	graph := testPlanCleanupGraph{incoming: make(map[string][]string)}
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
			if fn.DropGlueFor == (hir.Type{}) || term.Kind != mir.TerminatorCall || term.ForeignABI != "" {
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
