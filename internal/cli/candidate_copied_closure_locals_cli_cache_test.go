package cli

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func copiedClosureLocalsCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := copiedClosureCLIWorkspace(t)
	writeTestSource(t, root, "src/main.fy", strings.Replace(copiedClosureLocalsProgram(), "module root;", "module main;", 1))
	writeTestSource(t, root, "tests/closures.fy", `module tests::closures;
@test fn void locals_contract(){
var i32 base=40;let i32 sum=7;
let first=fn [i32 base] i32(i32 x){let i32 sum=base+x;let twice=sum*2;let result=twice/2;result};base=100;
assert(first(2)==42);assert(first(3)==43);assert(first(2)==42);assert(sum==7);
let second=fn [i32 base] i32(i32 x){let sum=base+x;let i32 result=sum+1;return result;};base=200;
assert(second(2)==103);assert(first(2)==42);
let i32 zero=0;let never_called=fn [i32 zero] i32(){let unused=1/zero;42};
let bounded=fn [] i32(i32 x){`+copiedClosureLocalsCountBody(16)+`};assert(bounded(42)==42);
let budget=fn [] i32(i32 x){`+copiedClosureLocalsBudgetBody(false)+`};assert(budget(1)==31);
}
`)
	return root
}

func TestCandidateCopiedClosureLocalsCLI(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			// This helper already selects the explicit private compiler-host builder,
			// retaining the authored main and ordinary product transport limits.
			image := copiedClosureCLIHost(t, profile)
			command := copiedClosureCommand{t: t, image: image, root: copiedClosureLocalsCLIWorkspace(t)}
			if out := command.cache(0, "cache: cold stored", "check"); !strings.Contains(out, "check ok:") {
				t.Fatal(out)
			}
			if out := command.cache(0, "cache: hit", "check", "--locked"); !strings.Contains(out, "check ok:") {
				t.Fatal(out)
			}
			for _, flags := range [][]string{nil, {"--release"}} {
				if out := command.cache(0, "cache: hit", append([]string{"build", "--locked"}, flags...)...); !strings.Contains(out, "built:") {
					t.Fatal(out)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(command.root, "build", "closure-cli.exe")).CombinedOutput()
				cancel()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("built locals program: %v %s", err, out)
				}
				if out := command.cache(42, "cache: hit", append([]string{"run", "--locked"}, flags...)...); out != "" {
					t.Fatal(out)
				}
			}
			out, diagnostics := command.run(0, "test", "--list", "--json", "--locked")
			if diagnostics != "" || !strings.Contains(out, "locals_contract") || !strings.Contains(out, `"count":1`) {
				t.Fatalf("locals list: %s %s", out, diagnostics)
			}
			for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
				out, diagnostics := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
				if diagnostics != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
					t.Fatalf("locals test: %s %s", out, diagnostics)
				}
			}
			t.Logf("%d independent CLI processes + 2 built images: typed/inferred/transitive locals, snapshots, repeated calls, unused creation, generics, 16 locals/64 nodes", command.calls)
		})
	}
}

func copiedClosureLocalsCacheProvider(generic bool) string {
	signature := "pub fn i32 value()"
	if generic {
		signature = "pub fn i32 value<T>()"
	}
	return "module answer;" + signature + `{
var i32 offset=40;
let add=fn [i32 offset] i32(i32 n){let first=1;let second=2;let i32 snapshot=offset;let sum=snapshot+n;let result=sum+0;return result;};
offset=100;assert(add(3)!=add(2));return add(2);}`
}

func copiedClosureLocalsCacheWorkspace(t *testing.T, generic bool) string {
	t.Helper()
	root := copiedClosureLocalsCLIWorkspace(t)
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.closure-locals-cache"
name = "closure-locals-cache"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeTestSource(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.closure-locals-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	call := "a::value()"
	if generic {
		call = "a::value<i32>()"
	}
	checks := ""
	if generic {
		checks = "assert(a::value<i64>()==a::value<i32>());"
	}
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;fn i32 main(){"+checks+"return "+call+";}")
	writeTestSource(t, root, "libs/answer/src/answer.fy", copiedClosureLocalsCacheProvider(generic))
	return root
}

type copiedClosureLocalsCacheEdit struct {
	name, source string
	value        int
}

func copiedClosureLocalsCacheEdits(provider string) []copiedClosureLocalsCacheEdit {
	return []copiedClosureLocalsCacheEdit{
		{"initializer", strings.Replace(provider, "sum=snapshot+n", "sum=snapshot-n", 1), 38},
		{"returned-local", strings.Replace(provider, "result=sum+0", "result=sum+1", 1), 43},
		{"unused-initializer", strings.Replace(provider, "first=1", "first=3", 1), 42},
		{"unused-order", strings.Replace(provider, "let first=1;let second=2;", "let second=2;let first=1;", 1), 42},
	}
}

func copiedClosureLocalsCacheRefusals(provider string) []copiedClosureLocalsCacheEdit {
	body := "let first=1;let second=2;let i32 snapshot=offset;let sum=snapshot+n;let result=sum+0;return result;"
	return []copiedClosureLocalsCacheEdit{
		{"mutable", strings.Replace(provider, "let first=1", "var i32 first=1", 1), 0},
		{"unused-bool", strings.Replace(provider, "first=1", "first=true", 1), 0},
		{"seventeen-locals", strings.Replace(provider, body, strings.ReplaceAll(copiedClosureLocalsCountBody(17), "=x;", "=n;"), 1), 0},
		{"sixty-five-nodes", strings.Replace(provider, body, strings.ReplaceAll(copiedClosureLocalsBudgetBody(true), "x", "n"), 1), 0},
	}
}

// Corruption repair need only restore the selected generation. Other disposable
// generations may remain corrupt. Keep every file in the stability comparison,
// including those rejected entries, and require at least one valid generation.
func copiedClosureLocalsCacheSnapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	rows, err := os.ReadDir(filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string][32]byte{}
	valid := 0
	for _, row := range rows {
		if strings.HasPrefix(row.Name(), ".stage") {
			t.Fatalf("cache stage leaked: %s", row.Name())
		}
		if !strings.HasPrefix(row.Name(), "fyproject0-") || !strings.HasSuffix(row.Name(), ".fyc") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "cache", row.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) >= 8 && string(raw[:8]) == "FYCACHE0" {
			valid++
		}
		result[row.Name()] = sha256.Sum256(raw)
	}
	if valid == 0 {
		t.Fatal("no valid persistent project generation")
	}
	return result
}

func copiedClosureLocalsWarm(t *testing.T, command *copiedClosureCommand, value int) {
	t.Helper()
	before := copiedClosureLocalsCacheSnapshot(t, command.root)
	lock, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	command.cache(0, "cache: hit", "check", "--locked")
	if out := command.cache(value, "cache: hit", "run", "--locked", "--release"); out != "" {
		t.Fatal(out)
	}
	after := copiedClosureLocalsCacheSnapshot(t, command.root)
	locked, err := os.ReadFile(filepath.Join(command.root, "fairy.lock"))
	if err != nil || string(lock) != string(locked) || !reflect.DeepEqual(before, after) {
		t.Fatal("fresh locked processes changed cache generation names/bytes or lock", err)
	}
}

func TestCandidateCopiedClosureLocalsPersistentCache(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			for _, generic := range []bool{false, true} {
				family := "ordinary"
				if generic {
					family = "generic"
				}
				t.Run(family, func(t *testing.T) {
					root := copiedClosureLocalsCacheWorkspace(t, generic)
					provider := copiedClosureLocalsCacheProvider(generic)
					set := func(raw string) { t.Helper(); writeTestSource(t, root, "libs/answer/src/answer.fy", raw) }
					command := copiedClosureCommand{t: t, image: image, root: root}
					command.cache(0, "cache: cold stored", "check")
					// Cold accepted sources must write normal valid project records.
					copiedClosureCacheSnapshot(t, root)
					copiedClosureLocalsWarm(t, &command, 42)
					for _, edit := range copiedClosureLocalsCacheEdits(provider) {
						t.Logf("edit %s", edit.name)
						before := copiedClosureLocalsCacheSnapshot(t, root)
						set(edit.source)
						command.cache(7, "cache: skipped", "check", "--locked")
						out, diagnostics := command.run(edit.value, "run", "--cache-root", "cache", "--cache-status")
						if out != "" || strings.HasPrefix(diagnostics, "cache: hit\n") || !strings.Contains(diagnostics, "stored") && !strings.Contains(diagnostics, "mixed") && !strings.Contains(diagnostics, "units hit") {
							t.Fatalf("edited locals were not rebuilt: %q %s", out, diagnostics)
						}
						if reflect.DeepEqual(before, copiedClosureLocalsCacheSnapshot(t, root)) {
							t.Fatal("initializer/result edit did not change persistent generation")
						}
						copiedClosureLocalsWarm(t, &command, edit.value)
					}
					valid := copiedClosureLocalsCacheEdits(provider)[3].source
					rows, err := os.ReadDir(filepath.Join(root, "cache"))
					if err != nil {
						t.Fatal(err)
					}
					for _, row := range rows {
						if strings.HasSuffix(row.Name(), ".fyc") {
							if err := os.WriteFile(filepath.Join(root, "cache", row.Name()), []byte("corrupt"), 0600); err != nil {
								t.Fatal(err)
							}
						}
					}
					command.cache(0, "cache: cold stored", "check", "--locked")
					copiedClosureLocalsWarm(t, &command, 42)
					for _, refusal := range copiedClosureLocalsCacheRefusals(provider) {
						t.Logf("refusal %s", refusal.name)
						before := copiedClosureLocalsCacheSnapshot(t, root)
						set(refusal.source)
						command.cache(7, "cache: skipped", "check", "--locked")
						// Observed Fairy admission returns 7 here; the Go fixture
						// oracle returns 3. This asserts refusal/no publication,
						// not exit-code or diagnostic parity (see the receipt).
						out, diagnostics := command.run(7, "check", "--cache-root", "cache", "--cache-status")
						if !strings.HasPrefix(diagnostics, "cache: skipped\n") || !strings.Contains(diagnostics, "fairyc: package workspace failure: libs/answer/answer.fy: invalid package input\n") || out != "" {
							t.Fatalf("invalid local did not report package-input refusal: %q %s", out, diagnostics)
						}
						if !reflect.DeepEqual(before, copiedClosureLocalsCacheSnapshot(t, root)) {
							t.Fatal("invalid local modified valid persistent generations")
						}
						set(valid)
						copiedClosureLocalsWarm(t, &command, 42)
					}
					t.Logf("%s: %d independent CLI processes; ordinary/generic provider locals, equal-width edits, unused order, locked byte stability, corruption repair and invalid-source refusal", family, command.calls)
				})
			}
		})
	}
}

// No native compiler: independently check every generated fixture with the Go
// frontend before spending time on the full Fairy-authored CLI image.
func TestCandidateCopiedClosureLocalsCLIFixtures(t *testing.T) {
	root := copiedClosureLocalsCLIWorkspace(t)
	if code, out, diagnostics := runForTest([]string{"check"}, root); code != 0 || diagnostics != "" {
		t.Fatalf("main fixture: %d %s %s", code, out, diagnostics)
	}
	if code, out, diagnostics := runForTest([]string{"test", "--json", "--locked"}, root); code != 0 || diagnostics != "" || !strings.Contains(out, `"passed":1`) {
		t.Fatalf("test fixture: %d %s %s", code, out, diagnostics)
	}
	for _, generic := range []bool{false, true} {
		root := copiedClosureLocalsCacheWorkspace(t, generic)
		provider := copiedClosureLocalsCacheProvider(generic)
		for _, edit := range append([]copiedClosureLocalsCacheEdit{{"baseline", provider, 42}}, copiedClosureLocalsCacheEdits(provider)...) {
			writeTestSource(t, root, "libs/answer/src/answer.fy", edit.source)
			if code, out, diagnostics := runForTest([]string{"check"}, root); code != 0 || diagnostics != "" {
				t.Fatalf("provider generic=%t %s: %d %s %s", generic, edit.name, code, out, diagnostics)
			}
		}
		for _, refusal := range copiedClosureLocalsCacheRefusals(provider) {
			writeTestSource(t, root, "libs/answer/src/answer.fy", refusal.source)
			if code, _, diagnostics := runForTest([]string{"check"}, root); code != ExitSourceDiagnostics || diagnostics == "" {
				t.Fatalf("provider generic=%t %s must refuse: %d %s", generic, refusal.name, code, diagnostics)
			}
		}
	}
}
