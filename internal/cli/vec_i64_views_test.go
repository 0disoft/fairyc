package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const vecI64ViewBase = `module main;
import std::vec as v;
fn i32 take(v::Vec<i64> values) { return 0; }
fn ref i64[] tail(ref v::Vec<i64> values) { return v::view_i64(values)[..]; }
fn Option<ref i64[]> tagged(ref v::Vec<i64> values) { return .some(tail(values)); }
fn i32 main() {
    var v::Vec<i64> values = v::empty<i64>();
    assert(switch (v::push<i64>(ref mut values, 4000000000)) { case .ok(_) : { true } case .err(_) : { false } });
    let ref i64[] view = v::view_i64(ref values);
    ACTION
    assert(view[0] == 4000000000);
    return 0;
}
`

func writeVecI64ViewWorkspace(t *testing.T, root, source string) {
	t.Helper()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.vec-i64-view\"\nname = \"vec-i64-view\"\nversion = \"0.1.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", source)
}

func TestVecI64ViewBorrowRejections(t *testing.T) {
	cases := []struct{ name, action, code string }{
		{"push", "let Result<usize, i64> result = v::push<i64>(ref mut values, 42);", "E2101"},
		{"reserve", "let Result<usize, u32> result = v::reserve<i64>(ref mut values, 1024);", "E2101"},
		{"pop", "let Option<i64> result = v::pop<i64>(ref mut values);", "E2101"},
		{"truncate", "let usize result = v::truncate<i64>(ref mut values, 1);", "E2101"},
		{"clear", "let usize result = v::clear<i64>(ref mut values);", "E2101"},
		{"exclusive-element", "let ref mut i64 result = v::at_mut<i64>(ref mut values, 0);", "E2101"},
		{"move", "let v::Vec<i64> result = values;", "E2105"},
		{"move-call", "let i32 result = take(values);", "E2105"},
		{"replace", "values = v::empty<i64>();", "E2101"},
		{"view-write", "view[0] = 2;", ""},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, strings.Replace(vecI64ViewBase, "ACTION", item.action, 1))
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || item.code != "" && !strings.Contains(stderr, item.code) {
				t.Fatalf("expected rejection %s: %d %s", item.code, code, stderr)
			}
		})
	}
	for name, source := range map[string]string{
		"subview-alias":          strings.Replace(strings.Replace(vecI64ViewBase, "ACTION", "let ref i64[] alias = view[..]; let usize ignored = v::clear<i64>(ref mut values);", 1), "assert(view[0]", "assert(alias[0]", 1),
		"derived-tail":           strings.Replace(strings.Replace(vecI64ViewBase, "v::view_i64(ref values);", "tail(ref values);", 1), "ACTION", "let usize ignored = v::clear<i64>(ref mut values);", 1),
		"tagged-view":            strings.Replace(strings.Replace(vecI64ViewBase, "v::view_i64(ref values);", "switch (tagged(ref values)) { case .some(value) : { value } case .none : { tail(ref values) } };", 1), "ACTION", "let usize ignored = v::clear<i64>(ref mut values);", 1),
		"index-effect":           strings.Replace(strings.Replace(vecI64ViewBase, "ACTION", "", 1), "view[0]", "view[{ let usize ignored = v::clear<i64>(ref mut values); 0 }]", 1),
		"bound-effect":           strings.Replace(vecI64ViewBase, "ACTION", "let ref i64[] child = view[..{ let usize ignored = v::clear<i64>(ref mut values); 1 }];", 1),
		"local-escape":           "module main; import std::vec as v; fn ref i64[] bad() { let v::Vec<i64> values = v::empty<i64>(); return v::view_i64(ref values); } fn i32 main() { return 0; }",
		"wrong-element":          "module main; import std::vec as v; fn i32 main() { let v::Vec<i32> values = v::empty<i32>(); let ref i64[] view = v::view_i64(ref values); assert(view.is_empty); return 0; }",
		"user-private-operation": "module main; fn i32 main() { let i64 unused = unsafe { storage_borrow<i64[]>(null, null, 0) }; return 0; }",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, stderr := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || stderr == "" {
				t.Fatalf("invalid view source accepted: %d %s", code, stderr)
			}
			want := "E2101"
			if name == "local-escape" {
				want = "E2102"
			}
			if name == "wrong-element" {
				want = "E1006"
			}
			if name != "user-private-operation" && !strings.Contains(stderr, want) {
				t.Fatalf("expected %s: %s", want, stderr)
			}
		})
	}
}

func vecI64NativeCases(t *testing.T) []struct{ name, source, trap string } {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "vec_i64_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"example", string(source), ""}}
	for _, item := range []struct{ name, expression string }{
		{"index-end", "view[1]"}, {"empty-index", "view[1..][0]"},
		{"reversed", "view[1..0][0]"}, {"past-end", "view[..2][0]"},
	} {
		text := strings.Replace(vecI64ViewBase, "ACTION", "let i64 unused = "+item.expression+";", 1)
		cases = append(cases, struct{ name, source, trap string }{item.name, text, "bounds"})
	}
	return cases
}

func TestVecI64ViewNative(t *testing.T) {
	runI64ViewNativeCases(t, "vec-i64-view", vecI64NativeCases(t))
}

func runI64ViewNativeCases(t *testing.T, label string, cases []struct{ name, source, trap string }) {
	t.Helper()
	if runtime.GOARCH != "amd64" || runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("requires Windows MSVC or Linux GCC/Clang amd64")
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", label+"-native", runtime.GOOS)
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, item.source)
			workspace, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := workspacecompiler.Check(workspace)
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			storage, err := workspaceSDKStorage(workspace)
			if err != nil {
				t.Fatal(err)
			}
			for _, release := range []bool{false, true} {
				args := []string{"run"}
				profile := "dev"
				if release {
					args = append(args, "--release", "--locked")
					profile = "release"
				}
				var data []byte
				code, out, diag := 0, "", ""
				row := map[string]any{"case": item.name, "profile": profile, "fixtureRoot": root, "expectedTrap": item.trap}
				if runtime.GOOS == "windows" {
					profile := cdriver.LowProfile
					if release {
						profile = cdriver.OptimizedProfile
					}
					var diagnostics bytes.Buffer
					artifact, buildCode := compileWindowsProgramPolicy(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, true, storage, &diagnostics)
					if buildCode != ExitSuccess {
						t.Fatal(diagnostics.String())
					}
					outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
					if err != nil {
						t.Fatal(err)
					}
					_, trap, err := classifyStableRunOutcome(outcome)
					if err != nil {
						t.Fatal(err)
					}
					code, out, diag = int(outcome.ExitCode), outcome.Stdout, outcome.Stderr
					if trap != "" {
						code, diag = ExitProgramTrap, "trap: "+trap+"\n"
					}
					data = artifact.Bytes
					row["executor"] = "cdriver.ExecuteWindowsMSVCTarget"
					row["rawOutcome"] = outcome
					row["toolchain"] = map[string]string{"name": artifact.Toolchain.Name, "version": artifact.Toolchain.Version, "path": artifact.Toolchain.Path}
				} else {
					code, out, diag = runForTest(args, root)
					row["executor"], row["argv"] = "stable CLI run", args
				}
				if item.trap != "" {
					if code != ExitProgramTrap || out != "" || diag != "trap: "+item.trap+"\n" {
						t.Fatalf("trap: %d %q %q", code, out, diag)
					}
				} else if code != 0 || out != "" || diag != "" {
					t.Fatalf("%s run: %d %q %q", profile, code, out, diag)
				}
				if runtime.GOOS == "windows" {
					if err := os.WriteFile(filepath.Join(evidenceRoot, item.name+"-"+profile+".exe"), data, 0700); err != nil {
						t.Fatal(err)
					}
				}
				sum, sourceSum := sha256.Sum256(data), sha256.Sum256([]byte(item.source))
				row["exit"], row["stdout"], row["stderr"], row["sourceSHA256"] = code, out, diag, hex.EncodeToString(sourceSum[:])
				if len(data) > 0 {
					row["imageSHA256"] = hex.EncodeToString(sum[:])
				}
				evidence = append(evidence, row)
			}
		})
	}
	if !t.Failed() {
		data, err := json.MarshalIndent(map[string]any{"schema": "fairyc." + label + "-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
