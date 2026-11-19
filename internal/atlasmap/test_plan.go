package atlasmap

import (
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const (
	maxTestPlanDepth   = 64
	maxTestPlanSymbols = 4096
)

type TestPlanPath struct {
	Test string   `json:"test"`
	Path []string `json:"path"`
}

// TestPlan shows one shortest exact local path per discovered test. Keep known
// paths when coverage is incomplete so the caller can recommend the full suite.
func (index *QueryIndex) TestPlan(selector string, tests []string) ([]TestPlanPath, []string) {
	subject, found := index.testPlanSubject(selector)
	paths := []TestPlanPath{}
	if !found {
		return paths, []string{"symbol-not-found"}
	}
	reasons := index.testPlanOmissions()
	type step struct {
		next  string
		depth int
	}
	parents := map[string]step{subject: {}}
	queue := []string{subject}
	truncated := false
	for i := 0; i < len(queue); i++ {
		target := queue[i]
		depth := parents[target].depth
		for _, pos := range index.incoming[target] {
			call := index.relations[pos]
			if !index.ordinaryTestPlanCall(call) {
				continue
			}
			if _, visited := parents[call.Source]; visited {
				continue
			}
			if depth == maxTestPlanDepth || len(queue) == maxTestPlanSymbols {
				truncated = true
				continue
			}
			parents[call.Source] = step{next: target, depth: depth + 1}
			queue = append(queue, call.Source)
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
			path = append(path, testPlanLabel(index.nodes[index.byVisible[at]].declaration.Key))
		}
		paths = append(paths, TestPlanPath{Test: test, Path: path})
	}
	if unmapped {
		reasons = append(reasons, "unmapped-test")
	}
	return paths, reasons
}

func (index *QueryIndex) testPlanSubject(selector string) (string, bool) {
	if position, err := index.resolveSymbol(selector); err == nil {
		node := index.nodes[position]
		return node.symbol.Visible, node.declaration.Key.Kind == semanticsnapshot.DeclarationFunction ||
			node.declaration.Key.Kind == semanticsnapshot.DeclarationForeignFunction
	}
	separator := strings.LastIndex(selector, "::")
	if separator < 0 {
		return "", false
	}
	key := semanticsnapshot.DeclarationKey{
		PackageInstance: index.header.PackageInstance,
		Module:          selector[:separator], Name: selector[separator+2:], Kind: semanticsnapshot.DeclarationFunction,
	}
	position, found := index.byKey[key]
	if !found {
		key.Kind = semanticsnapshot.DeclarationForeignFunction
		position, found = index.byKey[key]
	}
	if !found {
		return "", false
	}
	return index.nodes[position].symbol.Visible, true
}

func (index *QueryIndex) testPlanOmissions() []string {
	reasons := []string{}
	add := func(reason string) {
		if !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	callsExact := false
	for _, record := range index.completeness {
		if record.Relation == "calls" {
			callsExact = record.Status == semanticsnapshot.CompletenessExact && record.Unknown == 0 && record.External == 0
		}
		if record.Relation == "generic-template-bodies" {
			add("generic-call-coverage")
		}
	}
	if !callsExact {
		add("incomplete-call-coverage")
	}
	for _, call := range index.relations {
		switch {
		case call.Kind == RelationCallsForeign:
			add("external-ffi")
		case call.Kind != RelationCalls:
			continue
		case call.Target != nil && call.Target.PackageInstance != index.header.PackageInstance:
			add("cross-package-call")
		case !index.ordinaryTestPlanCall(call):
			add("unsupported-call-edge")
		}
	}
	slices.Sort(reasons)
	return reasons
}

func (index *QueryIndex) ordinaryTestPlanCall(call QueryRelation) bool {
	return call.Kind == RelationCalls && call.Target != nil && call.Target.Symbol != "" &&
		call.Target.PackageInstance == index.header.PackageInstance && call.SourceSpecialization == "" &&
		call.SourceBody == "" && call.Target.Specialization == "" && call.Span != nil &&
		call.Certainty == semanticsnapshot.CertaintyExact && call.Provenance == semanticsnapshot.ProvenanceCompilerObserved
}

func testPlanLabel(key semanticsnapshot.DeclarationKey) string {
	return key.Module + "::" + key.Name
}
