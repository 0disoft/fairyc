package atlasmap

import (
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const (
	maxTestPlanDepth      = 64
	maxTestPlanSymbols    = 4096
	testPlanCleanupSuffix = "::cleanup"
)

type TestPlanPath struct {
	Test string   `json:"test"`
	Path []string `json:"path"`
}

// TestPlan shows one shortest exact local path per discovered test. Keep known
// paths when coverage is incomplete so the caller can recommend the full suite.
// result and packageID must come from the verified compilation used by index.
func (index *QueryIndex) TestPlan(selector string, tests []string, result compiler.Result, packageID hir.PackageID) ([]TestPlanPath, []string) {
	selector, wantCleanup := strings.CutSuffix(selector, testPlanCleanupSuffix)
	subjects, found := index.testPlanSubjects(selector, wantCleanup)
	paths := []TestPlanPath{}
	if !found {
		return paths, []string{"symbol-not-found"}
	}
	cleanups := index.testPlanCleanups(result, packageID, wantCleanup)
	closures := index.testPlanClosures(result.MIR, packageID)
	reasons, specializedIncoming := index.testPlanCallFacts(cleanups.recovered+closures.recovered, cleanups.complete)
	if len(subjects) == 0 {
		if wantCleanup {
			return paths, append(reasons, "cleanup-call-coverage")
		}
		return paths, append(reasons, "generic-call-coverage")
	}
	if wantCleanup && !cleanups.complete {
		reasons = append(reasons, "cleanup-call-coverage")
	}
	type step struct {
		next  string
		depth int
	}
	truncated := len(subjects) > maxTestPlanSymbols
	queue := subjects[:min(len(subjects), maxTestPlanSymbols)]
	parents := make(map[string]step, len(queue))
	for _, subject := range queue {
		parents[subject] = step{}
	}
	for i := 0; i < len(queue); i++ {
		target := queue[i]
		depth := parents[target].depth
		enqueue := func(source string) {
			if _, visited := parents[source]; visited {
				return
			}
			if depth == maxTestPlanDepth || len(queue) == maxTestPlanSymbols {
				truncated = true
				return
			}
			parents[source] = step{next: target, depth: depth + 1}
			queue = append(queue, source)
		}
		if strings.Contains(target, "#") {
			for _, source := range specializedIncoming[target] {
				enqueue(source)
			}
		} else {
			for _, pos := range index.incoming[target] {
				if call := index.relations[pos]; index.localTestPlanCall(call, cleanups.complete) && call.Target.Specialization == "" {
					enqueue(testPlanEndpoint(call.Source, call.SourceSpecialization))
				}
			}
		}
		for _, source := range cleanups.incoming[target] {
			enqueue(source)
		}
		for _, source := range closures.incoming[target] {
			enqueue(source)
		}
	}
	if truncated {
		reasons = append(reasons, "traversal-limit")
	}

	testSymbols := make(map[string]string, len(tests))
	for _, node := range index.nodes {
		if index.files[node.declaration.Span.Source].Kind == semanticsnapshot.FileKindTest &&
			node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction {
			testSymbols[testPlanLabel(node.declaration.Key)] = node.symbol.Visible
		}
	}
	unmapped := false
	for _, test := range tests {
		symbol, exists := testSymbols[test]
		if !exists {
			unmapped = true
			continue
		}
		if _, reached := parents[symbol]; !reached {
			continue
		}
		path := make([]string, 0, parents[symbol].depth+1)
		for at := symbol; at != ""; at = parents[at].next {
			if label, found := index.testPlanEndpointLabel(at); found {
				path = append(path, label)
			} else if label, found := closures.labels[at]; found {
				path = append(path, label)
			} else {
				path = append(path, at)
			}
		}
		paths = append(paths, TestPlanPath{Test: test, Path: path})
	}
	if unmapped {
		reasons = append(reasons, "unmapped-test")
	}
	return paths, reasons
}

func (index *QueryIndex) testPlanSubjects(selector string, wantCleanup bool) ([]string, bool) {
	name, concrete, hasArguments := strings.Cut(selector, "<")
	var specialization string
	if strings.Contains(selector, "#") {
		var err error
		name, specialization, err = parseImpactSelector(selector)
		if err != nil {
			return nil, false
		}
	}
	node, found := index.testPlanSubject(name, wantCleanup)
	if !found {
		return nil, false
	}
	if len(node.declaration.TypeParameters) == 0 {
		if hasArguments || specialization != "" {
			return nil, false
		}
		if wantCleanup {
			if node.declaration.DropKind != string(hir.ResourceDropNonEmpty) {
				return nil, true
			}
			return []string{testPlanLabel(node.declaration.Key) + testPlanCleanupSuffix}, true
		}
		return []string{node.symbol.Visible}, true
	}
	var subjects []string
	for _, facts := range node.specializations {
		hasBody := !wantCleanup && facts.Function != nil || wantCleanup && facts.Cleanup != nil
		if hasBody && (specialization == "" || facts.ID == specialization) &&
			(!hasArguments || concrete == testPlanTypeArguments(facts.TypeArguments)+">") {
			subjects = append(subjects, testPlanEndpoint(node.symbol.Visible, facts.ID))
		}
	}
	return subjects, !hasArguments && specialization == "" || len(subjects) != 0
}

func (index *QueryIndex) testPlanSubject(selector string, wantCleanup bool) (*indexedQueryNode, bool) {
	if position, err := index.resolveSymbol(selector); err == nil {
		node := &index.nodes[position]
		if wantCleanup {
			return node, node.declaration.Key.Kind == semanticsnapshot.DeclarationResource
		}
		return node, node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction ||
			node.declaration.Key.Kind == semanticsnapshot.DeclarationForeignFunction
	}
	separator := strings.LastIndex(selector, "::")
	if separator < 0 {
		return nil, false
	}
	key := semanticsnapshot.DeclarationKey{
		PackageInstance: index.header.PackageInstance,
		Module:          selector[:separator], Name: selector[separator+2:], Kind: semanticsnapshot.DeclarationFunction,
	}
	if wantCleanup {
		key.Kind = semanticsnapshot.DeclarationResource
	}
	position, found := index.byKey[key]
	if !found && !wantCleanup {
		key.Kind = semanticsnapshot.DeclarationForeignFunction
		position, found = index.byKey[key]
	}
	if !found {
		return nil, false
	}
	return &index.nodes[position], true
}

func (index *QueryIndex) testPlanCallFacts(recoveredCalls int, genericCleanupsComplete bool) ([]string, map[string][]string) {
	reasons := []string{}
	add := func(reason string) {
		if !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	callsExact := false
	for _, record := range index.completeness {
		if record.Relation == "calls" {
			callsExact = record.External == 0 &&
				(record.Status == semanticsnapshot.CompletenessExact && record.Unknown == 0 ||
					recoveredCalls > 0 && recoveredCalls == record.Unknown &&
						!slices.ContainsFunc(record.Reasons, func(reason string) bool {
							return reason != ordinaryCleanupOmission && reason != indirectClosureOmission
						}))
		}
	}
	if !callsExact {
		add("incomplete-call-coverage")
	}
	specialized := make(map[string][]string)
	for _, call := range index.relations {
		switch {
		case call.Kind == RelationCallsForeign:
			add("external-ffi")
		case call.Kind != RelationCalls:
			continue
		case call.Target != nil && call.Target.PackageInstance != index.header.PackageInstance:
			add("cross-package-call")
		case !index.localTestPlanCall(call, genericCleanupsComplete):
			add("unsupported-call-edge")
		case call.Target.Specialization != "":
			target := testPlanEndpoint(call.Target.Symbol, call.Target.Specialization)
			specialized[target] = append(specialized[target], testPlanEndpoint(call.Source, call.SourceSpecialization))
		}
	}
	slices.Sort(reasons)
	return reasons, specialized
}

func (index *QueryIndex) localTestPlanCall(call QueryRelation, genericCleanupsComplete bool) bool {
	return call.Kind == RelationCalls && call.Target != nil && call.Target.Symbol != "" &&
		call.Target.PackageInstance == index.header.PackageInstance &&
		(call.SourceBody == "" || genericCleanupsComplete && call.SourceBody == "cleanup" && call.SourceSpecialization != "") && call.Span != nil &&
		call.Certainty == semanticsnapshot.CertaintyExact && call.Provenance == semanticsnapshot.ProvenanceCompilerObserved
}

func testPlanEndpoint(symbol, specialization string) string {
	if specialization == "" {
		return symbol
	}
	return symbol + "#" + specialization
}

func testPlanTypeArguments(arguments []semanticsnapshot.TypeRef) string {
	names := make([]string, len(arguments))
	for i, argument := range arguments {
		names[i] = argument.Name
	}
	return strings.Join(names, ",")
}

func (index *QueryIndex) testPlanEndpointLabel(endpoint string) (string, bool) {
	symbol, specialization, concrete := strings.Cut(endpoint, "#")
	position, found := index.byVisible[symbol]
	if !found {
		return "", false
	}
	node := &index.nodes[position]
	label := testPlanLabel(node.declaration.Key)
	if concrete {
		for _, facts := range node.specializations {
			if facts.ID == specialization {
				label += "<" + testPlanTypeArguments(facts.TypeArguments) + ">"
				if facts.Cleanup != nil {
					label += testPlanCleanupSuffix
				}
				return label, true
			}
		}
		return "", false
	}
	return label, true
}

func testPlanLabel(key semanticsnapshot.DeclarationKey) string {
	return key.Module + "::" + key.Name
}
