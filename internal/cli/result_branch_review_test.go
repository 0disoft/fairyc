package cli

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestResultBranchReviewReturnOnlyArm(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.result-review\"\nname = \"result-review\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", `module main;
resource Item { i32 number; cleanup {} }
fn Result<Item, i32> make() { return .ok(Item { .number = 42 }); }
fn i32 main() {
 let Item owned = switch (make()) { case .ok(value): { value } case .err(_): { return 3; } };
 return owned.number;
}`)
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(w)
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified {
		t.Fatalf("return-only Result arm rejected: diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
}

func TestResultBranchReviewVoidArmStillRejected(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.result-void-arm\"\nname = \"result-void-arm\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", `module main;
fn Result<i32, i32> make() { return .ok(42); }
fn i32 main() {
 let i32 value = switch (make()) { case .ok(value): { value } case .err(_): {} };
 return value;
}`)
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(w)
	if err != nil {
		t.Fatal(err)
	}
	if checked.Verified {
		t.Fatal("non-returning void arm accepted in value-producing Result switch")
	}
	found := false
	for _, d := range checked.Semantic.Diagnostics {
		found = found || d.Code == "E1006"
	}
	if !found {
		t.Fatalf("expected E1006, diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
}

func TestResultBranchReviewAllReturnCannotInitialize(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.result-all-return\"\nname = \"result-all-return\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", `module main;
fn Result<i32, i32> make() { return .ok(42); }
fn i32 main() {
 let i32 value = switch (make()) { case .ok(_): { return 1; } case .err(_): { return 2; } };
 return value;
}`)
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(w)
	if err != nil {
		t.Fatal(err)
	}
	if checked.Verified {
		t.Fatal("all-return Result switch initialized a value")
	}
	found := false
	for _, d := range checked.Semantic.Diagnostics {
		found = found || d.Code == "E1006"
	}
	if !found {
		t.Fatalf("expected E1006, diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
}

func TestResultBranchReviewReturnOnlyArmNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	for _, tc := range []struct{ name, result string }{
		{"success", ".ok(Item { .number = 42 })"},
		{"error", ".err(7)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.result-return-native\"\nname = \"result-return-native\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", `module main;
resource Item { i32 number; cleanup {} }
fn Result<Item, i32> make() { return `+tc.result+`; }
fn i32 main() {
 let Item owned = switch (make()) { case .ok(value): { value } case .err(_): { return 0; } };
 assert(owned.number == 42);
 return 0;
}`)
			for _, args := range [][]string{{"check"}, {"run", "--locked"}, {"run", "--release", "--locked"}} {
				code, out, stderr := runForTest(args, root)
				if code != ExitSuccess || stderr != "" {
					t.Fatalf("%v: code=%d out=%q stderr=%q", args, code, out, stderr)
				}
			}
		})
	}
}

func TestResultBranchReviewDiagnosticsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC native evidence")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	normal := `var v::Vec<diagnostics::Entry> entries = v::empty<diagnostics::Entry>();
let i32 status = diagnostics::add(ref mut entries, 1, 0, 1, false, b"x");
assert(v::len<diagnostics::Entry>(ref entries) == if (status == 0) { 1 } else { 0 });
if (status == 0) { assert(diagnostics::info(v::at<diagnostics::Entry>(ref entries,0)).code == 1); assert(diagnostics::token_view(v::at<diagnostics::Entry>(ref entries,0)).len == 1); }
return status;`
	limits := `var v::Vec<diagnostics::Entry> entries = v::empty<diagnostics::Entry>();
var usize i = 0;
while (i < 256) { assert(diagnostics::add(ref mut entries, 1, 0, 1, false, b"x") == 0); i += 1; }
assert(diagnostics::add_limited(ref mut entries, 256, 256, 1, 0, 1, false, b"x") == 5);
assert(v::len<diagnostics::Entry>(ref entries) == 256);
assert(diagnostics::add(ref mut entries, 1, 0, 1, false, b"x") == 4);
assert(v::len<diagnostics::Entry>(ref entries) == 257);
var v::Vec<diagnostics::Entry> overflow = v::empty<diagnostics::Entry>();
i = 0;
while (i < 256) { assert(diagnostics::add(ref mut overflow, 1, 0, 1, false, b"x") == 0); i += 1; }
assert(diagnostics::add(ref mut overflow, 10, 0, 1, false, b"x") == 0);
assert(diagnostics::add(ref mut overflow, 10, 0, 1, false, b"x") == 4);
assert(v::len<diagnostics::Entry>(ref overflow) == 258);
return 0;`
	for _, tc := range []struct {
		name, body   string
		fail, status int
	}{
		{"success", normal, 0, 0}, {"copy-allocation-failure", normal, 1, 3}, {"push-allocation-failure", normal, 2, 3}, {"limits", limits, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerStreamExample(t, root)
			writeTestSource(t, root, "src/main.fy", `module main; import diagnostics as diagnostics; import std::vec as v;
@link_name("fairyc_heap_finish") unsafe extern C fn i32 finish(i32 status);
fn i32 exercise() { `+tc.body+` }
fn i32 main() { let i32 status = exercise(); return unsafe { finish(status) }; }`)
			var diagnostics strings.Builder
			lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "result branch review", &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("lower: %s", diagnostics.String())
			}
			unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			var symbols []string
			for _, symbol := range unit.ForeignSymbols {
				if !strings.HasPrefix(symbol, "fairyc_heap_") {
					symbols = append(symbols, symbol)
				}
			}
			support, err := windowsVectorSupportSource(symbols, false)
			if err != nil {
				t.Fatal(err)
			}
			support = fmt.Sprintf("#define HEAP_FAIL_AT %d\n", tc.fail) + lexerHeapProbe + support
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" {
					t.Fatalf("%s: %+v %v", profile.Name, outcome, err)
				}
				var row struct {
					Attempts, Allocations, Frees, Failures, LiveBytes, LiveBlocks uint64
					Status                                                        int
				}
				if err := json.Unmarshal([]byte(outcome.Stdout), &row); err != nil {
					t.Fatal(err)
				}
				wantFailures := uint64(0)
				if tc.fail != 0 {
					wantFailures = 1
				}
				if row.Status != tc.status || row.Failures != wantFailures || row.Allocations != row.Frees || row.LiveBytes != 0 || row.LiveBlocks != 0 || row.Attempts != row.Allocations+row.Failures {
					t.Fatalf("%s ledger: %+v", profile.Name, row)
				}
				t.Logf("%s: status=%d allocations=%d frees=%d failures=%d live=%d", profile.Name, row.Status, row.Allocations, row.Frees, row.Failures, row.LiveBlocks)
			}
		})
	}
}
