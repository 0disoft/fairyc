package atlasmap

import (
	"errors"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const maxTestPlanWorkspaceCalls = 65536

// WorkspaceTestPlan finds direct call paths across exact-current partitions
// while keeping test discovery and execution in one package instance.
func WorkspaceTestPlan(partitions []Partition, expectations map[string]LoadExpectation, testInstance, subjectInstance, selector string, tests []string) ([]TestPlanPath, []string) {
	packages := make(map[string]*QueryIndex)
	ledger, err := joinWorkspaceCalls(partitions, expectations, WorkspaceCallBudget{
		MaxPackages: maxTestPlanSymbols, MaxCalls: maxTestPlanWorkspaceCalls,
	}, LogicalSchemaV3, packages)
	empty := []TestPlanPath{}
	if errors.Is(err, ErrWorkspaceJoinBudget) {
		return empty, []string{"workspace-call-limit"}
	}
	if err != nil || packages[testInstance] == nil || packages[subjectInstance] == nil {
		return empty, []string{"atlas-unavailable"}
	}
	selector, cleanup := strings.CutSuffix(selector, testPlanCleanupSuffix)
	subjects, found := packages[subjectInstance].testPlanSubjects(selector, cleanup)
	switch {
	case !found:
		return empty, []string{"symbol-not-found"}
	case cleanup:
		return empty, []string{"cleanup-call-coverage"}
	case len(subjects) == 0:
		return empty, []string{"generic-call-coverage"}
	}
	roots := make([]workspaceCallEndpoint, 0, min(len(subjects), maxTestPlanSymbols+1))
	for _, subject := range subjects[:cap(roots)] {
		symbol, specialization, _ := strings.Cut(subject, "#")
		roots = append(roots, workspaceCallEndpoint{WorkspaceSymbol{subjectInstance, symbol}, specialization})
	}
	reasons := workspaceTestPlanCoverage(packages)
	parents, truncated := walkTestPlan(roots, func(target workspaceCallEndpoint, enqueue func(workspaceCallEndpoint)) {
		positions := ledger.incoming[target.symbol]
		if target.specialization != "" {
			positions = ledger.incomingSpecializations[target]
		}
		for _, position := range positions {
			call := ledger.calls[position]
			if call.SourceBody == "" && call.TargetSpecialization == target.specialization {
				enqueue(workspaceCallEndpoint{call.Source, call.SourceSpecialization})
			}
		}
	})
	if truncated {
		reasons = append(reasons, "traversal-limit")
	}
	testSymbols := make(map[string]workspaceCallEndpoint)
	for test, symbol := range packages[testInstance].testPlanTestSymbols() {
		testSymbols[test] = workspaceCallEndpoint{symbol: WorkspaceSymbol{testInstance, symbol}}
	}
	return testPlanPaths(tests, testSymbols, parents, reasons, func(at workspaceCallEndpoint) string {
		instance := at.symbol.PackageInstance
		pkg := packages[instance]
		label, _ := pkg.testPlanEndpointLabel(testPlanEndpoint(at.symbol.Symbol, at.specialization))
		if instance != testInstance {
			return pkg.header.PackageUID + "::" + label
		}
		return label
	})
}

func workspaceTestPlanCoverage(packages map[string]*QueryIndex) []string {
	reasons := []string{}
	add := func(reason string) {
		if !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	for _, pkg := range packages {
		for _, record := range pkg.completeness {
			if record.Relation != "calls" {
				continue
			}
			if record.Status != semanticsnapshot.CompletenessExact || record.Unknown != 0 || record.External != 0 {
				add("incomplete-call-coverage")
			}
			if record.External != 0 {
				add("external-ffi")
			}
		}
		for _, node := range pkg.nodes {
			if node.declaration.Key.Kind == semanticsnapshot.DeclarationResource && node.declaration.DropKind == string(hir.ResourceDropNonEmpty) {
				add("cleanup-call-coverage")
				break
			}
		}
	}
	slices.Sort(reasons)
	return reasons
}
