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

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const aggregateSignatureMaker = aggregatePlanMaker + `
pub fn Holder<i32> make_i32(i32 value){return make<i32>(value);}
pub fn Pair<i32,u8> pair(Pair<i32,u8> value){return value;}
pub fn i32 read(ref Holder<i32> box){return box.payload.value;}
`

func TestCandidateGenericAggregateSignatures(t *testing.T) {
	wire := candidateAggregateSignatureOracle(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_signatures.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_signatures;", "module main;", 1)
	main += "\n@effects(console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "module_signatures", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, wire); code != 0 || out != "" {
			t.Fatalf("aggregate signatures: %d %q", code, out)
		}
	}, "project_source_plan")
}

func candidateAggregateSignatureOracle(t *testing.T) []byte {
	t.Helper()
	type fixture struct {
		name, extra string
		reverse     bool
		reject      uint64
	}
	cases := []fixture{
		{name: "ordinary aggregate signatures"},
		{name: "owner and module reorder", reverse: true},
		{name: "ordered second instance", extra: `pub fn Pair<u8,i32> swapped(Pair<u8,i32> value){return value;}`},
		{name: "extended scalar", extra: `pub fn Cell<f64> decimal(Cell<f64> value){return value;}`},
		{name: "wrappers", extra: `pub fn Result<Cell<i32>,i32> wrapped(Result<Cell<i32>,i32> value){return value;}`},
		{name: "borrow", extra: `pub fn i32 cell_read(ref Cell<i32> value){return value.value;}`},
		{name: "must use struct", extra: `@must_use pub fn Cell<i32> used(Cell<i32> value){return value;}`},
		{name: "must use void", extra: `@must_use fn void invalid(){}`, reject: 4201},
		{name: "foreign generic aggregate", extra: `@link_name("invalid") unsafe extern C fn Cell<i32> invalid();`, reject: 5102},
		{name: "generic owner method", extra: `fn i32 Cell::invalid(ref Cell<i32> this){return this.value;}`, reject: 4501},
	}
	var wire bytes.Buffer
	wire.WriteString("FYGSGN01")
	number := func(n uint64) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(uint64(len(s))); wire.WriteString(s) }
	number(uint64(len(cases)))
	for _, tc := range cases {
		texts := []string{strings.Replace(aggregatePlanRoot, "return box.payload.value;", "return m::read(ref box);", 1), aggregateSignatureMaker + tc.extra, aggregatePlanHelper}
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
			parsed, err := frontend.AnalyzeSyntax(source.NewFile([]string{"root.fy", "maker.fy", "helper.fy"}[original], []byte(texts[original])), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("%s parse: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
			}
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(original+1), 64))
			inputs = append(inputs, genericsource.PackageModule{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}})
			semanticInput.Files = append(semanticInput.Files, semantic.File{Package: hir.PackageID(original + 1), Module: parsed.AST, Path: []string{"root.fy", "maker.fy", "helper.fy"}[original]})
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
			t.Fatalf("%s Go analysis: %v", tc.name, err)
		}
		checked := analysis.Analysis.Semantic
		if rejected := len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 || checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone; rejected != (tc.reject != 0) {
			t.Fatalf("%s Go admission: reject=%v diagnostics=%+v unsupported=%+v phase=%s", tc.name, tc.reject, checked.Diagnostics, checked.Unsupported, checked.CompletedThrough)
		}
		expanded, err := genericsource.PreparePackages(inputs, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("%s prepare: %v", tc.name, err)
		}
		number(tc.reject)
		if tc.reject != 0 {
			continue
		}
		names, render := candidateAggregateTypeOracle(t, expanded)
		count := 0
		for _, m := range expanded.Modules {
			for _, f := range m.Functions {
				if _, generated := names[f.Name]; !generated {
					count++
				}
			}
		}
		number(uint64(count))
		for m, module := range expanded.Modules {
			for _, f := range module.Functions {
				if _, generated := names[f.Name]; generated {
					continue
				}
				number(uint64(m))
				field(f.Name)
				field(render(f.ResultType))
				number(uint64(len(f.Parameters)))
				for _, p := range f.Parameters {
					field(render(p.Type))
				}
			}
		}
	}
	return wire.Bytes()
}
