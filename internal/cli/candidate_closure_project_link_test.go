package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type closureProjectLinkCase struct {
	name                         string
	files                        []string
	accept, closures, aggregates bool
	made, called, batches        int
	outOfGraph                   bool
}

func candidateClosureProjectLinkCases() []closureProjectLinkCase {
	const factory = `module provider;pub fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}`
	cases := []closureProjectLinkCase{
		{"single-unit-raw-link", []string{`module main;fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}fn i32 main(){let original=make(40);let copy=original;return copy(2);}`}, true, true, false, 1, 1, 1, false},
		{"factory-in-provider", []string{`module main;import provider as p;fn i32 main(){let action=p::make(40);return action(2);}`, factory}, true, true, false, 1, 1, 1, false},
		{"local-param-struct-cross-units", []string{`module main;import provider as p;import apply as a;fn i32 main(){let original=p::make(40);let copy=original;let holder=p::Holder{.action=copy};let stored=p::keep(holder);let extracted=stored.action;return a::apply(extracted,2)+extracted(0);}`, strings.Replace(factory, "module provider;", `module provider;pub struct Holder{fn i32(i32) action;}pub fn Holder keep(Holder value){return value;}`, 1), `module apply;pub fn i32 apply(fn i32(i32) action,i32 x){return action(x);}`}, true, true, false, 1, 2, 1, false},
		{"same-helper-declaration-different-modules", []string{`module main;import left as a;import right as b;fn i32 apply(fn i32(i32) action,i32 x){return action(x);}fn i32 main(){let a=a::make(40);let b=b::make(2);return apply(a,2)+apply(b,1);}`, strings.Replace(factory, "module provider;", "module left;", 1), strings.Replace(strings.Replace(factory, "module provider;", "module right;", 1), "base+x", "base*x", 1)}, true, true, false, 2, 1, 1, false},
		{"unused-value-keeps-helper-reachable", []string{`module main;import provider as p;fn i32 main(){let action=p::make(0);discard action;return 0;}`, factory}, true, true, false, 1, 0, 1, false},
		{"four-captures-provider", []string{`module main;import provider as p;fn i32 main(){let action=p::make();return action(2);}`, `module provider;pub fn fn i32(i32) make(){let a=1;let b=2;let c=3;let d=4;let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){return a+b+c+d+x;};return action;}`}, true, true, false, 1, 1, 1, false},
		{"resource-and-closure-neighbor", []string{`module main;import provider as p;fn i32 main(){let owner=p::owner();let action=p::make(40);return action(2);}`, strings.Replace(factory, "module provider;", `module provider;pub resource R{i32 n;cleanup{discard this.n;}}pub fn R owner(){return R{.n=2};}`, 1)}, true, true, false, 1, 1, 1, false},
		{"signature-only-no-construction", []string{`module main;fn i32 apply(fn i32(i32) action){return action(2);}fn i32 main(){return 0;}`}, true, true, false, 0, 1, 1, false},
		{"scalar-aggregate-closure-signature", []string{`module main;struct Box<T>{T value;}fn i32 apply(fn i32(i32) action,Box<i32> box){return action(box.value);}fn i32 main(){return 0;}`}, true, true, true, 0, 1, 1, false},
		{"resource-aggregate-closure-signature", []string{`module main;resource Box<T>{T value;cleanup{}}fn i32 apply(fn i32(i32) action,Box<i32> owner){return action(owner.value);}fn i32 main(){return 0;}`}, true, true, true, 0, 1, 1, false},
		{"old-scalar-single", []string{`module main;fn i32 main(){return 42;}`}, true, false, false, 0, 0, 1, false},
		{"old-scalar-multiple", []string{`module main;import provider as p;fn i32 main(){return p::value();}`, `module provider;pub fn i32 value(){return 42;}`}, true, false, false, 0, 0, 1, false},
		{"old-scalar-aggregate", []string{`module main;struct Box<T>{T value;}fn i32 read(Box<i32> box){return box.value;}fn i32 main(){return 0;}`}, true, false, true, 0, 0, 1, false},
	}
	padding := ""
	for i := 0; i < 16; i++ {
		padding += fmt.Sprintf("fn i32 pad%d(){return %d;}", i, i)
	}
	cases = append(cases, closureProjectLinkCase{"helper-target-is-not-a-type", []string{`module main;import provider as p;fn i32 main(){let u8[3] bytes=[1,2,3];discard bytes;let action=p::make(40);return action(2);}`, strings.Replace(factory, "module provider;", "module provider;"+padding, 1)}, true, true, false, 1, 1, 1, true})
	// Each module stays below the 128-declaration catalog cap. Its 576 padding
	// parameters fit the unit's 1024-entry budget; combined inputs exceed it.
	// Exercise real automatic batching without changing any compiler limit.
	rootPad, providerPad := "", ""
	var parameters []string
	for i := 0; i < 96; i++ {
		parameters = append(parameters, fmt.Sprintf("i32 p%d", i))
	}
	for i := 0; i < 6; i++ {
		rootPad += fmt.Sprintf("fn i32 rootPad%d(%s){return 0;}", i, strings.Join(parameters, ","))
		providerPad += fmt.Sprintf("fn i32 providerPad%d(%s){return 0;}", i, strings.Join(parameters, ","))
	}
	cases = append(cases, closureProjectLinkCase{"automatic-two-module-batches", []string{`module main;import provider as p;` + rootPad + `fn i32 main(){let action=p::make(40);return action(2);}`, strings.Replace(factory, "module provider;", "module provider;"+providerPad, 1)}, true, true, false, 1, 1, 2, false})
	for _, shape := range []struct{ name, text string }{
		{"option", "Option<p::Holder>"},
		{"result-success", "Result<p::Holder,i32>"},
		{"result-error", "Result<i32,p::Holder>"},
		{"pointer", "ptr p::Holder"},
		{"borrow", "ref p::Holder"},
	} {
		cases = append(cases, closureProjectLinkCase{name: "reject-" + shape.name, files: []string{"module main;import provider as p;fn " + shape.text + " keep(" + shape.text + " value){return value;}fn i32 main(){return 0;}", `module provider;pub struct Holder{fn i32(i32) action;}`}, batches: 1})
	}
	for _, tc := range []struct{ name, raw string }{
		{"resource-closure-field", `module main;resource Stored{fn i32(i32) action;cleanup{}}fn i32 main(){return 0;}`},
		{"ffi-closure", `module main;unsafe extern C fn void consume(fn i32(i32) action);fn i32 main(){return 0;}`},
		{"generic-closure-argument", `module main;fn T id<T>(T value){return value;}fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let copy=id<fn i32(i32)>(action);return copy(2);}`},
	} {
		cases = append(cases, closureProjectLinkCase{name: "reject-" + tc.name, files: []string{tc.raw}, batches: 1})
	}
	return cases
}

func candidateClosureProjectLinkControls(t *testing.T) []closureProjectLinkCase {
	t.Helper()
	cases := candidateClosureProjectLinkCases()
	for _, tc := range cases {
		var files []semantic.File
		for i, raw := range tc.files {
			module := strings.TrimSpace(strings.TrimPrefix(raw[:strings.IndexByte(raw, ';')], "module "))
			path := strings.ReplaceAll(module, "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(raw)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("Go project link original syntax %s/%d: %v %+v", tc.name, i, err, parsed.Tree.Diagnostics)
			}
			files = append(files, semantic.File{Package: hir.DefaultPackage, Path: path, Module: parsed.AST})
		}
		checked, err := compiler.CheckExplicitFiles(files)
		if accepted := err == nil && checked.Verified; accepted != tc.accept {
			t.Errorf("Go project link %s accepted=%v want=%v err=%v diagnostics=%+v unsupported=%+v", tc.name, accepted, tc.accept, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
	}
	if t.Failed() {
		t.Fatal("original-source project link Go controls failed")
	}
	return cases
}

func candidateClosureProjectLinkWorkspace(t *testing.T) (string, []closureProjectLinkCase) {
	t.Helper()
	cases := candidateClosureProjectLinkControls(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_project_link.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_project_link.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("project link fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_project_link;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + "\n@effects(console) fn i32 main(){\n"
	for _, tc := range cases {
		texts := [3]string{}
		copy(texts[:], tc.files)
		main += fmt.Sprintf("verify(b%s,b%s,b%s,b%s,%t,%t,%t,%d,%d,%d,%t);\n", strconv.Quote(tc.name), strconv.Quote(texts[0]), strconv.Quote(texts[1]), strconv.Quote(texts[2]), tc.accept, tc.closures, tc.aggregates, tc.made, tc.called, tc.batches, tc.outOfGraph)
	}
	main += "closure_project_link_missing_duplicate_and_empty_inputs_stay_closed();return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "module_project_link", "module_project_compile", "module_project_calls", "module_unit_body_ir_closures", "module_unit_cfg_lower", "module_unit_c_reachability", "module_unit_c_frame_index", "module_unit_c_callable_index", "project_source_plan", "generic_aggregate_types", "sha256_stream", "copied_closure_tables", "syntax", "scanner")
	return root, cases
}

// Real Go source checking only. Do not compile or execute native Fairy code
// while the parent reserves CPU for its separate integration probes.
func TestCandidateClosureProjectLinkSource(t *testing.T) {
	root, cases := candidateClosureProjectLinkWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure project link source check: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source controls; single/unit/module/batch project link fixture checked, not executed", len(cases))
}

// Reserved for parent execution. This runs the source-to-project-link producer;
// it does not claim closure program execution, CLI or restored artifact parity.
func TestCandidateClosureProjectLinkNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("project link native producer requires Windows AMD64; Linux remains batched")
	}
	root, cases := candidateClosureProjectLinkWorkspace(t)
	// Existing private producer partitioning keeps the real per-part 60s host
	// limit. This is separate from stable CLI optimized execution acceptance.
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "closure-project-link-"+profile.Name+".exe")
		if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		code, out := candidateAtlasProcess(t, path, nil)
		if code != ExitSuccess {
			t.Fatalf("closure project link native %s: %d\nstdout: %s", profile.Name, code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(cases) {
			t.Fatalf("project link receipts=%d want=%d: %s", len(lines), len(cases), out)
		}
		for i, tc := range cases {
			if !strings.HasPrefix(lines[i], "closure-project-link case="+tc.name+" ") {
				t.Fatalf("missing ordered project link receipt for %s: %q", tc.name, lines[i])
			}
		}
		t.Logf("%s: %d private producer parts, %d original-source project link receipts", profile.Name, len(build.Sources), len(cases))
	}
}
