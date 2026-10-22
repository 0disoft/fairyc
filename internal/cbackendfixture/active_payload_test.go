package cbackendfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These expectations come from the authored source, not from the interpreter's
// trace. Interpreter/native agreement on an omitted/wrong drop is not enough.
func assertActivePayloadDrops(t *testing.T, report Report) {
	t.Helper()
	expected := map[string]struct{ resource, binding string }{
		"resource-option-some":          {"main::Token", "token"},
		"resource-option-none":          {},
		"resource-result-ok":            {"main::Token", "token"},
		"resource-result-err":           {"main::Token", "token"},
		"resource-result-propagate-ok":  {"main::Token", "token"},
		"resource-result-propagate-err": {"main::Token", "token"},
		"resource-option-drop":          {"main::Token", "present"},
		"resource-result-drop-ok":       {"main::Success", "wrapped"},
		"resource-result-drop-err":      {"main::Failure", "wrapped"},
	}
	for _, fixture := range report.Emitted {
		name := strings.TrimPrefix(fixture.ID, "pass.c-emission.")
		want, found := expected[name]
		if !found {
			continue
		}
		delete(expected, name)
		t.Run(name, func(t *testing.T) {
			if want.resource == "" {
				if len(fixture.DropTrace) != 0 {
					t.Fatalf("inactive payload produced drops: %+v", fixture.DropTrace)
				}
				return
			}
			if len(fixture.DropTrace) != 1 {
				t.Fatalf("active resource drop count = %d, want one: %+v", len(fixture.DropTrace), fixture.DropTrace)
			}
			contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "c-emission", "cases", "pass", name, "src", "main.fy"))
			if err != nil {
				t.Fatal(err)
			}
			start := strings.Index(string(contents), want.binding)
			if start < 0 {
				t.Fatal("expected binding absent from source")
			}
			event := fixture.DropTrace[0]
			if event.ResourceType.String() != want.resource || event.Binding.Source != "main.fy" ||
				event.Binding.StartByte != start || event.Binding.EndByte != start+len(want.binding) {
				t.Fatalf("drop = %+v, want %s at %s:%d-%d", event, want.resource, "main.fy", start, start+len(want.binding))
			}
		})
	}
	if len(expected) != 0 {
		t.Errorf("active-payload cases missing from emission evidence: %v", expected)
	}
}
