package packagegraph

import (
	"slices"
	"testing"
)

func TestResolveDeduplicatesIdenticalDependencyInstances(t *testing.T) {
	t.Parallel()
	shared := Identity{PackageUID: "shared", SourceIdentity: "source", VersionIdentity: "1", Features: []string{"base"}}
	result, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "app", Role: RoleRoot, Identity: Identity{PackageUID: "app", SourceIdentity: "workspace", VersionIdentity: "workspace"}, Modules: []Module{{Path: "main", Imports: []string{"shared"}}}},
			{ID: "shared-left", Role: RoleDependency, Identity: shared, Modules: []Module{{Path: "shared"}}},
			{ID: "shared-right", Role: RoleDependency, Identity: shared, Modules: []Module{{Path: "shared"}}},
		},
		Dependencies: []Dependency{
			{Owner: "app", Binding: "left", Target: "shared-left"},
			{Owner: "app", Binding: "right", Target: "shared-right"},
		},
	})
	if err != nil || rejection != nil {
		t.Fatalf("Resolve() error = %v, rejection = %+v", err, rejection)
	}
	if len(result.InstanceClasses) != 2 || len(result.Lookups) != 1 || !slices.Equal(result.Lookups[0].ProviderMembers, []string{"shared-left", "shared-right"}) {
		t.Fatalf("Resolve() = %+v", result)
	}
	sharedID, err := ComputeInstanceID(shared)
	if err != nil {
		t.Fatalf("ComputeInstanceID() error = %v", err)
	}
	if result.InstanceClasses[1].ID != sharedID {
		t.Fatalf("shared instance id = %s, want %s", result.InstanceClasses[1].ID, sharedID)
	}
}

func TestResolveRejectsAmbiguousDirectProviders(t *testing.T) {
	t.Parallel()
	result, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main", Imports: []string{"shared"}}}},
			{ID: "left", Role: RoleDependency, Identity: testIdentity("left"), Modules: []Module{{Path: "shared"}}},
			{ID: "right", Role: RoleDependency, Identity: testIdentity("right"), Modules: []Module{{Path: "shared"}}},
		},
		Dependencies: []Dependency{{Owner: "app", Binding: "left", Target: "left"}, {Owner: "app", Binding: "right", Target: "right"}},
	})
	if err != nil || len(result.Lookups) != 0 || rejection == nil || rejection.Reason != ReasonAmbiguousVisibleModule || !slices.Equal(rejection.Subjects, []string{"left", "right"}) {
		t.Fatalf("Resolve() = result %+v, rejection %+v, error %v", result, rejection, err)
	}
}

func TestResolveHidesTransitiveModules(t *testing.T) {
	t.Parallel()
	_, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main", Imports: []string{"leaf"}}}},
			{ID: "middle", Role: RoleDependency, Identity: testIdentity("middle"), Modules: []Module{{Path: "middle", Imports: []string{"leaf"}}}},
			{ID: "leaf", Role: RoleDependency, Identity: testIdentity("leaf"), Modules: []Module{{Path: "leaf"}}},
		},
		Dependencies: []Dependency{{Owner: "app", Binding: "middle", Target: "middle"}, {Owner: "middle", Binding: "leaf", Target: "leaf"}},
	})
	if err != nil || rejection == nil || rejection.Reason != ReasonMissingVisibleModule || rejection.ImportPath != "leaf" || !slices.Equal(rejection.Subjects, []string{"app", "leaf"}) {
		t.Fatalf("Resolve() rejection = %+v, error = %v", rejection, err)
	}
}

func TestResolveRejectsDuplicateBindingBeforeImports(t *testing.T) {
	t.Parallel()
	_, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "main"}}},
			{ID: "left", Role: RoleDependency, Identity: testIdentity("left"), Modules: []Module{{Path: "left"}}},
			{ID: "right", Role: RoleDependency, Identity: testIdentity("right"), Modules: []Module{{Path: "right"}}},
		},
		Dependencies: []Dependency{{Owner: "app", Binding: "dep", Target: "left"}, {Owner: "app", Binding: "dep", Target: "right"}},
	})
	if err != nil || rejection == nil || rejection.Reason != ReasonDuplicateBinding || !slices.Equal(rejection.Subjects, []string{"app", "left", "right"}) {
		t.Fatalf("Resolve() rejection = %+v, error = %v", rejection, err)
	}
}

func TestResolveRejectsDependencyCycleBeforeImports(t *testing.T) {
	t.Parallel()
	_, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "app", Role: RoleRoot, Identity: testIdentity("app"), Modules: []Module{{Path: "app", Imports: []string{"missing"}}}},
			{ID: "middle", Role: RoleDependency, Identity: testIdentity("middle"), Modules: []Module{{Path: "middle"}}},
			{ID: "leaf", Role: RoleDependency, Identity: testIdentity("leaf"), Modules: []Module{{Path: "leaf"}}},
		},
		Dependencies: []Dependency{
			{Owner: "app", Binding: "middle", Target: "middle"},
			{Owner: "middle", Binding: "leaf", Target: "leaf"},
			{Owner: "leaf", Binding: "app", Target: "app"},
		},
	})
	if err != nil || rejection == nil || rejection.Reason != ReasonDependencyCycle || !slices.Equal(rejection.Subjects, []string{"app", "middle", "leaf"}) || rejection.ImportPath != "" {
		t.Fatalf("Resolve() rejection = %+v, error = %v", rejection, err)
	}
}

func TestResolveRejectsSelfDependencyAfterInstanceDeduplication(t *testing.T) {
	t.Parallel()
	shared := testIdentity("shared")
	_, rejection, err := Resolve(Graph{
		Nodes: []Node{
			{ID: "shared-left", Role: RoleRoot, Identity: shared, Modules: []Module{{Path: "shared"}}},
			{ID: "shared-right", Role: RoleDependency, Identity: shared, Modules: []Module{{Path: "shared"}}},
		},
		Dependencies: []Dependency{{Owner: "shared-left", Binding: "self", Target: "shared-right"}},
	})
	if err != nil || rejection == nil || rejection.Reason != ReasonDependencyCycle || !slices.Equal(rejection.Subjects, []string{"shared-left", "shared-right"}) {
		t.Fatalf("Resolve() rejection = %+v, error = %v", rejection, err)
	}
}

func testIdentity(name string) Identity {
	return Identity{PackageUID: name, SourceIdentity: "source-" + name, VersionIdentity: "1"}
}
