package genericsource

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

const groupedIdentitySource = `module ops;
@interface(Bundle) pub fn U zebra<U>(U value){}
@interface(Bundle) pub fn T alpha<T>(T value){}
@implements(zebra,i64) fn i64 zebra64(i64 value){return value;}
@implements(alpha,i64) fn i64 alpha64(i64 value){return value;}
@implements(zebra,i32) fn i32 zebra32(i32 value){return value;}
@implements(alpha,i32) fn i32 alpha32(i32 value){return value;}
pub fn T wrap<T>(T value){return alpha<T>(value);}
fn i32 main(){return wrap<i32>(42);}
fn i32 unrelated(){return 11;}`

func groupedIdentityExpansion(t *testing.T, text string) (*expander, *genericplan.Catalog, genericplan.Plan, []PublicTemplate) {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("ops.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v %+v", err, parsed.Tree)
	}
	before := staticJSON(t, parsed.AST)
	x := &expander{module: parsed.AST, owner: "owner", templates: map[string]ast.Function{}, names: map[string]bool{}, allowPublic: true}
	var catalog *genericplan.Catalog
	_, plan, err := expandAll([]*expander{x}, genericplan.DefaultPlanLimits(), func(c *genericplan.Catalog) error { catalog = c; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if before != staticJSON(t, parsed.AST) {
		t.Fatal("expansion changed original written AST")
	}
	templates, err := publicTemplates([]*expander{x}, catalog)
	if err != nil {
		t.Fatal(err)
	}
	return x, catalog, plan, templates
}

func staticJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestGroupedStaticInterfaceIdentityRecipe(t *testing.T) {
	x, catalog, _, _ := groupedIdentityExpansion(t, groupedIdentitySource)
	f := x.module.Functions
	// Independently enumerate the required flat AST order; source order is
	// deliberately neither operation-name nor scalar-name order.
	expected := genericplan.DigestText("source-static-operation-group-v1", staticJSON(t, []ast.Function{f[1], f[5], f[3], f[0], f[4], f[2]}))
	if expected.IsZero() || x.operations["alpha"].definition != expected || x.operations["zebra"].definition != expected {
		t.Fatal("group definition is not the exact flat, sorted original AST array")
	}
	if x.operations["alpha"].capability == x.operations["zebra"].capability {
		t.Fatal("group members lost distinct capability IDs")
	}
	for _, name := range []string{"alpha", "zebra"} {
		op := x.operations[name]
		if op.capability != genericplan.CapabilityID("owner::ops::"+name+"::operation") {
			t.Fatal("capability ID changed")
		}
		cap, ok := catalog.Capability(op.capability)
		if !ok || cap.Definition != expected {
			t.Fatal("capability does not bind shared group definition")
		}
		// Package binding remains the existing package AST domain plus group digest.
		body := staticJSON(t, x.templates[name])
		packageDefinition := genericplan.DigestText("source-package-function-v0", body+"\nnull")
		bound := genericplan.DigestText("source-static-bound-operation-v1", packageDefinition.String()+expected.String())
		declaration, ok := catalog.Declaration(x.id(name))
		if !ok || declaration.Definition != bound {
			t.Fatalf("bound operation recipe changed: %s", name)
		}
	}
	// Select both distinct capabilities with their existing per-provider domain.
	requests := []genericplan.Request{{Declaration: x.id("alpha"), Arguments: []genericplan.TypeID{"i32"}}, {Declaration: x.id("zebra"), Arguments: []genericplan.TypeID{"i32"}}}
	plan, err := catalog.Plan(requests, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range plan.Specializations {
		name := strings.TrimPrefix(string(s.Declaration), "owner::ops::")
		provider := x.operations[name].providers["i32"]
		want := genericplan.DigestText("source-static-provider-v1", staticJSON(t, provider))
		if len(s.Constraints) != 1 || s.Constraints[0].Provider != genericplan.ProviderID("owner::ops::"+provider.Name) || s.Constraints[0].ProviderDigest != want || s.Constraints[0].Capability != x.operations[name].capability {
			t.Fatalf("provider identity changed: %+v", s)
		}
	}
}

func TestGroupedStaticInterfaceUnusedSiblingClosure(t *testing.T) {
	baseX, baseCatalog, basePlan, baseTemplates := groupedIdentityExpansion(t, groupedIdentitySource)
	for _, tc := range []struct{ name, old, new string }{
		{"unused sibling provider", "zebra32(i32 value){return value;}", "zebra32(i32 value){return 00042;}"},
		{"unused scalar tuple", "zebra64(i64 value){return value;}", "zebra64(i64 value){return 00042;}"},
		{"unused sibling declaration", "fn U zebra<U>(U value)", "fn V zebra<V>(V value)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(groupedIdentitySource, tc.old, tc.new, 1)
			if text == groupedIdentitySource || len(text) != len(groupedIdentitySource) {
				t.Fatal("closure edit must be nontrivial and equal width")
			}
			x, catalog, plan, templates := groupedIdentityExpansion(t, text)
			// Neither selected declaration/provider AST nor downstream spans changed.
			if !reflect.DeepEqual(baseX.templates["alpha"], x.templates["alpha"]) || !reflect.DeepEqual(baseX.operations["alpha"].providers["i32"], x.operations["alpha"].providers["i32"]) || !reflect.DeepEqual(baseX.templates["wrap"], x.templates["wrap"]) {
				t.Fatal("incidental selected AST/span change invalidates closure proof")
			}
			for _, name := range []string{"alpha", "zebra"} {
				old, _ := baseCatalog.Declaration(baseX.id(name))
				next, _ := catalog.Declaration(x.id(name))
				if baseX.operations[name].definition == x.operations[name].definition || old.Definition == next.Definition {
					t.Fatalf("sibling closure did not reach %s", name)
				}
			}
			if len(baseTemplates) != len(templates) {
				t.Fatal("public surface changed")
			}
			for i := range templates {
				if baseTemplates[i].Name != templates[i].Name || baseTemplates[i].Closure == templates[i].Closure {
					t.Fatalf("public closure failed to invalidate: %s", templates[i].Name)
				}
			}
			if len(basePlan.Specializations) != len(plan.Specializations) {
				t.Fatal("selected specialization set changed")
			}
			for i := range plan.Specializations {
				if basePlan.Specializations[i].Declaration != plan.Specializations[i].Declaration || basePlan.Specializations[i].ID == plan.Specializations[i].ID {
					t.Fatal("selected operation/dependent specialization did not invalidate")
				}
			}
		})
	}
	_, _, privatePlan, privateTemplates := groupedIdentityExpansion(t, strings.Replace(groupedIdentitySource, "return 11;", "return 12;", 1))
	if !reflect.DeepEqual(basePlan.Specializations, privatePlan.Specializations) || !reflect.DeepEqual(baseTemplates, privateTemplates) {
		t.Fatal("unrelated equal-width private edit invalidated selected identities")
	}
}

func TestGroupedStaticInterfaceLabelAdmission(t *testing.T) {
	for _, label := range []string{"a", "_", "A_09", strings.Repeat("x", 64)} {
		if !validStaticGroupLabel(label) {
			t.Fatalf("valid label rejected: %q", label)
		}
	}
	for _, label := range []string{"", "9a", "a-b", "a b", "é", "a한", strings.Repeat("x", 65), "a\x00"} {
		if validStaticGroupLabel(label) {
			t.Fatalf("invalid label admitted: %q", label)
		}
	}
}

func TestGroupedStaticInterfaceModuleScope(t *testing.T) {
	var inputs []PackageModule
	for _, name := range []string{"left", "right"} {
		text := "module " + name + ";@interface(Same) pub fn T operation<T>(T value){}"
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(name+".fy", []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			t.Fatalf("parse: %v %+v", err, parsed.Tree)
		}
		inputs = append(inputs, PackageModule{Module: parsed.AST, Owner: "owner"})
	}
	before := staticJSON(t, inputs)
	result, err := PreparePackages(inputs, genericplan.DefaultPlanLimits())
	var located *SourceError
	if !errors.As(err, &located) || located.Module != "left" || located.Span != inputs[0].Module.Functions[0].Span || result.Modules != nil || len(result.Plan.Specializations) != 0 {
		t.Fatalf("same labels in different modules formed a group or lost source location: %v %+v", err, result)
	}
	if before != staticJSON(t, inputs) {
		t.Fatal("module-scope rejection mutated original AST")
	}
}

func TestStaticInterfaceLegacyIdentityBytes(t *testing.T) {
	text := `module ops;
@interface pub fn T alpha<T>(T value){}
@implements(alpha,i64) fn i64 alpha64(i64 value){return value;}
@implements(alpha,i32) fn i32 alpha32(i32 value){return value;}
fn i32 main(){return alpha<i32>(42);}`
	x, catalog, plan, _ := groupedIdentityExpansion(t, text)
	f := x.module.Functions
	want := genericplan.DigestText("source-static-operation-v1", staticJSON(t, []ast.Function{f[0], f[2], f[1]}))
	if x.operations["alpha"].definition != want {
		t.Fatal("legacy operation bytes/domain changed")
	}
	packageDefinition := genericplan.DigestText("source-package-function-v0", staticJSON(t, f[0])+"\nnull")
	bound := genericplan.DigestText("source-static-bound-operation-v1", packageDefinition.String()+want.String())
	declaration, _ := catalog.Declaration(x.id("alpha"))
	provider := genericplan.DigestText("source-static-provider-v1", staticJSON(t, f[2]))
	if declaration.Definition != bound || len(plan.Specializations) != 1 || len(plan.Specializations[0].Constraints) != 1 || plan.Specializations[0].Constraints[0].ProviderDigest != provider {
		t.Fatal("legacy binding/provider bytes changed")
	}
	// Exercise the original private-module path as well as package publication.
	private := strings.Replace(text, "pub fn", "fn", 1)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("ops.fy", []byte(private)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatal(err)
	}
	before := staticJSON(t, parsed.AST)
	_, p, err := Expand(parsed.AST, "owner", genericplan.DefaultPlanLimits())
	if err != nil || len(p.Specializations) != 1 {
		t.Fatalf("legacy private path: %v %+v", err, p)
	}
	if before != staticJSON(t, parsed.AST) {
		t.Fatal("legacy expansion mutated source")
	}
	for _, receipt := range []struct{ got, want string }{
		{want.String(), "fa1b8df1e84c65c7c3668a2ba5e6f53123c61e998a1d9c207cd9b03b630fd1f3"},
		{bound.String(), "3d243be26ca9a43b79a0ab039e5cd3bd6b73e9a06870924e32cc4050b2a2681e"},
		{provider.String(), "cf2a185654f52ca7ca137c36f6776b9b92fab591fd131ce2cba7491881083b84"},
		{string(plan.Specializations[0].ID), "g0_d9e6a44c9ba21ef02111d8814310d5c3720efbeb3812c48844fd5cfd56491605"},
		{string(p.Specializations[0].ID), "g0_def8e8b0d61dd115db769c717fea47feb8dceb1d5dc80c11c378ca1d9b31634b"},
	} {
		if receipt.got != receipt.want {
			t.Fatalf("frozen legacy identity changed: %s want %s", receipt.got, receipt.want)
		}
	}
	t.Logf("legacy operation=%s bound=%s provider=%s specialization=%s private=%s", want, bound, provider, plan.Specializations[0].ID, p.Specializations[0].ID)
}
