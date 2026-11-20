package atlasmap

import (
	"errors"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const maxTestPlanWorkspaceCalls = 65536

// WorkspaceTestPlan finds call and cleanup paths across exact-current
// partitions. The package IDs and MIR belong to the compilation that produced
// the partitions.
func WorkspaceTestPlan(partitions []Partition, expectations map[string]LoadExpectation, testInstance, subjectInstance, selector string, tests []string, result compiler.Result, packageIDs map[string]hir.PackageID) ([]TestPlanPath, []string) {
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
	case len(subjects) == 0:
		if cleanup {
			return empty, []string{"cleanup-call-coverage"}
		}
		return empty, []string{"generic-call-coverage"}
	}
	roots := make([]workspaceCallEndpoint, 0, min(len(subjects), maxTestPlanSymbols+1))
	for _, subject := range subjects[:cap(roots)] {
		symbol, specialization, _ := strings.Cut(subject, "#")
		roots = append(roots, workspaceCallEndpoint{WorkspaceSymbol{subjectInstance, symbol}, specialization})
	}
	cleanups := workspaceTestPlanCleanups(result, packages, packageIDs)
	reasons := workspaceTestPlanCoverage(packages, cleanups)
	parents, truncated := walkTestPlan(roots, func(target workspaceCallEndpoint, enqueue func(workspaceCallEndpoint)) {
		positions := ledger.incoming[target.symbol]
		if target.specialization != "" {
			positions = ledger.incomingSpecializations[target]
		}
		for _, position := range positions {
			call := ledger.calls[position]
			if call.TargetSpecialization == target.specialization {
				enqueue(workspaceCallEndpoint{call.Source, call.SourceSpecialization})
			}
		}
		for _, source := range cleanups.incoming[target] {
			enqueue(source)
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
		endpoint := testPlanEndpoint(at.symbol.Symbol, at.specialization)
		label, found := pkg.testPlanEndpointLabel(endpoint)
		if !found {
			label = endpoint
		}
		if instance != testInstance {
			return pkg.header.PackageUID + "::" + label
		}
		return label
	})
}

func workspaceTestPlanCleanups(result compiler.Result, packages map[string]*QueryIndex, packageIDs map[string]hir.PackageID) testPlanCleanupGraph[workspaceCallEndpoint] {
	bindings := testPlanCleanupBindings[workspaceCallEndpoint]{
		functions: make(map[string]workspaceCallEndpoint), cleanups: make(map[hir.Type]workspaceCallEndpoint),
	}
	for instance, pkg := range packages {
		packageID, found := packageIDs[instance]
		if !found || !result.Verified {
			return testPlanCleanupGraph[workspaceCallEndpoint]{}
		}
		bindings.addPackage(pkg, packageID, func(symbol, specialization string) workspaceCallEndpoint {
			return workspaceCallEndpoint{WorkspaceSymbol{instance, symbol}, specialization}
		})
	}
	if bindings.expectedCleanups == 0 {
		return testPlanCleanupGraph[workspaceCallEndpoint]{complete: true}
	}
	return buildTestPlanCleanupGraph(result, bindings)
}

func workspaceTestPlanCoverage(packages map[string]*QueryIndex, cleanups testPlanCleanupGraph[workspaceCallEndpoint]) []string {
	reasons := []string{}
	add := func(reason string) {
		if !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	if !cleanups.complete {
		add("cleanup-call-coverage")
	}
	for _, pkg := range packages {
		for _, record := range pkg.completeness {
			if record.Relation != "calls" {
				continue
			}
			exact := record.Status == semanticsnapshot.CompletenessExact && record.Unknown == 0
			// Each partition carries the same workspace-wide completeness counts.
			exact = exact || cleanups.complete && cleanups.recovered > 0 && record.Unknown == cleanups.recovered &&
				slices.Equal(record.Reasons, []string{ordinaryCleanupOmission})
			if !exact || record.External != 0 {
				add("incomplete-call-coverage")
			}
			if record.External != 0 {
				add("external-ffi")
			}
		}
	}
	slices.Sort(reasons)
	return reasons
}
