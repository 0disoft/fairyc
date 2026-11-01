package atlasmap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestImpactCurrentReverseClosureAndBounds(t *testing.T) {
	w, r := loadQuickstart(t)
	build := testCompilerBuild()
	_, p, err := PrepareWorkspace(w, r, build)
	if err != nil {
		t.Fatal(err)
	}
	e, err := CurrentWorkspaceExpectations(w, r, build)
	if err != nil {
		t.Fatal(err)
	}
	i, err := NewWorkspaceCallIndex(p, e, WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 32})
	if err != nil {
		t.Fatal(err)
	}
	var instance string
	for _, p := range p {
		if len(p.Header().Dependencies) == 0 {
			instance = p.Header().PackageInstance
		}
	}
	s := workspaceTestSymbol(t, p, instance, "answer", "value")
	a, err := i.Impact(instance, s.Visible, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	b, err := i.Impact(instance, s.Full, 4, 8)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("noncanonical selector result", err)
	}
	if len(a.Nodes) != 2 || !a.TraversalComplete || a.State == "exact" || len(a.Contexts) != 2 {
		t.Fatalf("impact %+v", a)
	}
	for _, limit := range []struct{ depth, symbols int }{{0, 8}, {4, 1}} {
		c, err := i.Impact(instance, s.Visible, limit.depth, limit.symbols)
		if err != nil || c.TraversalComplete || c.State != "partial" || len(c.Nodes) != 1 {
			t.Fatalf("cutoff %+v %v", c, err)
		}
	}
	encoded, _ := json.Marshal(a)
	if !json.Valid(encoded) {
		t.Fatal("invalid JSON")
	}
	e[instance] = LoadExpectation{}
	if _, err := NewWorkspaceCallIndex(p, e, WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 32}); err == nil {
		t.Fatal("stale map accepted")
	}
}
