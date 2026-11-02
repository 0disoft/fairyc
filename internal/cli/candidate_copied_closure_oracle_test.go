package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/source"
)

// The real Go package expander supplies independent root/specialization IDs and
// public closure digests. No Go-expanded AST is passed into the Fairy compiler.
func candidateCopiedClosureOracle(t *testing.T, dir string) string {
	t.Helper()
	root := "module root;import helper as h;fn i32 main(){assert(h::keep<i64>(14)==14);return h::keep<i32>(42);}"
	provider := "module helper;pub fn T keep<T>(T value){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};assert(add(2)==42);return value;}fn i32 unused(){return 1;}"
	var wire bytes.Buffer
	wire.WriteString("FYGPLAN1")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(len(s)); wire.WriteString(s) }
	cases := []struct{ name, provider string }{
		{"base", provider}, {"provider edit", strings.Replace(provider, "offset+x", "offset+x+0", 1)},
		{"unused tuple edit", strings.Replace(provider, "offset=40", "offset=41", 1)}, {"private edit", strings.Replace(provider, "return 1;", "return 2;", 1)},
	}
	number(len(cases))
	var base genericsource.PackageExpansion
	for c, tc := range cases {
		inputs := make([]genericsource.PackageModule, 2)
		for i, raw := range []string{root, tc.provider} {
			result, err := frontend.AnalyzeSyntax(source.NewFile([]string{"root.fy", "helper.fy"}[i], []byte(raw)), source.DefaultLimits())
			if err != nil || result.AST == nil || !result.Tree.ErrorFree() {
				t.Fatalf("oracle parse %s/%d: %v %+v", tc.name, i, err, result.Tree.Diagnostics)
			}
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(i+1), 64))
			inputs[i] = genericsource.PackageModule{Module: result.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}}
		}
		inputs[0].Imports["helper"] = inputs[1].Owner
		expanded, err := genericsource.PreparePackages(inputs, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("oracle %s: %v", tc.name, err)
		}
		if c == 0 {
			base = expanded
		} else if c == 3 {
			if !slices.Equal(base.Plan.Roots, expanded.Plan.Roots) || !slices.Equal(base.Templates, expanded.Templates) {
				t.Fatal("private edit changed dependent identity")
			}
		} else if slices.Equal(base.Plan.Roots, expanded.Plan.Roots) || slices.Equal(base.Templates, expanded.Templates) {
			t.Fatal("provider edit did not change dependent identity")
		}
		field(tc.name)
		number(2)
		for i, raw := range []string{root, tc.provider} {
			number(i + 1)
			field(raw)
			if i == 0 {
				number(1)
				field("h")
				number(1)
			} else {
				number(0)
			}
		}
		number(len(expanded.Plan.Roots))
		for _, id := range expanded.Plan.Roots {
			field(string(id))
		}
		number(len(expanded.Plan.Specializations))
		for _, spec := range expanded.Plan.Specializations {
			field(string(spec.ID))
			field(string(spec.Declaration))
			var args []string
			for _, b := range spec.Bindings {
				args = append(args, string(b.Type))
			}
			field(string(spec.Declaration) + "<" + strings.Join(args, ",") + ">")
			number(len(spec.Bindings))
			for _, b := range spec.Bindings {
				field(string(b.Parameter))
				field(string(b.Type))
			}
			number(len(spec.Dependencies))
			for _, id := range spec.Dependencies {
				field(string(id))
			}
		}
		templates := slices.Clone(expanded.Templates)
		slices.SortFunc(templates, func(a, b genericsource.PublicTemplate) int { return strings.Compare(a.Name, b.Name) })
		number(len(templates))
		for _, v := range templates {
			field(v.Name)
			field(v.Closure.String())
		}
	}
	name := filepath.Join(dir, "copied-closure-oracle.bin")
	if err := os.WriteFile(name, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return name
}
