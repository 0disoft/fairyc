package atlasmap

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestWorkspaceCallIndexJoinsQuickstartCallsBothDirections(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	budget := WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 16}
	index, err := NewWorkspaceCallIndex(partitions, expectations, budget)
	if err != nil {
		t.Fatal(err)
	}
	if usage := index.Usage(); usage != (WorkspaceCallUsage{Packages: 2, Calls: 1, CrossPackageCalls: 1}) {
		t.Fatalf("Usage() = %+v", usage)
	}

	var rootInstance, dependencyInstance string
	for _, partition := range partitions {
		header := partition.Header()
		if len(header.Dependencies) == 1 {
			rootInstance = header.PackageInstance
			dependencyInstance = header.Dependencies[0]
		}
	}
	if rootInstance == "" || dependencyInstance == "" {
		t.Fatal("quickstart partitions do not expose one root dependency")
	}
	rootSymbol := workspaceTestSymbol(t, index, rootInstance, "main", "main")
	dependencySymbol := workspaceTestSymbol(t, index, dependencyInstance, "answer", "value")
	outgoing, err := index.OutgoingCalls(rootInstance, rootSymbol.Visible)
	if err != nil {
		t.Fatal(err)
	}
	incoming, err := index.IncomingCalls(dependencyInstance, dependencySymbol.Full)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoing) != 1 || len(incoming) != 1 || !reflect.DeepEqual(outgoing, incoming) {
		t.Fatalf("joined calls outgoing=%+v incoming=%+v", outgoing, incoming)
	}
	call := outgoing[0]
	if call.Source != (WorkspaceSymbol{PackageInstance: rootInstance, Symbol: rootSymbol.Visible}) ||
		call.Target != (WorkspaceSymbol{PackageInstance: dependencyInstance, Symbol: dependencySymbol.Visible}) {
		t.Fatalf("joined call = %+v", call)
	}

	outgoing[0].Target.Symbol = "corrupt"
	if fresh, _ := index.OutgoingCalls(rootInstance, rootSymbol.Full); len(fresh) != 1 || fresh[0].Target.Symbol != dependencySymbol.Visible {
		t.Fatalf("caller mutated workspace call index: %+v", fresh)
	}
	reversed := slices.Clone(partitions)
	slices.Reverse(reversed)
	reordered, err := NewWorkspaceCallIndex(reversed, expectations, budget)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(index.Calls(), reordered.Calls()) {
		t.Fatalf("partition order changed joined calls\nfirst:  %+v\nsecond: %+v", index.Calls(), reordered.Calls())
	}
}

func TestWorkspaceCallIndexJoinsLocalCallsWithoutCrossPackageClaims(t *testing.T) {
	t.Parallel()
	fixture := localCallGraphFixture(t)
	instance := fixture.partition.Header().PackageInstance
	index, err := NewWorkspaceCallIndex(
		[]Partition{fixture.partition},
		map[string]LoadExpectation{instance: fixture.expectation},
		WorkspaceCallBudget{MaxPackages: 1, MaxCalls: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if usage := index.Usage(); usage != (WorkspaceCallUsage{Packages: 1, Calls: 2}) {
		t.Fatalf("Usage() = %+v", usage)
	}
	middle := workspaceTestSymbol(t, index, instance, "main", "middle")
	leaf := workspaceTestSymbol(t, index, instance, "main", "leaf")
	outgoing, err := index.OutgoingCalls(instance, middle.Visible)
	if err != nil {
		t.Fatal(err)
	}
	incoming, err := index.IncomingCalls(instance, leaf.Visible)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoing) != 1 || len(incoming) != 1 || outgoing[0].Target.Symbol != leaf.Visible || incoming[0].Source.Symbol != middle.Visible {
		t.Fatalf("local joined calls outgoing=%+v incoming=%+v", outgoing, incoming)
	}
}

func TestWorkspaceCallIndexRejectsIncompleteStaleAndOverBudgetInputs(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	budget := WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 16}
	if _, err := NewWorkspaceCallIndex(nil, nil, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("empty join error = %v", err)
	}
	if _, err := NewWorkspaceCallIndex(partitions, expectations, WorkspaceCallBudget{}); err == nil {
		t.Fatal("invalid workspace join budget succeeded")
	}
	if _, err := NewWorkspaceCallIndex(partitions, expectations, WorkspaceCallBudget{MaxPackages: 1, MaxCalls: 16}); !errors.Is(err, ErrWorkspaceJoinBudget) {
		t.Fatalf("package-budget error = %v", err)
	}
	if _, err := NewWorkspaceCallIndex(partitions, expectations, WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 0}); !errors.Is(err, ErrWorkspaceJoinBudget) {
		t.Fatalf("call-budget error = %v", err)
	}

	var root Partition
	for _, partition := range partitions {
		if len(partition.Header().Dependencies) != 0 {
			root = partition
		}
	}
	rootExpectation := map[string]LoadExpectation{root.Header().PackageInstance: expectations[root.Header().PackageInstance]}
	if _, err := NewWorkspaceCallIndex([]Partition{root}, rootExpectation, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("missing-dependency error = %v", err)
	}

	missingExpectation := make(map[string]LoadExpectation, len(expectations)-1)
	for instance, expectation := range expectations {
		if instance != root.Header().PackageInstance {
			missingExpectation[instance] = expectation
		}
	}
	if _, err := NewWorkspaceCallIndex(partitions, missingExpectation, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("missing-expectation error = %v", err)
	}
	extraExpectation := make(map[string]LoadExpectation, len(expectations)+1)
	for instance, expectation := range expectations {
		extraExpectation[instance] = expectation
	}
	extraExpectation["unknown-package"] = expectations[root.Header().PackageInstance]
	if _, err := NewWorkspaceCallIndex(partitions, extraExpectation, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("extra-expectation error = %v", err)
	}
	duplicate := append(slices.Clone(partitions), partitions[0])
	if _, err := NewWorkspaceCallIndex(duplicate, expectations, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("duplicate-partition error = %v", err)
	}

	stale := make(map[string]LoadExpectation, len(expectations))
	for instance, expectation := range expectations {
		stale[instance] = expectation
	}
	rootCurrent := stale[root.Header().PackageInstance]
	rootCurrent.Current = cloneCurrentPackage(rootCurrent.Current)
	rootCurrent.Current.Files[0].Digest[0] ^= 0xff
	stale[root.Header().PackageInstance] = rootCurrent
	if _, err := NewWorkspaceCallIndex(partitions, stale, budget); !errors.Is(err, ErrStaleMap) {
		t.Fatalf("stale workspace join error = %v", err)
	}

	broken := slices.Clone(partitions)
	for partitionIndex := range broken {
		if broken[partitionIndex].Header().PackageInstance != root.Header().PackageInstance {
			continue
		}
		broken[partitionIndex].data.functions = broken[partitionIndex].Functions()
		for functionIndex := range broken[partitionIndex].data.functions {
			for callIndex := range broken[partitionIndex].data.functions[functionIndex].Calls {
				call := &broken[partitionIndex].data.functions[functionIndex].Calls[callIndex]
				if call.Target.PackageInstance != root.Header().PackageInstance {
					call.Target.Name = "missing"
				}
			}
		}
	}
	if _, err := NewWorkspaceCallIndex(broken, expectations, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("missing-target error = %v", err)
	}
	privateTarget := slices.Clone(partitions)
	for partitionIndex := range privateTarget {
		if privateTarget[partitionIndex].Header().PackageInstance == root.Header().PackageInstance {
			continue
		}
		privateTarget[partitionIndex].data.declarations = privateTarget[partitionIndex].Declarations()
		for declarationIndex := range privateTarget[partitionIndex].data.declarations {
			if privateTarget[partitionIndex].data.declarations[declarationIndex].Key.Name == "value" {
				privateTarget[partitionIndex].data.declarations[declarationIndex].Public = false
			}
		}
	}
	if _, err := NewWorkspaceCallIndex(privateTarget, expectations, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
		t.Fatalf("private-target error = %v", err)
	}

	index, err := NewWorkspaceCallIndex(partitions, expectations, budget)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.OutgoingCalls("unknown-package", "not-a-symbol"); err == nil {
		t.Fatal("unknown package selector succeeded")
	}
	if _, err := index.IncomingCalls(root.Header().PackageInstance, "not-a-symbol"); err == nil {
		t.Fatal("malformed symbol selector succeeded")
	}
}

func workspaceTestSymbol(t *testing.T, index *WorkspaceCallIndex, packageInstance, module, name string) Symbol {
	t.Helper()
	packageIndex := index.packages[packageInstance]
	if packageIndex == nil {
		t.Fatalf("workspace call index has no package %q", packageInstance)
	}
	for _, node := range packageIndex.nodes {
		if node.declaration.Key.Module == module && node.declaration.Key.Name == name {
			return node.symbol
		}
	}
	t.Fatalf("package %q has no symbol %s::%s", packageInstance, module, name)
	return Symbol{}
}
