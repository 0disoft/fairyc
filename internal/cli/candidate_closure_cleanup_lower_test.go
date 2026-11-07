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

type closureCleanupLowerCase struct {
	name, raw                    string
	accept, closures, aggregates bool
	drops, policyCode            int
}

func candidateClosureCleanupLowerCases() []closureCleanupLowerCase {
	cases := []closureCleanupLowerCase{
		{"old-scalar", `module main;fn i32 main(){return 0;}`, true, false, false, 0, 0},
		{"old-resource", `module main;resource R{i32 n;cleanup{discard this.n;}}fn i32 read(R owner){return owner.n;}`, true, false, false, 1, 0},
		{"signature-only", `module main;fn void consume(fn i32(i32) action){discard action(2);return;}`, true, true, false, 0, 0},
		{"factory-copy-struct", `module main;struct Holder{fn i32(i32) action;}fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}fn i32 main(){let original=make(40);let copy=original;let holder=Holder{.action=copy};let stored=holder;let extracted=stored.action;return extracted(2);}`, true, true, false, 0, 0},
		{"different-helpers", `module main;fn i32 apply(fn i32(i32) action,i32 x){return action(x);}fn i32 main(){let fn i32(i32) a=fn [] i32(i32 x){return x;};let base=4;let fn i32(i32) b=fn [i32 base] i32(i32 x){return base+x;};return apply(a,2)+apply(b,3);}`, true, true, false, 0, 0},
		{"closure-resource-neighbor", `module main;resource R{i32 n;cleanup{discard this.n;}}fn i32 apply(fn i32(i32) action,R owner){return action(owner.n);}`, true, true, false, 1, 0},
		{"closure-authorized-cleanup", `module main;@effects(console)resource R{i32 n;cleanup{discard this.n;}}@effects(console)fn i32 apply(fn i32(i32) action,R owner){return action(owner.n);}`, true, true, false, 1, 0},
		{"closure-missing-cleanup-effect", `module main;@effects(console)resource R{i32 n;cleanup{discard this.n;}}fn i32 apply(fn i32(i32) action,R owner){return action(owner.n);}`, false, true, false, 0, 7102},
		{"closure-no-drop", `module main;@abilities(store)resource R{i32 n;cleanup{}}fn i32 apply(fn i32(i32) action,R owner){return action(owner.n);}`, false, true, false, 0, 1108},
		{"old-aggregate", `module main;struct Box<T>{T value;}fn i32 read(Box<i32> box){return box.value;}`, true, false, true, 0, 0},
		{"closure-aggregate-neighbor", `module main;struct Box<T>{T value;}fn i32 apply(fn i32(i32) action,Box<i32> box){return action(box.value);}`, true, true, true, 0, 0},
		{"closure-aggregate-resource-neighbor", `module main;resource Box<T>{T value;cleanup{}}fn i32 apply(fn i32(i32) action,Box<i32> owner){return action(owner.value);}`, true, true, true, 1, 0},
	}
	for _, shape := range []struct{ name, text string }{
		{"option", "Option<fn i32(i32)>"},
		{"result-first", "Result<fn i32(i32),i32>"},
		{"result-second", "Result<i32,fn i32(i32)>"},
		{"array", "(fn i32(i32))[2]"},
		{"borrow", "ref fn i32(i32)"},
		{"pointer", "ptr fn i32(i32)"},
	} {
		cases = append(cases, closureCleanupLowerCase{name: "reject-" + shape.name, raw: "module main;fn void consume(" + shape.text + " action){return;}"})
	}
	for _, tc := range []struct{ name, raw string }{
		{"resource-field", `module main;resource Stored{fn i32(i32) action;cleanup{}}`},
		{"resource-struct-field", `module main;struct Holder{fn i32(i32) action;}resource Stored{Holder value;cleanup{}}`},
		{"repr-c", `module main;@repr(C)struct Holder{fn i32(i32) action;}`},
		{"ffi", `module main;unsafe extern C fn void consume(fn i32(i32) action);`},
		{"unsafe-callback-capture", `module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let unsafe extern C fn i32(i32) callback=action;return 0;}`},
		{"mutable-closure", `module main;fn i32 main(){var fn i32(i32) action=fn [] i32(i32 x){return x;};return action(2);}`},
		{"vec-wrapper", `module main;import std::vec as v;fn void consume(v::Vec<fn i32(i32)> action){return;}`},
	} {
		cases = append(cases, closureCleanupLowerCase{name: "reject-" + tc.name, raw: tc.raw})
	}
	return cases
}

func candidateClosureCleanupLowerControls(t *testing.T) []closureCleanupLowerCase {
	t.Helper()
	cases := candidateClosureCleanupLowerCases()
	for _, tc := range cases {
		p, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
			t.Fatalf("cleanup lower original syntax %s: %v %+v", tc.name, err, p.Tree.Diagnostics)
		}
		// std::vec is deliberately absent from this detached Go API. Resolve its
		// real SDK through a separate temporary CLI workspace, never count a
		// missing import as proof of wrapper refusal.
		if tc.name == "reject-vec-wrapper" {
			root := t.TempDir()
			copyLexerExample(t, root)
			writeTestSource(t, root, "src/main.fy", tc.raw)
			code, out, diagnostic := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diagnostic, "E4502") {
				t.Fatalf("Go real SDK Vec wrapper refusal: %d %s %s", code, out, diagnostic)
			}
			continue
		}
		checked, errCheck := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: p.AST}})
		if accepted := errCheck == nil && checked.Verified; accepted != tc.accept {
			t.Errorf("Go cleanup lower %s accepted=%v want=%v err=%v diagnostics=%+v unsupported=%+v", tc.name, accepted, tc.accept, errCheck, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
	}
	if t.Failed() {
		t.Fatal("original-source Go controls failed")
	}
	return cases
}

func candidateClosureCleanupLowerWorkspace(t *testing.T) (string, []closureCleanupLowerCase) {
	t.Helper()
	cases := candidateClosureCleanupLowerControls(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_cleanup_lower.fy"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_cleanup_lower.fy", raw), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		t.Fatalf("closure cleanup lower fixture syntax: %v %+v", err, p.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_cleanup_lower;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + "\n@effects(console) fn i32 main(){\n"
	for _, tc := range cases {
		provider := ""
		if tc.name == "reject-vec-wrapper" {
			data, err := os.ReadFile(filepath.Join("..", "sdkstdlib", "sources", "optional", "std", "vec.fy"))
			if err != nil {
				t.Fatal(err)
			}
			provider = string(data)
		}
		main += fmt.Sprintf("verify(b%s,b%s,b%s,%t,%t,%t,%d,%d);\n", strconv.Quote(tc.name), strconv.Quote(tc.raw), strconv.Quote(provider), tc.accept, tc.closures, tc.aggregates, tc.drops, tc.policyCode)
	}
	main += "closure_cleanup_lower_checked_sources_stay_fresh();return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "module_unit_cleanup_mir_lower", "module_unit_cleanup_mir_verify", "project_source_plan", "generic_aggregate_types", "module_catalog", "module_public_type_text", "sha256_stream")
	return root, cases
}

// Source-only Go frontend controls and real CLI check. No native compilation or
// execution: safe while the parent reserves the native compiler CPU.
func TestCandidateClosureCleanupLowerSource(t *testing.T) {
	root, cases := candidateClosureCleanupLowerWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure cleanup lower source check: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source Go controls; Fairy source/import closure checked, not executed", len(cases))
}

// Parent-only after CPU coordination. This executes the source-to-cleanup
// producer, not the source closure program. Linux remains a separate batch.
func TestCandidateClosureCleanupLowerNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small cleanup lower native probe requires Windows AMD64; Linux remains batched")
	}
	root, cases := candidateClosureCleanupLowerWorkspace(t)
	// The complete aggregate-aware producer is a compiler-sized input. Use the
	// existing split host builder; keep every source/control and the 60s limit.
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
		path := filepath.Join(root, "closure-cleanup-lower-"+profile.Name+".exe")
		if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		code, out := candidateAtlasProcess(t, path, nil)
		if code != ExitSuccess {
			t.Fatalf("closure cleanup lower native %s: %d\nstdout: %s", profile.Name, code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(cases) {
			t.Fatalf("cleanup lower receipts=%d want=%d: %s", len(lines), len(cases), out)
		}
		for i, tc := range cases {
			if !strings.HasPrefix(lines[i], "closure-cleanup-lower case="+tc.name+" ") || !strings.Contains(lines[i], " leaves=") || !strings.Contains(lines[i], " drops=") || !strings.Contains(lines[i], " work=") {
				t.Fatalf("missing ordered cleanup lower receipt for %s: %q", tc.name, lines[i])
			}
		}
		t.Logf("%s: %d C parts; %d original-source receipts and freshness assertions", profile.Name, len(build.Sources), len(cases))
	}
}
