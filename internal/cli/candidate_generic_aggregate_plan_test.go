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

const aggregatePlanRoot = `module root;import maker as m;
fn i32 main(){let box=m::make<i32>(42);let small=m::make<u8>(3);let pair=m::Pair<i32,u8>{.left=2,.right=3};discard pair;return box.payload.value;}`
const aggregatePlanMaker = `module maker;import helper as h;
pub struct Cell<T>{T value;}
pub struct Pair<T,U>{T left;U right;}
pub resource Holder<T>{Cell<T> payload;cleanup{discard h::finish<T>(this.payload.value);}}
pub fn Holder<T> make<T>(T value){return Holder<T>{.payload=Cell<T>{.value=value}};}
pub struct Unused<T>{T value;}
fn i32 private_value(){return 1;}`
const aggregatePlanHelper = `module helper;pub fn T finish<T>(T value){return value;}`

func TestCandidateGenericAggregatePlan(t *testing.T) {
	wire := candidateGenericAggregatePlanOracle(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_plan.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_plan;", "module main;", 1) + "\n@effects(console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "project_source_plan", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, wire)
		if code != 0 || out != "" {
			t.Fatalf("aggregate source-plan oracle: %d %q", code, out)
		}
	}, "module_public_templates", "project_source_plan", "generic_aggregate_types")
}

func candidateGenericAggregatePlanOracle(t *testing.T) []byte {
	return candidateGenericAggregateOracle(t, false)
}

func candidateGenericAggregateOracle(t *testing.T, materialized bool) []byte {
	t.Helper()
	type item struct {
		name, root, maker, helper string
		reverse, reject           bool
	}
	cases := []item{
		{name: "base"},
		{name: "cleanup helper edit", helper: strings.Replace(aggregatePlanHelper, "return value;", "let T copied=value;return copied;", 1)},
		{name: "reordered", reverse: true},
		{name: "private edit", maker: strings.Replace(aggregatePlanMaker, "return 1;", "return 2;", 1)},
		{name: "unused template edit", maker: strings.Replace(aggregatePlanMaker, "pub struct Unused<T>{T value;}", "pub struct Unused<T>{T other;}", 1)},
		{name: "private aggregate", maker: strings.Replace(aggregatePlanMaker, "pub struct Pair", "struct Pair", 1), reject: true},
		{name: "wrong arity", root: strings.Replace(aggregatePlanRoot, "Pair<i32,u8>", "Pair<i32>", 1), reject: true},
		{name: "recursive field", maker: strings.Replace(aggregatePlanMaker, "pub struct Cell<T>{T value;}", "pub struct Cell<T>{Cell<T> value;}", 1), reject: true},
		{name: "nonscalar argument", root: strings.Replace(aggregatePlanRoot, "Pair<i32,u8>", "Pair<str,u8>", 1), reject: true},
		{name: "implicit arguments", root: strings.Replace(aggregatePlanRoot, "let pair=", "let m::Pair pair=", 1), reject: true},
		{name: "duplicate binder", maker: strings.Replace(aggregatePlanMaker, "Pair<T,U>", "Pair<T,T>", 1), reject: true},
		{name: "generated name", maker: aggregatePlanMaker + "fn i32 _fy_generic_hidden(){return 0;}", reject: true},
	}
	if materialized {
		for _, scalar := range []string{"bool", "i8", "i16", "i64", "isize", "u16", "u32", "u64", "usize", "f32", "f64"} {
			cases = append(cases, item{name: "scalar " + scalar, root: strings.ReplaceAll(aggregatePlanRoot, "i32", scalar)})
		}
		cases = append(cases, item{name: "local annotation", root: strings.Replace(aggregatePlanRoot, "let box=", "let m::Holder<i32> box=", 1)}, item{name: "ordered tuple", root: strings.Replace(aggregatePlanRoot, "Pair<i32,u8>", "Pair<u8,i32>", 1)})
		cases = append(cases, item{name: "structural fields", maker: strings.Replace(aggregatePlanMaker, "pub struct Pair<T,U>{T left;U right;}", "pub struct Pair<T,U>{Option<(T)> left;Result<U,T> right;T[2] values;ptr T raw;ref Cell<T> borrowed;ptr mut Cell<T> nested;Cell<T>[2] cells;}", 1)})
	}
	var wire bytes.Buffer
	wire.WriteString("FYGAGG01")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(value string) { number(len(value)); wire.WriteString(value) }
	number(len(cases))
	var base genericsource.PackageExpansion
	for c, tc := range cases {
		texts := []string{tc.root, tc.maker, tc.helper}
		for i, fallback := range []string{aggregatePlanRoot, aggregatePlanMaker, aggregatePlanHelper} {
			if texts[i] == "" {
				texts[i] = fallback
			}
		}
		order := []int{0, 1, 2}
		if tc.reverse {
			slices.Reverse(order)
		}
		inputs := make([]genericsource.PackageModule, 3)
		byPath := map[string]int{}
		for i, original := range order {
			file := []string{"root.fy", "maker.fy", "helper.fy"}[original]
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(file, []byte(texts[original])), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("aggregate plan parse %s: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
			}
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(original+1), 64))
			inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}}
			byPath[parsed.AST.Path] = i
		}
		inputs[byPath["root"]].Imports["maker"] = inputs[byPath["maker"]].Owner
		inputs[byPath["maker"]].Imports["helper"] = inputs[byPath["helper"]].Owner
		expanded, err := genericsource.PreparePackages(inputs, genericplan.DefaultPlanLimits())
		if (err != nil) != tc.reject {
			t.Fatalf("aggregate plan oracle %s: reject=%v error=%v", tc.name, tc.reject, err)
		}
		field(tc.name)
		number(3)
		for _, original := range order {
			number(original + 1)
			field(texts[original])
			switch original {
			case 0:
				number(1)
				field("m")
				number(byPath["maker"])
			case 1:
				number(1)
				field("h")
				number(byPath["helper"])
			default:
				number(0)
			}
		}
		if tc.reject {
			number(1)
			continue
		}
		number(0)
		expectedRoots := 3
		if tc.name == "local annotation" {
			expectedRoots = 4
		}
		if len(expanded.Plan.Roots) != expectedRoots || len(expanded.Plan.Specializations) != 9 {
			t.Fatalf("aggregate plan coverage %s: roots=%d implementations=%d", tc.name, len(expanded.Plan.Roots), len(expanded.Plan.Specializations))
		}
		templates := slices.DeleteFunc(slices.Clone(expanded.Templates), func(p genericsource.PublicTemplate) bool { return p.Module != "maker" })
		slices.SortFunc(templates, func(a, b genericsource.PublicTemplate) int { return strings.Compare(a.Name, b.Name) })
		if len(templates) != 5 {
			t.Fatal("aggregate exported template coverage", len(templates))
		}
		if c == 0 {
			base = expanded
		} else if c == 1 {
			if slices.Equal(base.Plan.Roots, expanded.Plan.Roots) {
				t.Fatal("cleanup helper edit lost aggregate dependency")
			}
		} else if c < 5 && !slices.Equal(base.Plan.Roots, expanded.Plan.Roots) {
			t.Fatal("unrelated source/order edit changed concrete identity", tc.name)
		}
		number(len(expanded.Plan.Roots))
		for _, root := range expanded.Plan.Roots {
			field(string(root))
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
				field(b.Parameter)
				field(string(b.Type))
			}
			number(len(spec.Dependencies))
			for _, dependency := range spec.Dependencies {
				field(string(dependency))
			}
		}
		number(len(templates))
		for _, p := range templates {
			field(p.Name)
			field(p.Closure.String())
		}
		if materialized {
			candidateGenericAggregateTypeFacts(t, expanded, number, field)
		}
	}
	return wire.Bytes()
}
