//go:build ignore

// This test is overlaid into internal/genericsource only by its Python harness.
// It reads the real private source adapter and immutable planner catalog.
package genericsource

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCandidateSourceDependencyOracle(t *testing.T) {
	root := os.Getenv("FAIRYC_SOURCE_DEPENDENCY_ORACLE_ROOT")
	repo := os.Getenv("FAIRYC_SOURCE_DEPENDENCY_REPO")
	if root == "" || repo == "" {
		t.Fatal("explicit fixture paths required")
	}
	vec, err := os.ReadFile(filepath.Join(repo, "internal/sdkstdlib/sources/optional/std/vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	rawRoot := []byte("module root;import helper as a;import helper as a0;import std::vec as vec;pub fn T api<T>(T value){let T saved=local<T>(value);let i32 first=a::id<i32>(1);let i32 again=a0::id<i32>(first);let vec::Vec<a::Token> tokens=vec::empty<a::Token>();return a::id<T>(saved);}fn T local<T>(T value){return value;}pub fn T loop<T>(T value){return loop<T>(value);}")
	rawHelper := []byte("module helper;pub struct Token{i32 value;}pub fn U id<U>(U value){return pair<U,i64>(secret<U>(value),1);}fn A pair<A,B>(A a,B b){return a;}fn T secret<T>(T value){return value;}pub fn T flow<T>(T value){var T x=value;if(true){x=id<T>(x);}switch(.some(true)){case .some(flag):{if(flag){x=id<T>(x);}}case .none:{}};while(false){x=id<T>(x);}for(var i32 i=0;i<1;i+=1){x=id<T>(x);}unsafe{x=id<T>(x);}region zone in arena{x=id<T>(x);}defer ordinary(1);return x;}fn void ordinary(i32 n){}pub fn T views<T>(ref u8[] data,T value){for(let u8 byte in data){discard id<T>(value);}for(let u8 other in producer<T>(value)){discard id<T>(value);}return value;}fn ref u8[] producer<T>(T value){return b\"\";}")
	type testCase struct {
		name   string
		raw    [][]byte
		digits []int
	}
	cases := []testCase{
		{"Base", [][]byte{rawRoot, rawHelper, vec}, []int{1, 2, 3}},
		{"Edited", [][]byte{rawRoot, bytes.ReplaceAll(rawHelper, []byte("fn T secret<T>(T value){return value;}"), []byte("fn T secret<T>(T value){return pair<T,i32>(value,2);}")), vec}, []int{1, 2, 3}},
		{"Reordered", [][]byte{vec, rawRoot, rawHelper}, []int{3, 1, 2}},
		{"InertView", [][]byte{rawRoot, bytes.ReplaceAll(rawHelper, []byte("return b\"\";"), []byte("return b\"changed\";")), vec}, []int{1, 2, 3}},
	}
	var wire bytes.Buffer
	wire.WriteString("FYGDEP01")
	number := func(n int) {
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(v []byte) { number(len(v)); wire.Write(v) }
	number(len(cases))
	var summary []map[string]any
	for _, tc := range cases {
		field([]byte(tc.name))
		number(len(tc.raw))
		modules := make([]*ast.Module, len(tc.raw))
		owners := make([]genericplan.PackageID, len(tc.raw))
		inputs := make([]PackageModule, len(tc.raw))
		byPath := map[string]int{}
		for i, raw := range tc.raw {
			path := "root.fy"
			if bytes.HasPrefix(raw, []byte("module helper;")) {
				path = "helper.fy"
			} else if bytes.HasPrefix(raw, []byte("module std::vec;")) {
				path = "std/vec.fy"
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, raw), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("%s parse %s: %v %v", tc.name, path, err, parsed.Tree.Diagnostics)
			}
			modules[i] = parsed.AST
			owners[i] = genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(tc.digits[i]), 64))
			byPath[parsed.AST.Path] = i
			inputs[i] = PackageModule{Module: parsed.AST, Owner: owners[i], Imports: map[string]genericplan.PackageID{}}
		}
		for i, m := range modules {
			number(tc.digits[i])
			field(tc.raw[i])
			number(len(m.Imports))
			for _, imported := range m.Imports {
				target, exists := byPath[imported.Path]
				if !exists {
					t.Fatal("missing fixture provider")
				}
				inputs[i].Imports[imported.Path] = owners[target]
				field([]byte(imported.Alias))
				number(target)
			}
		}
		sdkOwner := owners[byPath["std::vec"]]
		prepared, err := PrepareSDKStoragePackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{sdkOwner})
		if err != nil {
			t.Fatalf("%s actual package preparation: %v", tc.name, err)
		}
		// The private observer captures exactly the catalog used before expansion.
		expanders := make([]*expander, len(inputs))
		for i, input := range inputs {
			allowed := input.Owner == sdkOwner
			expanders[i] = &expander{module: input.Module, owner: input.Owner, templates: map[string]ast.Function{}, names: map[string]bool{}, imports: map[string]*expander{}, allowPublic: true, allowNominal: true, packageNominal: true, allowStorage: allowed, allowStorageTransfers: allowed, scalarTemplatesOnly: !allowed}
		}
		for i, m := range modules {
			for _, imported := range m.Imports {
				expanders[i].imports[imported.Alias] = expanders[byPath[imported.Path]]
			}
			if err := expanders[i].activateSDKVectorLoops(); err != nil {
				t.Fatal(err)
			}
		}
		if err := bindNominalContext(expanders); err != nil {
			t.Fatal(err)
		}
		var actual *genericplan.Catalog
		_, _, err = expandAll(expanders, genericplan.DefaultPlanLimits(), func(c *genericplan.Catalog) error { actual = c; return nil })
		if err != nil {
			t.Fatal(err)
		}
		number(len(prepared.Declarations))
		totalEdges := 0
		for _, origin := range prepared.Declarations {
			declaration, exists := actual.Declaration(origin.ID)
			if !exists {
				t.Fatal("missing canonical declaration")
			}
			encoded, err := json.Marshal(declaration)
			if err != nil {
				t.Fatal(err)
			}
			digest, err := templateClosure(actual, origin.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range prepared.Templates {
				if p.Owner == origin.Owner && p.Module == origin.Module && p.Name == origin.Name && p.Closure != digest {
					t.Fatal("observer closure differs from real package preparation")
				}
			}
			field([]byte(origin.ID))
			field(encoded)
			field([]byte(digest.String()))
			totalEdges += len(declaration.Dependencies)
		}
		summary = append(summary, map[string]any{"case": tc.name, "templates": len(prepared.Declarations), "dependencies": totalEdges, "publicTemplates": len(prepared.Templates)})
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	bad, err := frontend.AnalyzeSyntax(source.NewFile("bad.fy", []byte("module bad;pub fn _fy_generic_hidden api<T>(T value){return value;}")), source.DefaultLimits())
	if err != nil || bad.AST == nil || !bad.Tree.ErrorFree() {
		t.Fatal("generated-type rejection fixture did not parse")
	}
	if _, err := PreparePackages([]PackageModule{{Module: bad.AST, Owner: "bad"}}, genericplan.DefaultPlanLimits()); err == nil || !strings.Contains(err.Error(), "generated aggregate") {
		t.Fatalf("generated type was not rejected by the actual source adapter: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "oracle.bin"), wire.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"cases": summary, "bytes": wire.Len(), "fixture": filepath.Join(root, "oracle.bin"), "source": "real expandAll catalog.Declaration JSON and templateClosure; checked against PrepareSDKStoragePackages", "generatedTypeRejected": true}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "oracle.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d source graphs, %d exact fixture bytes", len(cases), wire.Len())
}
