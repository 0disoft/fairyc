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

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const aggregateLayoutMaker = `module maker;import helper as h;
pub struct Cell<T>{T value;}
pub struct Pair<T,U>{T left;U right;}
pub resource Holder<T>{Cell<T> payload;cleanup{discard h::finish<T>(this.payload.value);}}
pub fn Holder<i32> hold(Holder<i32> value){return value;}
pub fn Pair<i32,u8> pair(Pair<i32,u8> value){return value;}
pub struct Unused<T>{T value;}
`

func TestCandidateGenericAggregateLayouts(t *testing.T) {
	wire := candidateAggregateLayoutOracle(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_layouts.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_layouts;", "module main;", 1) + "\n@effects(console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "module_unit_interface_declarations", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, wire); code != 0 || out != "" {
			t.Fatalf("aggregate field/layout contracts: %d %q", code, out)
		}
	}, "project_source_plan")
}

func candidateAggregateLayoutOracle(t *testing.T) []byte {
	t.Helper()
	type fixture struct {
		name, maker string
		reverse     bool
		reject      uint64
		layoutOnly  bool
	}
	cases := []fixture{
		{name: "nested scalar struct and resource"},
		{name: "coordinate and module reorder", reverse: true},
		{name: "structural wrappers", maker: strings.Replace(aggregateLayoutMaker, "T value;", "T value;Option<T> optional;Result<T,u8> result;", 1)},
		{name: "scalar and record arrays", maker: strings.Replace(aggregateLayoutMaker, "T left;U right;", "T left;U right;T[2] values;Cell<T>[2] records;", 1)},
		{name: "ordinary pointer cycle", maker: strings.Replace(aggregateLayoutMaker, "pub struct Cell<T>{T value;}", "pub struct Node{ptr Node next;}pub struct Cell<T>{T value;Node link;}", 1)},
		{name: "repr C scalar record", maker: strings.Replace(aggregateLayoutMaker, "pub struct Cell<T>", "@repr(C) pub struct Cell<T>", 1)},
		{name: "zero field resource", maker: strings.Replace(aggregateLayoutMaker, "pub resource Holder<T>{Cell<T> payload;cleanup{discard h::finish<T>(this.payload.value);}}", "pub resource Holder<T>{cleanup{}}", 1)},
		{name: "wrapped aggregate field", maker: strings.Replace(aggregateLayoutMaker, "T left;U right;", "T left;U right;Option<Cell<T>> nested;", 1)},
		{name: "extended scalar array", maker: aggregateLayoutMaker + `pub struct Values<T>{T[2] elements;} pub fn Values<f64> values(Values<f64> value){return value;}`, reject: 2},
		{name: "second scalar resource", maker: aggregateLayoutMaker + `pub fn Holder<f64> extra(Holder<f64> value){return value;}`},
		{name: "copied owning field", maker: strings.Replace(aggregateLayoutMaker, "pub struct Cell<T>{T value;}", "pub resource Leaf{i32 value;cleanup{}}pub struct Cell<T>{T value;Leaf hidden;}", 1), reject: 1002},
		{name: "borrowed field", maker: strings.Replace(aggregateLayoutMaker, "T value;", "T value;ref T borrowed;", 1), reject: 1003},
		{name: "duplicate field", maker: strings.Replace(aggregateLayoutMaker, "T value;", "T value;T value;", 1), reject: 102},
		{name: "ordinary recursive layout", maker: strings.Replace(aggregateLayoutMaker, "pub struct Cell<T>{T value;}", "pub struct Loop{Loop next;}pub struct Cell<T>{T value;Loop hidden;}", 1), reject: 9, layoutOnly: true},
	}
	var wire bytes.Buffer
	wire.WriteString("FYGFLD01")
	number := func(n uint64) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(uint64(len(s))); wire.WriteString(s) }
	number(uint64(len(cases)))
	for _, tc := range cases {
		maker := tc.maker
		if maker == "" {
			maker = aggregateLayoutMaker
		}
		texts := []string{"module root;import maker as m;fn i32 main(){return 0;}", maker, aggregatePlanHelper}
		order := []int{0, 1, 2}
		if tc.reverse {
			slices.Reverse(order)
		}
		var inputs []genericsource.PackageModule
		var semanticInput semantic.PackageInput
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
		byPath := map[string]int{}
		field(tc.name)
		number(3)
		for i, original := range order {
			file := []string{"root.fy", "maker.fy", "helper.fy"}[original]
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(file, []byte(texts[original])), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("%s syntax: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
			}
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(original+1), 64))
			inputs = append(inputs, genericsource.PackageModule{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}})
			semanticInput.Files = append(semanticInput.Files, semantic.File{Package: hir.PackageID(original + 1), Module: parsed.AST, Path: file})
			coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(fmt.Sprint(original+1), 64))
			if err != nil {
				t.Fatal(err)
			}
			coordinates[hir.PackageID(original+1)] = coordinate
			byPath[parsed.AST.Path] = i
			number(uint64(original + 1))
			field(texts[original])
		}
		inputs[byPath["root"]].Imports["maker"] = inputs[byPath["maker"]].Owner
		inputs[byPath["maker"]].Imports["helper"] = inputs[byPath["helper"]].Owner
		semanticInput.Imports = []semantic.PackageImport{{FromPackage: 1, ImportPath: "maker", ToPackage: 2}, {FromPackage: 2, ImportPath: "helper", ToPackage: 3}}
		analysis, err := compiler.AnalyzeGenericPackages(semanticInput, coordinates, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("%s analysis: %v", tc.name, err)
		}
		// Go Flow does not attest finite by-value layouts; the recursive fixture
		// independently requires the Fairy layout contract to reject that cycle.
		checked := analysis.Analysis.Semantic
		if reject := len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 || checked.FailedPhase != semantic.PhaseNone || checked.CompletedThrough != semantic.PhaseFlow; reject != (tc.reject != 0 && !tc.layoutOnly) {
			t.Fatalf("%s Go layout rejection=%v: %+v %+v phase=%s", tc.name, tc.reject, checked.Diagnostics, checked.Unsupported, checked.CompletedThrough)
		}
		expanded, err := genericsource.PreparePackages(inputs, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("%s expansion: %v", tc.name, err)
		}
		number(tc.reject)
		if tc.reject != 0 {
			continue
		}
		names, render := candidateAggregateTypeOracle(t, expanded)
		type shape struct {
			name                    string
			module                  int
			spanStart, spanEnd      int
			fields                  []ast.FieldDeclaration
			resource, repr, cleanup bool
		}
		var shapes []shape
		repr := func(attrs []ast.Attribute) bool {
			for _, a := range attrs {
				if a.Name == "repr" {
					return true
				}
			}
			return false
		}
		for m, module := range expanded.Modules {
			for _, s := range module.Structs {
				shapes = append(shapes, shape{s.Name, m, s.Span.StartByte, s.Span.EndByte, s.Fields, false, repr(s.Attributes), false})
			}
			for _, r := range module.Resources {
				shapes = append(shapes, shape{r.Name, m, r.Span.StartByte, r.Span.EndByte, r.Fields, true, false, len(r.Cleanup.Body.Statements) != 0 || r.Cleanup.Body.Tail != nil})
			}
		}
		number(uint64(len(shapes)))
		for _, s := range shapes {
			name := s.name
			if id, ok := names[name]; ok {
				name = id
			}
			field(name)
			number(uint64(s.module))
			number(uint64(s.spanStart))
			number(uint64(s.spanEnd))
			for _, b := range []bool{s.resource, s.repr, s.cleanup} {
				if b {
					number(1)
				} else {
					number(0)
				}
			}
			number(uint64(len(s.fields)))
			for _, f := range s.fields {
				field(f.Name)
				number(uint64(f.Span.StartByte))
				number(uint64(f.Span.EndByte))
				field(render(f.Type))
			}
		}
	}
	return wire.Bytes()
}
