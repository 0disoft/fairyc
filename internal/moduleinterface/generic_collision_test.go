package moduleinterface_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func genericCollisionFixture(count int) (moduleinterface.Descriptor, []moduleinterface.GenericTemplate) {
	items := make([]moduleinterface.Item, count)
	templates := make([]moduleinterface.GenericTemplate, count)
	for i := range count {
		items[i] = moduleinterface.Item{Kind: "function", Name: fmt.Sprintf("item_%06d", i), ResultType: "i32"}
		name := fmt.Sprintf("template_%06d", i)
		templates[i] = moduleinterface.GenericTemplate{Module: "util", Name: name, Closure: genericplan.DigestText("collision-bench", name)}
	}
	return moduleinterface.Descriptor{Modules: []moduleinterface.Module{{Path: "util", Items: items}}}, templates
}

func TestGenericConflictLookupPreservesHash(t *testing.T) {
	descriptor, templates := genericCollisionFixture(64)
	got, err := moduleinterface.ComputeGeneric(descriptor, templates)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the pre-index implementation; this identity must not change.
	const want = "fygif0-sha256-50bc24a5bb0c8bb68f03672e21c9a1b0dfea520d94e09baf62940ae03a89b8ae"
	if got.String() != want {
		t.Fatalf("generic interface hash=%s, want=%s", got.String(), want)
	}
	slices.Reverse(templates)
	reordered, err := moduleinterface.ComputeGeneric(descriptor, templates)
	if err != nil || got != reordered || templates[0].Name != "template_000063" || descriptor.Modules[0].Items[0].Name != "item_000000" {
		t.Fatalf("hash, template order or descriptor changed: %v", err)
	}
}

func TestGenericConflictLookupKeepsScopeAndErrorOrder(t *testing.T) {
	digest := genericplan.DigestText("test", "closure")
	template := func(module, name string) moduleinterface.GenericTemplate {
		return moduleinterface.GenericTemplate{Module: module, Name: name, Closure: digest}
	}
	descriptor := moduleinterface.Descriptor{Modules: []moduleinterface.Module{
		{Path: "a", Items: []moduleinterface.Item{
			{Kind: "enum", Name: "z"},
			{Kind: "function", Name: "a"},
			{Kind: "struct", Name: "m"},
		}},
		{Path: "b"},
	}}
	cases := []struct {
		name      string
		templates []moduleinterface.GenericTemplate
		want      string
	}{
		{"empty", nil, ""},
		{"single", []moduleinterface.GenericTemplate{template("a", "b")}, ""},
		{"scoped", []moduleinterface.GenericTemplate{template("a", "b"), template("a", "c"), template("b", "a"), template("b", "m")}, ""},
		{"single-conflict", []moduleinterface.GenericTemplate{template("a", "a")}, "generic template conflicts with ordinary public item"},
		{"function-conflict", []moduleinterface.GenericTemplate{template("a", "b"), template("a", "a")}, "generic template conflicts with ordinary public item"},
		{"struct-conflict", []moduleinterface.GenericTemplate{template("a", "b"), template("a", "m")}, "generic template conflicts with ordinary public item"},
		{"enum-conflict", []moduleinterface.GenericTemplate{template("a", "b"), template("a", "z")}, "generic template conflicts with ordinary public item"},
		{"conflict-before-duplicate", []moduleinterface.GenericTemplate{template("a", "a"), template("a", "a")}, "generic template conflicts with ordinary public item"},
		{"duplicate-before-conflict", []moduleinterface.GenericTemplate{template("a", "b"), template("a", "b"), template("a", "z")}, "duplicate public generic template"},
		{"invalid-before-conflict", []moduleinterface.GenericTemplate{{Module: "a", Name: "a"}, template("a", "b")}, "invalid public generic template"},
		{"absent-module", []moduleinterface.GenericTemplate{template("absent", "a")}, "invalid public generic template"},
	}
	for _, indexed := range []bool{false, true} {
		for _, test := range cases {
			t.Run(fmt.Sprintf("indexed-%t/%s", indexed, test.name), func(t *testing.T) {
				candidate := descriptor
				templates := slices.Clone(test.templates)
				if indexed {
					candidate.Modules = slices.Clone(descriptor.Modules)
					candidate.Modules[0].Items = slices.Clone(candidate.Modules[0].Items)
					for i := range 64 {
						candidate.Modules[0].Items = append(candidate.Modules[0].Items, moduleinterface.Item{Kind: "struct", Name: fmt.Sprintf("ordinary_%03d", i)})
						templates = append(templates, template("a", fmt.Sprintf("zz_%03d", i)))
					}
				}
				_, err := moduleinterface.ComputeGeneric(candidate, templates)
				got := ""
				if err != nil {
					got = err.Error()
				}
				if got != test.want {
					t.Fatalf("error=%q, want=%q", got, test.want)
				}
			})
		}
	}
}

func BenchmarkGenericInterfaceCollisions(b *testing.B) {
	for _, count := range []int{32, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			descriptor, templates := genericCollisionFixture(count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := moduleinterface.ComputeGeneric(descriptor, templates); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
