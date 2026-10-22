package cbackendfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertNestedResourceDrops(t *testing.T, report Report) {
	t.Helper()
	const name = "nested-resource-drop"
	contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "c-emission", "cases", "pass", name, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range report.Emitted {
		if fixture.ID != "pass.c-emission."+name {
			continue
		}
		// Distinct child types make a reversed sibling order observable, even
		// though both fields inherit their owner's source binding span.
		expected := []struct{ resource, declaration, binding string }{
			{"main::Box", "let Box owner", "owner"},
			{"main::TemporaryLeaf", "let TemporaryLeaf temporary", "temporary"},
			{"main::LastLeaf", "let Box owner", "owner"},
			{"main::FirstLeaf", "let Box owner", "owner"},
		}
		if len(fixture.DropTrace) != len(expected) {
			t.Fatalf("nested drops = %+v, want four", fixture.DropTrace)
		}
		for index, want := range expected {
			declaration := strings.Index(string(contents), want.declaration)
			if declaration < 0 {
				t.Fatalf("missing binding declaration %q", want.declaration)
			}
			start := declaration + len(want.declaration) - len(want.binding)
			event := fixture.DropTrace[index]
			if event.ResourceType.String() != want.resource || event.Binding.Source != "main.fy" ||
				event.Binding.StartByte != start || event.Binding.EndByte != start+len(want.binding) {
				t.Fatalf("nested drop %d = %+v, want %s at %d-%d", index, event, want.resource, start, start+len(want.binding))
			}
		}
		return
	}
	t.Fatal("nested-resource cleanup fixture missing")
}
