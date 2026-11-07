package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// These are original source files, never reconstructed declaration/graph tables.
// Semantic acceptance and the narrower current MIR storage boundary are separate.
type closureInterfaceCase struct {
	name                    string
	files                   []string
	semanticOK, interfaceOK bool
	badModule               int
	badText                 string
}

func closureInterfaceSources(t *testing.T) []closureInterfaceCase {
	t.Helper()
	const holder = "module provider; // 제공자 원문\r\npub struct Holder{fn /* leaf */ i32(i32) action;}"
	const prefix = "module prefix;fn i32 unchanged(){return 7;}"
	one := func(name, raw string, sem, inter bool, bad string) closureInterfaceCase {
		return closureInterfaceCase{name, []string{prefix, raw}, sem, inter, 1, bad}
	}
	imported := func(name, typ string) closureInterfaceCase {
		return closureInterfaceCase{name, []string{prefix, "module main; // 호출자 원문\r\nimport provider as p;fn " + typ + " keep(" + typ + " value){return value;}", holder}, false, false, 1, typ}
	}
	cases := []closureInterfaceCase{
		one("direct-typed-identity", "module main; // 원문\r\nfn fn /* result */ i32(i32) keep(fn i32(i32) value){let fn i32(i32) copy=value;return copy;}", true, true, ""),
		one("ordinary-direct-field", "module main;struct Holder{fn i32(i32) action;}fn Holder keep(Holder value){let Holder copy=value;return copy;}", true, true, ""),
		{"imported-plain-copy", []string{prefix, "module main;import provider as p;fn p::Holder keep(p::Holder value){let p::Holder copy=value;return copy;}", holder}, true, true, 0, ""},
		{"distinct-provider-identity", []string{prefix, "module main;import left as a;import right as b;fn a::Holder left(a::Holder value){return value;}fn b::Holder right(b::Holder value){return value;}", "module left;pub struct Holder{fn i32(i32) action;}", "module right;pub struct Holder{fn i32(i32) action;}"}, true, true, 0, ""},
		imported("imported-pointer", "ptr p::Holder"),
		imported("imported-mutable-pointer", "ptr mut p::Holder"),
		imported("imported-reference", "ref p::Holder"),
		imported("imported-mutable-reference", "ref mut p::Holder"),
		imported("imported-option", "Option<p::Holder>"),
		imported("imported-result-success", "Result<p::Holder,u32>"),
		imported("imported-result-failure", "Result<i32,p::Holder>"),
		imported("imported-array", "p::Holder[2]"),
		{"imported-generic-argument", []string{prefix, "module main;import provider as p;struct Box<T>{T value;}fn Box<p::Holder> keep(Box<p::Holder> value){return value;}", holder}, false, false, 1, "Box<p::Holder>"},
		{"imported-generic-call", []string{prefix, "module main;import provider as p;fn T echo<T>(T value){return value;}fn p::Holder keep(p::Holder value){return echo<p::Holder>(value);}", holder}, false, false, 1, "p::Holder"},
		one("resource-direct-field", "module main;resource Stored{fn i32(i32) action;cleanup{}}", false, false, "fn i32(i32)"),
		one("enum-direct-payload", "module main;enum Stored{Value{fn i32(i32) action;};}", false, false, "fn i32(i32)"),
		one("repr-direct-field", "module main;@repr(C) struct Stored{fn i32(i32) action;}", false, false, "fn i32(i32)"),
		{"resource-imported-field", []string{prefix, "module main;import provider as p;resource Stored{p::Holder value;cleanup{}}", holder}, false, false, 1, "p::Holder"},
		{"enum-imported-payload", []string{prefix, "module main;import provider as p;enum Stored{Value{p::Holder value;};}", holder}, false, false, 1, "p::Holder"},
		{"repr-imported-field", []string{prefix, "module main;import provider as p;@repr(C) struct Stored{p::Holder value;}", holder}, false, false, 1, "p::Holder"},
		// Go semantic accepts this source; mir.Lower owns the current stricter storage refusal.
		{"nested-carrier-storage", []string{prefix, "module main;import provider as p;struct Outer{p::Holder value;}fn Outer keep(Outer value){return value;}", holder}, true, false, 1, "p::Holder"},
		one("cycle-option-before-leaf", "module main;struct Loop{Option<Loop> next;fn i32(i32) action;}", false, false, "Option<Loop>"),
		one("cycle-pointer-after-leaf", "module main;struct Loop{fn i32(i32) action;ptr Loop next;}", false, false, "ptr Loop"),
		one("mutual-cycle-result", "module main;struct A{Result<i32,B> next;}struct B{A back;fn i32(i32) action;}", false, false, "Result<i32,B>"),
		{"provider-chain-wrapper", []string{prefix, "module main;import middle as m;fn Option<m::Middle> keep(Option<m::Middle> value){return value;}", "module middle;import provider as p;pub struct Middle{p::Holder value;}", holder}, false, false, 1, "Option<m::Middle>"},
		{"provider-cycle-wrapper", []string{prefix, "module main;import left as a;fn ptr a::A keep(ptr a::A value){return value;}", "module left;import right as b;pub struct A{b::B next;}", "module right;import left as a;pub struct B{Option<a::A> back;fn i32(i32) action;}"}, false, false, 1, "ptr a::A"},
		one("legacy-scalar", "module main;fn i32 keep(i32 value){let i32 copy=value;return copy;}", true, true, ""),
		one("legacy-recursive-pointer", "module main;struct Node{ptr Node next;i32 value;}fn Option<Node> keep(Option<Node> value){return value;}", true, true, ""),
		one("legacy-resource-enum-repr", "module main;resource R{i32 value;cleanup{}}enum E{Value{i32 value;};}@repr(C) struct Packet{u32 value;}fn i32 keep(i32 value){return value;}", true, true, ""),
		{"legacy-imported-wrapper", []string{prefix, "module main;import provider as p;fn Option<p::Plain> keep(Option<p::Plain> value){return value;}", "module provider;pub struct Plain{i32 value;}"}, true, true, 0, ""},
	}
	vec, err := os.ReadFile(filepath.Join("..", "sdkstdlib", "sources", "optional", "std", "vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases = append(cases, closureInterfaceCase{"imported-vector", []string{prefix, "module main;import provider as p;import std::vec as v;fn v::Vec<p::Holder> keep(v::Vec<p::Holder> value){return value;}", holder, string(vec)}, false, false, 1, "v::Vec<p::Holder>"})
	return cases
}

func candidateClosureInterfacesOracle(t *testing.T, root string) string {
	t.Helper()
	var wire bytes.Buffer
	wire.WriteString("FYCVIF01")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(len(s)); wire.WriteString(s) }
	cases := closureInterfaceSources(t)
	number(len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var files []semantic.File
			var originals [][]byte
			for _, raw := range tc.files {
				path := strings.ReplaceAll(strings.TrimPrefix(raw[:strings.Index(raw, ";")], "module "), "::", "/") + ".fy"
				p, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(raw)), source.DefaultLimits())
				if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
					t.Fatalf("original Go parse %s: %v %+v", path, err, p.Tree.Diagnostics)
				}
				before, err := json.Marshal(p.AST)
				if err != nil {
					t.Fatal(err)
				}
				originals = append(originals, before)
				files = append(files, semantic.File{Package: hir.DefaultPackage, Path: path, Module: p.AST})
			}
			analysis := semantic.Analyze(files)
			if strings.Contains(tc.name, "generic") || tc.name == "imported-vector" {
				input := semantic.PackageInput{Files: append([]semantic.File(nil), files...)}
				coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
				for i := range input.Files {
					input.Files[i].Package = hir.PackageID(i + 1)
					coordinates[hir.PackageID(i+1)] = packagegraph.CoordinateID(sha256.Sum256([]byte(tc.files[i])))
				}
				for _, f := range input.Files {
					for _, imported := range f.Module.Imports {
						found := false
						for _, provider := range input.Files {
							if provider.Module.Path == imported.Path {
								input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: f.Package, ImportPath: imported.Path, ToPackage: provider.Package})
								found = true
								break
							}
						}
						if !found {
							t.Fatalf("missing original provider %s", imported.Path)
						}
					}
				}
				var generic compiler.GenericAnalysis
				var err error
				if tc.name == "imported-vector" {
					sdkPackage := input.Files[len(input.Files)-1].Package
					generic, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{sdkPackage})
				} else {
					generic, err = compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
				}
				if err != nil {
					t.Fatalf("original Go generic source pipeline: %v", err)
				}
				analysis = generic.Analysis.Semantic
			}
			accepted := len(analysis.Diagnostics) == 0 && len(analysis.Unsupported) == 0 && analysis.CompletedThrough == semantic.PhaseFlow
			if accepted != tc.semanticOK {
				t.Fatalf("original Go semantic accepted=%v want=%v: diagnostics=%+v unsupported=%+v", accepted, tc.semanticOK, analysis.Diagnostics, analysis.Unsupported)
			}
			if accepted {
				p, err := mir.Lower(analysis.HIR)
				failures := mir.Verify(p)
				if (err == nil && len(failures) == 0) != tc.interfaceOK {
					t.Fatalf("original Go MIR storage: lower=%v verification=%+v; interface acceptance want %v", err, failures, tc.interfaceOK)
				}
				t.Logf("original Go semantic accepted; MIR lower=%v verification=%+v", err, failures)
			} else {
				t.Logf("original Go rejection: %+v %+v", analysis.Diagnostics, analysis.Unsupported)
			}
			for i, f := range files {
				after, err := json.Marshal(f.Module)
				if err != nil || !bytes.Equal(after, originals[i]) {
					t.Fatalf("original AST mutated for module %d: %v", i, err)
				}
			}
			field(tc.name)
			number(len(tc.files))
			for _, raw := range tc.files {
				field(raw)
			}
			if tc.interfaceOK {
				number(1)
			} else {
				number(0)
			}
			number(tc.badModule)
			code := 9
			if tc.name == "imported-generic-call" {
				code = 4502
			}
			number(code)
			start, end := 0, 0
			if tc.badText != "" {
				start = strings.Index(tc.files[tc.badModule], tc.badText)
				if tc.name == "imported-generic-call" {
					start = strings.LastIndex(tc.files[tc.badModule], tc.badText)
				}
				if start < 0 {
					t.Fatal("missing original failure interval")
				}
				end = start + len(tc.badText)
			}
			number(start)
			number(end)
		})
	}
	if t.Failed() {
		t.Fatal("original Go classification failed; refusing to run a partial oracle")
	}
	path := filepath.Join(root, "closure-value-interfaces-oracle.bin")
	if err := os.WriteFile(path, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d original-source interface classifications", len(cases))
	return path
}

func TestCandidateClosureValueInterfacesOracle(t *testing.T) {
	candidateClosureInterfacesOracle(t, t.TempDir())
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_interfaces.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_value_interfaces.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("Fairy interface fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
}

func TestCandidateClosureValueInterfaces(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small native interfaces probe requires Windows AMD64; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	input := candidateClosureInterfacesOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_interfaces.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_value_interfaces;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(filesystem,console) fn i32 main(){closure_interfaces_match_original_go_sources();closure_interfaces_preserve_source_authority();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "module_signatures", "module_catalog", "module_unit_bind", "module_unit_interfaces", "module_types", "syntax", "syntax_parse_driver", "scanner", "file_source")
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("small original-source interfaces check: %d %s %s", code, out, diagnostic)
	}
	for _, args := range [][]string{{"run", "--", input}, {"run", "--release", "--locked", "--", input}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("small original-source interfaces %v: %d %q %s", args, code, out, diagnostic)
		}
		t.Logf("small original-source interfaces %v: exit 0, empty stdout/stderr", args)
	}
}
