// Produce caller-resolved identity facts from the real Go planner and SDK Vec.
// bun tools/run-go.ts run tools/verify-candidate-specialization.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type argument struct {
	ID, Owner, Definition string
	Abilities             []genericplan.Ability
}
type operand struct{ ID, Definition string }
type constraint struct {
	ID, Definition               string
	Arguments                    []operand
	Provider, ProviderDefinition string
}
type record struct {
	Name, Expected, Declaration, Owner, Definition string
	Arguments                                      []argument
	Constraints                                    []constraint
	Dependencies                                   []genericplan.SpecializationID
}

func fact(c *genericplan.Catalog, s genericplan.Specialization, name string) (record, error) {
	d, ok := c.Declaration(s.Declaration)
	if !ok {
		return record{}, fmt.Errorf("missing declaration")
	}
	r := record{Name: name, Expected: string(s.ID), Declaration: string(d.ID), Owner: string(d.Owner), Definition: d.Definition.String(), Dependencies: s.Dependencies}
	for _, b := range s.Bindings {
		t, ok := c.Type(b.Type)
		if !ok {
			return r, fmt.Errorf("missing type")
		}
		r.Arguments = append(r.Arguments, argument{string(t.ID), string(t.Owner), t.Definition.String(), t.Abilities})
	}
	for _, v := range s.Constraints {
		cap, ok := c.Capability(v.Capability)
		if !ok {
			return r, fmt.Errorf("missing capability")
		}
		x := constraint{ID: string(v.Capability), Definition: cap.Definition.String(), Provider: string(v.Provider), ProviderDefinition: v.ProviderDigest.String()}
		for _, id := range v.Arguments {
			t, ok := c.Type(id)
			if !ok {
				return r, fmt.Errorf("missing constraint operand")
			}
			x.Arguments = append(x.Arguments, operand{string(id), t.Definition.String()})
		}
		r.Constraints = append(r.Constraints, x)
	}
	return r, nil
}
func synthetic(mode int) (record, error) {
	b, e := genericplan.NewBuilder(genericplan.DefaultCatalogLimits())
	if e != nil {
		return record{}, e
	}
	for _, name := range []string{"i32", "i64"} {
		def := genericplan.DigestText("source-scalar-v0", name)
		if mode == 6 && name == "i32" {
			def = genericplan.DigestText("source-scalar-v0", "changed-i32")
		}
		abilities := []genericplan.Ability{genericplan.AbilityScalar}
		if mode == 7 {
			abilities = []genericplan.Ability{genericplan.AbilityCopy, genericplan.AbilityInert, genericplan.AbilityStore}
		}
		if e = b.AddType(genericplan.TypeDescriptor{ID: genericplan.TypeID(name), Owner: "builtin", Definition: def, Abilities: abilities}); e != nil {
			return record{}, e
		}
	}
	for _, name := range []string{"A", "B"} {
		def := genericplan.DigestText("capability", name)
		if mode == 4 && name == "A" {
			def = genericplan.DigestText("capability", "changed-A")
		}
		cap := genericplan.CapabilityID("app::" + name)
		if e = b.AddCapability(genericplan.CapabilityDescriptor{ID: cap, Owner: "app", Arity: 1, Definition: def}); e != nil {
			return record{}, e
		}
		provider := genericplan.ProviderID("app::provide_" + strings.ToLower(name))
		body := genericplan.DigestText("provider", name)
		if mode == 3 && name == "A" {
			body = genericplan.DigestText("provider", "changed-A")
		}
		for _, t := range []genericplan.TypeID{"i32", "i64"} {
			if e = b.AddImplementation(genericplan.Implementation{Capability: cap, Arguments: []genericplan.TypeID{t}, Owner: "app", Provider: provider, ProviderDigest: body}); e != nil {
				return record{}, e
			}
		}
	}
	parameter := []genericplan.TypeParameter{{Name: "T"}}
	for _, name := range []string{"leaf_a", "leaf_b"} {
		def := genericplan.DigestText("leaf", name)
		if mode == 5 && name == "leaf_a" {
			def = genericplan.DigestText("leaf", "changed")
		}
		if e = b.AddDeclaration(genericplan.Declaration{ID: genericplan.DeclarationID("app::" + name), Owner: "app", Definition: def, Parameters: parameter}); e != nil {
			return record{}, e
		}
	}
	definition := genericplan.DigestText("root", "body")
	if mode == 2 {
		definition = genericplan.DigestText("root", "changed")
	}
	d := genericplan.Declaration{ID: "app::root", Owner: "app", Definition: definition, Parameters: parameter, Constraints: []genericplan.CapabilityConstraint{{Capability: "app::B", Arguments: []genericplan.TypeExpression{genericplan.ParameterType("T")}}, {Capability: "app::A", Arguments: []genericplan.TypeExpression{genericplan.ParameterType("T")}}}, Dependencies: []genericplan.Dependency{{Declaration: "app::leaf_b", Arguments: []genericplan.TypeExpression{genericplan.ParameterType("T")}}, {Declaration: "app::leaf_a", Arguments: []genericplan.TypeExpression{genericplan.ParameterType("T")}}}}
	if mode == 8 {
		d.Constraints = append(d.Constraints, genericplan.CapabilityConstraint{Capability: "app::A", Arguments: []genericplan.TypeExpression{genericplan.ConcreteType("i32")}})
	}
	if mode == 9 {
		d.Parameters = append(d.Parameters, genericplan.TypeParameter{Name: "U"})
	}
	if e = b.AddDeclaration(d); e != nil {
		return record{}, e
	}
	c, e := b.Build()
	if e != nil {
		return record{}, e
	}
	t := genericplan.TypeID("i32")
	if mode == 1 {
		t = "i64"
	}
	args := []genericplan.TypeID{t}
	if mode == 9 {
		args = append(args, "i64")
	}
	plan, e := c.Plan([]genericplan.Request{{Declaration: d.ID, Arguments: args}}, genericplan.DefaultPlanLimits())
	if e != nil {
		return record{}, e
	}
	for _, s := range plan.Specializations {
		if s.Declaration == d.ID {
			return fact(c, s, fmt.Sprintf("planner-%d", mode))
		}
	}
	return record{}, fmt.Errorf("root missing from plan")
}
func vector(element string) (record, error) {
	dir := filepath.Join(".cache", "generic-specialization-20261013", "vec-"+element)
	if e := os.MkdirAll(filepath.Join(dir, "src"), 0755); e != nil {
		return record{}, e
	}
	manifest := "schema='fairyc.manifest/v1'\n[package]\nuid='example.identity.vec'\nname='vec-identity'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
	if e := os.WriteFile(filepath.Join(dir, "fairy.toml"), []byte(manifest), 0644); e != nil {
		return record{}, e
	}
	source := "module main;import std::vec as v;fn i32 main(){let value=v::empty<" + element + ">();return 0;}\n"
	if e := os.WriteFile(filepath.Join(dir, "src", "main.fy"), []byte(source), 0644); e != nil {
		return record{}, e
	}
	w, e := workspaceloader.Load(dir)
	if e != nil {
		return record{}, e
	}
	input, e := w.SemanticInput()
	if e != nil {
		return record{}, e
	}
	coordinates, e := w.PackageCoordinates()
	if e != nil {
		return record{}, e
	}
	packages, e := w.SemanticPackages()
	if e != nil {
		return record{}, e
	}
	var owners []hir.PackageID
	owner := ""
	for _, p := range packages {
		if p.SourceKind == workspaceloader.SourceKindSDK {
			owners = append(owners, p.ID)
			owner = "package_coordinate_v1_" + strings.TrimPrefix(p.Coordinate.String(), "fypc1-sha256-")
		}
	}
	checked, e := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	if e != nil {
		return record{}, e
	}
	s := checked.Analysis.Semantic
	if s.CompletedThrough != semantic.PhaseFlow || s.FailedPhase != semantic.PhaseNone || len(s.Diagnostics) != 0 || len(s.Unsupported) != 0 {
		return record{}, fmt.Errorf("Vec semantic analysis did not complete: %+v", s.Diagnostics)
	}
	declaration := owner + "::std::vec::Vec"
	var resource *ast.Resource
	for _, f := range input.Files {
		if f.Module.Path == "std::vec" {
			for i := range f.Module.Resources {
				if f.Module.Resources[i].Name == "Vec" {
					resource = &f.Module.Resources[i]
				}
			}
		}
	}
	if resource == nil {
		return record{}, fmt.Errorf("SDK Vec resource absent")
	}
	encoded, e := json.Marshal(struct {
		Struct   *ast.Struct
		Resource *ast.Resource
	}{Resource: resource})
	if e != nil {
		return record{}, e
	}
	definition := genericplan.DigestText("source-package-aggregate-v0", string(encoded)+"\nnull")
	// Dependencies come from the actual admitted plan; hash publication is tested
	// separately by the Fairy encoder, rather than granting this helper admission.
	for _, sp := range checked.Plan.Specializations {
		if string(sp.Declaration) == declaration && len(sp.Bindings) == 1 && string(sp.Bindings[0].Type) == element {
			t := genericplan.TypeDescriptor{ID: genericplan.TypeID(element), Owner: "builtin", Abilities: []genericplan.Ability{"copy", "drop", "inert", "scalar", "store"}, Definition: genericplan.DigestText("source-scalar-v0", element)}
			return record{Name: "sdk-Vec<" + element + ">", Expected: string(sp.ID), Declaration: declaration, Owner: owner, Definition: definition.String(), Arguments: []argument{{string(t.ID), string(t.Owner), t.Definition.String(), t.Abilities}}, Dependencies: sp.Dependencies}, nil
		}
	}
	return record{}, fmt.Errorf("SDK Vec specialization absent")
}

func verifyRecord(r record) error {
	var out bytes.Buffer
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(s string) { number(len(s)); out.WriteString(s) }
	digest := func(s string) {
		d, e := hex.DecodeString(s)
		if e != nil || len(d) != 32 {
			panic("malformed oracle digest")
		}
		out.Write(d)
	}
	field("fairyc.genericplan.specialization.v0")
	field(r.Declaration)
	field(r.Owner)
	digest(r.Definition)
	number(len(r.Arguments))
	for _, a := range r.Arguments {
		field(a.ID)
		field(a.Owner)
		number(len(a.Abilities))
		for _, v := range a.Abilities {
			field(string(v))
		}
		digest(a.Definition)
	}
	number(len(r.Constraints))
	for _, c := range r.Constraints {
		field(c.ID)
		digest(c.Definition)
		number(len(c.Arguments))
		for _, a := range c.Arguments {
			field(a.ID)
			digest(a.Definition)
		}
		field(c.Provider)
		digest(c.ProviderDefinition)
	}
	number(len(r.Dependencies))
	for _, d := range r.Dependencies {
		field(string(d))
	}
	h := sha256.Sum256(out.Bytes())
	actual := "g0_" + hex.EncodeToString(h[:])
	if actual != r.Expected {
		return fmt.Errorf("oracle facts mismatch %s: %s != %s", r.Name, actual, r.Expected)
	}
	return nil
}

func run() error {
	var records []record
	for i := 0; i < 10; i++ {
		r, e := synthetic(i)
		if e != nil {
			return e
		}
		records = append(records, r)
	}
	for _, element := range []string{"i32", "i64"} {
		r, e := vector(element)
		if e != nil {
			return e
		}
		records = append(records, r)
	}
	for _, r := range records {
		if e := verifyRecord(r); e != nil {
			return e
		}
	}
	var out bytes.Buffer
	out.WriteString("FYGSPEC1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(s string) { number(len(s)); out.WriteString(s) }
	number(len(records))
	for _, r := range records {
		field(r.Expected)
		field(r.Declaration)
		field(r.Owner)
		field(r.Definition)
		number(len(r.Arguments))
		for _, a := range r.Arguments {
			field(a.ID)
			field(a.Owner)
			field(a.Definition)
			number(len(a.Abilities))
			for _, ability := range a.Abilities {
				field(string(ability))
			}
		}
		number(len(r.Constraints))
		for _, c := range r.Constraints {
			field(c.ID)
			field(c.Definition)
			number(len(c.Arguments))
			for _, a := range c.Arguments {
				field(a.ID)
				field(a.Definition)
			}
			field(c.Provider)
			field(c.ProviderDefinition)
		}
		number(len(r.Dependencies))
		for _, d := range r.Dependencies {
			field(string(d))
		}
	}
	dir := filepath.Join(".cache", "generic-specialization-20261013")
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	if e := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "records": records, "bytes": out.Len()})
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
