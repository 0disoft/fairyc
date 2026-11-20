package atlasmap

import (
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
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
// program and packageID must come from the verified compilation used by index.
func (index *QueryIndex) TestPlan(selector string, tests []string, program mir.Program, packageID hir.PackageID) ([]TestPlanPath, []string) {
	subject, found := index.testPlanSubject(selector)
	paths := []TestPlanPath{}
	if !found {
		return paths, []string{"symbol-not-found"}
	}
	cleanups := index.testPlanCleanups(program, packageID)
	closures := index.testPlanClosures(program, packageID)
	reasons := index.testPlanOmissions(cleanups.recovered + closures.recovered)
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
		for _, pos := range index.incoming[target] {
			if call := index.relations[pos]; index.ordinaryTestPlanCall(call) {
				enqueue(call.Source)
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
			if position, found := index.byVisible[at]; found {
				path = append(path, testPlanLabel(index.nodes[position].declaration.Key))
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

func (index *QueryIndex) testPlanOmissions(recoveredCalls int) []string {
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
