package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Both copies are independently handled. Actual payload values come from the C
// observer; no new scalar literals, scalar operators or Result foreign ABI.
const compositeResultCLIProvider = `module answer;
pub fn Result<T,u32> success<T>(T left,T rght){return .ok(left);}
pub fn Result<T,u32> tail<T>(Result<T,u32> value){return value;}
pub fn Result<T,u32> relay<T>(Result<T,u32> value){
let Result<T,u32> copied=value;discard value;
var Result<T,u32> replaced=.err(9);discard replaced;
replaced=copied;discard copied;return replaced;
}
pub fn Result<T,E> bind<T,E>(Result<T,E> value){return value;}
pub fn u32 error_code(){return 77;}
pub fn Result<T,u32> failure<T>(){return .err(error_code());}
pub fn Result<T,u32> attempt<T>(T left,T rght,bool fail,ref mut i32 count){
*count+=1;if(fail){return failure<T>();}return success<T>(left,rght);
}
pub fn Result<T,u32> forward<T>(T left,T rght,bool fail,ref mut i32 count){
let T actual=attempt<T>(left,rght,fail,count)?;return .ok(actual);
}
pub fn T select<T>(T left,T rght,bool copied){
var i32 count=0;let Result<T,u32> original=forward<T>(left,rght,false,ref mut count);
let Result<T,u32> copy=bind<T,u32>(relay<T>(original));
let T original_value=switch(original){case .ok(value):{value}case .err(_):{assert(false);rght}};
let T copied_value=switch(copy){case .ok(value):{value}case .err(_):{assert(false);rght}};
let Result<T,u32> failed=forward<T>(left,rght,true,ref mut count);
assert(switch(failed){case .ok(_):{false}case .err(code):{code==error_code()}});
assert(count==2);return if(copied){copied_value}else{original_value};
}
pub fn T pick<T>(T left,T rght){return select<T>(left,rght,true);}
pub fn Result<T,u32> counted_error<T>(ref mut i32 count){*count+=1;return failure<T>();}
pub fn Result<T,u32> error_once<T>(ref mut i32 count){let T actual=counted_error<T>(count)?;return .ok(actual);}
pub fn u32 error_value<T>(){var i32 count=0;let Result<T,u32> value=error_once<T>(ref mut count);assert(count==1);return switch(value){case .ok(_):{0}case .err(code):{code}};}
fn i32 private_helper(){return 1;}
`

const compositeResultCLITests = `module tests::generics;import answer as a;
@test fn void result_contract(){
assert(a::pick<i32>(42,43)==42);assert(a::select<i64>(7,9,false)==7);
assert(a::select<i64>(7,9,true)==7);assert(a::error_value<i64>()==77);
let Result<i32,u32> original=a::success<i32>(42,43);
let Result<i32,u32> copied=a::relay<i32>(original);
assert(switch(original){case .ok(value):{value==42}case .err(_):{false}});
assert(switch(copied){case .ok(value):{value==42}case .err(_):{false}});}`

func compositeResultCLIMain() string {
	var wrappers strings.Builder
	wrappers.WriteString("module main;import answer as a;")
	for _, v := range genericCLIValues {
		fmt.Fprintf(&wrappers, `fn Result<%[1]s,u32> written_%[1]s(Result<%[1]s,u32> value){return value;}
pub fn %[1]s probe_%[1]s(%[1]s left,%[1]s rght,i32 marker){
let Result<%[1]s,u32> original=.ok(a::select<%[1]s>(left,rght,marker!=0));
let Result<%[1]s,u32> copied=written_%[1]s(original);discard original;
return switch(copied){case .ok(value):{value}case .err(_):{assert(false);rght}};}`, v.fairy)
	}
	wrappers.WriteString("pub fn u32 result_error_probe(i32 marker){return a::error_value<u16>();}")
	main := genericCLIRoot("a::", 42, 43)
	main = strings.Replace(main, "fn i32 main(){", "fn i32 main(){let Result<i8,u32> payload_choice=a::failure<i8>();discard payload_choice;assert(result_error_probe(0)==a::error_code());", 1)
	main = strings.Replace(main, "return a::pick<i32>(42,43);", "return if(a::error_code()==77){a::pick<i32>(42,43)}else{44};", 1)
	return wrappers.String() + main
}

func compositeResultCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := genericCLIWorkspace(t)
	manifest, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", strings.Replace(string(manifest), "generic-cli", "result-cli", -1)+`
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeTestSource(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.result-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "libs/answer/src/answer.fy", compositeResultCLIProvider)
	writeTestSource(t, root, "src/main.fy", compositeResultCLIMain())
	writeTestSource(t, root, "tests/generics.fy", compositeResultCLITests)
	return root
}

type compositeResultCLIEdit struct {
	name, provider, main string
	value                int
	second               bool
	errorCode            uint32
}

func compositeResultCLIEdits(full bool) []compositeResultCLIEdit {
	provider, main := compositeResultCLIProvider, compositeResultCLIMain()
	if !full {
		provider = strings.Replace(provider, "return .ok(left);", "return .ok(rght);", 1)
		return []compositeResultCLIEdit{{"success", provider, main, 43, true, 77}}
	}
	helper := strings.Replace(provider, "return replaced;", "return tail<T>(replaced);", 1)
	success := strings.Replace(helper, "return .ok(left);", "return .ok(rght);", 1)
	payload := strings.ReplaceAll(strings.Replace(main, "Result<i8,u32> payload_choice", "Result<i16,u32> payload_choice", 1), "payload_choice=a::failure<i8>()", "payload_choice=a::failure<i16>()")
	errorValue := strings.Replace(success, "error_code(){return 77;}", "error_code(){return 78;}", 1)
	private := strings.Replace(errorValue, "private_helper(){return 1;}", "private_helper(){return 2;}", 1)
	return []compositeResultCLIEdit{
		{"helper", helper, main, 42, false, 77},
		{"success", success, main, 43, true, 77},
		{"payload", success, payload, 43, true, 77},
		{"error-value", errorValue, payload, 44, true, 78},
		{"private-body", private, payload, 44, true, 78},
	}
}

func compositeResultCLIRefusals() []struct{ name, extra, code string } {
	return []struct{ name, extra, code string }{
		{"copy-pending", "fn void pending(Result<f32,u32> original){let Result<f32,u32> copied=original;discard original;}", "E4101"},
		{"original-pending", "fn void pending(Result<f32,u32> original){let Result<f32,u32> copied=original;discard copied;original;}", "E4101"},
		{"error-contract", "fn Result<f32,i32> mismatch(Result<f32,u32> value){let f32 actual=value?;return .ok(actual);}", "E4103"},
	}
}

// Go controls never build the authored compiler host. The interpreter executes
// error and i32/i64 success paths; the dead source seeds retain all seven types.
func TestCandidateCompositeResultCLIFixtures(t *testing.T) {
	root := compositeResultCLIWorkspace(t)
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Go executable=%s provider-sha256=%x main-sha256=%x tests-sha256=%x", identity.String(), sha256.Sum256([]byte(compositeResultCLIProvider)), sha256.Sum256([]byte(compositeResultCLIMain())), sha256.Sum256([]byte(compositeResultCLITests)))
	if code, out, diagnostic := runForTest([]string{"test", "--json"}, root); code != 0 || diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
		t.Fatalf("Go Result tests: %d %s %s", code, out, diagnostic)
	}
	edits := append([]compositeResultCLIEdit{{"baseline", compositeResultCLIProvider, compositeResultCLIMain(), 42, false, 77}}, compositeResultCLIEdits(true)...)
	for _, edit := range edits {
		writeTestSource(t, root, "libs/answer/src/answer.fy", edit.provider)
		writeTestSource(t, root, "src/main.fy", edit.main)
		if code, out, diagnostic := runForTest([]string{"check"}, root); code != 0 || diagnostic != "" {
			t.Fatalf("Go %s: %d %s %s", edit.name, code, out, diagnostic)
		}
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		compiled, err := workspacecompiler.Check(workspace)
		if err != nil || !compiled.Verified {
			t.Fatalf("Go workspace %s: %v %+v", edit.name, err, compiled.Semantic)
		}
		id, err := workspace.RootPackageID()
		if err != nil {
			t.Fatal(err)
		}
		out, err := interpreter.Execute(compiled.MIR, hir.QualifiedModulePath(id, "main")+"::main")
		if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != int32(edit.value) {
			t.Fatalf("Go %s value: %+v %v", edit.name, out, err)
		}
		t.Logf("Go %s result=%d provider-sha256=%x main-sha256=%x", edit.name, out.Value, sha256.Sum256([]byte(edit.provider)), sha256.Sum256([]byte(edit.main)))
	}
	valid := edits[len(edits)-1].main
	for _, refusal := range compositeResultCLIRefusals() {
		writeTestSource(t, root, "src/main.fy", valid+refusal.extra)
		if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSourceDiagnostics || !strings.Contains(diagnostic, refusal.code) {
			t.Fatalf("Go %s: %d %s", refusal.name, code, diagnostic)
		}
		t.Logf("Go refusal=%s expected=%s", refusal.name, refusal.code)
	}
}

// Observe private scalar C entry points in the fresh companion packet. Ordinary
// CLI build/run still use the original, unmodified product packet and main.
func observeCompositeResultCLIPacket(t *testing.T, path string, second bool, errorCode uint32, exitCode int, profile cdriver.Profile) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := candidatebuild.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	var observer strings.Builder
	observer.WriteString("#include \"fy_project.h\"\n#include <limits.h>\n#include <string.h>\nint fairyc_result_observe(void){\n")
	observations := 0
	for _, v := range genericCLIValues {
		pattern := `(?m)^(?:extern )?` + regexp.QuoteMeta(v.c) + ` (fy_fn_[0-9]+)\(` + regexp.QuoteMeta(v.c) + ` [^,()]+,\s*` + regexp.QuoteMeta(v.c) + ` [^,()]+,\s*int32_t [^,()]+\);`
		found := regexp.MustCompile(pattern).FindAllStringSubmatch(packet.Header, -1)
		if len(found) != 1 {
			t.Fatalf("expected unique %s Result scalar probe, got %d", v.fairy, len(found))
		}
		values := [][2]string{{v.first, v.second}}
		if v.fairy == "isize" {
			values[0] = [2]string{"(-INTPTR_MAX-1)", "INTPTR_MAX"}
		} else if v.fairy == "f32" {
			values = append(values, [2]string{"0.0f", "0.0f"}, [2]string{"-0.0f", "-0.0f"}, [2]string{"1.25f", "1.25f"})
		} else if v.fairy == "f64" {
			values = append(values, [2]string{"0.0", "0.0"}, [2]string{"-0.0", "-0.0"}, [2]string{"-3.5", "-3.5"})
		}
		for _, pair := range values {
			want := pair[0]
			if second {
				want = pair[1]
			}
			for marker := 0; marker < 2; marker++ {
				fmt.Fprintf(&observer, "{%s actual=%s((%s)(%s),(%s)(%s),%d);%s expected=(%s)(%s);if(memcmp(&actual,&expected,sizeof expected)!=0)return 1;}\n", v.c, found[0][1], v.c, pair[0], v.c, pair[1], marker, v.c, v.c, want)
				observations++
			}
		}
	}
	errorEntry := regexp.MustCompile(`(?m)^(?:extern )?uint32_t (fy_fn_[0-9]+)\(int32_t [^,()]+\);`).FindAllStringSubmatch(packet.Header, -1)
	if len(errorEntry) != 1 {
		t.Fatalf("expected unique u32 error probe, got %d", len(errorEntry))
	}
	fmt.Fprintf(&observer, "if(%s(0)!=UINT32_C(%d))return 2;return 0;}\n", errorEntry[0][1], errorCode)
	packet.Header += "\nint fairyc_result_observe(void);\n"
	entry := regexp.MustCompile(`\bint main\([^)]*\)\s*\{`)
	changed := 0
	for i, unit := range packet.Units {
		changed += len(entry.FindAllStringIndex(unit, -1))
		packet.Units[i] = entry.ReplaceAllString(unit, "${0}if(fairyc_result_observe()!=0)return 99;")
	}
	if changed != 1 {
		t.Fatalf("expected one C main, got %d", changed)
	}
	rt, err := cruntime.ScalarSource(128)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsMSVCTargetParts(tc, profile, packet.Header, packet.Literals, packet.Units, rt+"\n"+observer.String())
	if err != nil {
		t.Fatal(err)
	}
	out, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
	if err != nil || out.ExitCode != int32(exitCode) || out.Stdout != "" || out.Stderr != "" {
		t.Fatalf("Result scalar observer: %+v %v", out, err)
	}
	t.Logf("observer profile=%s packet-sha256=%x executable-sha256=%x active-payloads=%d both-copies second=%t error=%d exit=%d; source counters assert ? success+error once", profile.Name, sha256.Sum256(raw), sha256.Sum256(artifact.Bytes), observations, second, errorCode, exitCode)
}

func compositeResultCLIAcceptance(t *testing.T, image, artifacts string, full bool) {
	t.Helper()
	root, err := os.MkdirTemp(artifacts, "workspace-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, os.DirFS(compositeResultCLIWorkspace(t))); err != nil {
		t.Fatal(err)
	}
	packet := filepath.Join(root, "observed.bin")
	t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
	command := copiedClosureCommand{t: t, image: image, root: root}
	t.Logf("retained Result workspace=%s full-vectors=%t", root, full)
	groupedStaticCLIExact(t, &command, 0, "cache: cold stored", "check")
	current := copiedClosureCacheSnapshot(t, root)
	selected, ledger := groupedStaticCLIProject(t, root, nil, current)
	keys := groupedStaticCLILedger(t, ledger)
	groupedStaticCLIWarm(t, &command, 42)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		flags := []string{}
		if profile.Name == "optimized" {
			flags = append(flags, "--release")
		}
		if out := groupedStaticCLIExact(t, &command, 0, "cache: hit", append([]string{"build", "--locked"}, flags...)...); !strings.Contains(out, "built:") {
			t.Fatal(out)
		}
		path := filepath.Join(root, "build", "result-cli.exe")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		out, err := exec.CommandContext(ctx, path).CombinedOutput()
		cancel()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 || len(out) != 0 {
			t.Fatalf("standalone Result product: %v %q", err, out)
		}
		t.Logf("product=%s sha256=%x exit42 empty streams", profile.Name, sha256.Sum256(raw))
		groupedStaticCLIExact(t, &command, 42, "cache: hit", append([]string{"run", "--locked"}, flags...)...)
		if full || profile.Name == "optimized" {
			observeCompositeResultCLIPacket(t, packet, false, 77, 42, profile)
		}
	}
	out, diagnostic := command.run(0, "test", "--list", "--json", "--locked")
	if diagnostic != "" || !strings.Contains(out, `"count":1`) || !strings.Contains(out, "result_contract") {
		t.Fatalf("Result list: %s %s", out, diagnostic)
	}
	for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
		out, diagnostic := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
		if diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
			t.Fatalf("Result test flags=%v: %s %s", flags, out, diagnostic)
		}
	}
	var valid compositeResultCLIEdit
	for _, edit := range compositeResultCLIEdits(full) {
		before := copiedClosureCacheSnapshot(t, root)
		writeTestSource(t, root, "libs/answer/src/answer.fy", edit.provider)
		writeTestSource(t, root, "src/main.fy", edit.main)
		t.Logf("edit=%s provider-sha256=%x main-sha256=%x", edit.name, sha256.Sum256([]byte(edit.provider)), sha256.Sum256([]byte(edit.main)))
		groupedStaticCLIStaleLock(t, &command)
		out, diagnostic := command.run(edit.value, "run", "--cache-root", "cache", "--cache-status")
		if out != "" || diagnostic != "cache: mixed\n" && diagnostic != "cache: cold stored\n" && diagnostic != "cache: units hit\n" {
			t.Fatalf("Result %s was not rebuilt: %q %q", edit.name, out, diagnostic)
		}
		selected, ledger = groupedStaticCLIProject(t, root, before, copiedClosureCacheSnapshot(t, root))
		next := groupedStaticCLILedger(t, ledger)
		stable := 0
		for coordinate, old := range keys {
			if value, exists := next[coordinate]; !exists {
				t.Fatal("Result edit changed package coordinate")
			} else if value == old {
				stable++
			}
		}
		wantStable := 0
		if edit.name == "payload" || edit.name == "private-body" {
			wantStable = 1
		}
		if stable != wantStable || edit.name == "private-body" && diagnostic != "cache: mixed\n" {
			t.Fatalf("Result %s stable package generic keys=%d want %d cache=%q", edit.name, stable, wantStable, diagnostic)
		}
		t.Logf("Result %s stable package generic keys=%d/2 cache=%q", edit.name, stable, diagnostic)
		keys = next
		if full {
			groupedStaticCLIWarm(t, &command, edit.value)
		}
		observeCompositeResultCLIPacket(t, packet, edit.second, edit.errorCode, edit.value, cdriver.OptimizedProfile)
		valid = edit
	}
	good := copiedClosureCacheSnapshot(t, root)
	if err := os.WriteFile(filepath.Join(root, "cache", selected), []byte("corrupt selected Result project generation"), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic = command.run(valid.value, "run", "--locked", "--cache-root", "cache", "--cache-status")
	if out != "" || diagnostic != "cache: units hit\n" && diagnostic != "cache: mixed\n" && diagnostic != "cache: cold stored\n" || !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
		t.Fatalf("selected Result project repair: %q %q", out, diagnostic)
	}
	t.Logf("repair=%s selected=%s sha256=%x", strings.TrimSpace(diagnostic), selected, good[selected])
	if full {
		observeCompositeResultCLIPacket(t, packet, valid.second, valid.errorCode, valid.value, cdriver.OptimizedProfile)
	}
	refusals := compositeResultCLIRefusals()
	if !full {
		refusals = refusals[:1]
	}
	for _, refusal := range refusals {
		writeTestSource(t, root, "src/main.fy", valid.main+refusal.extra)
		groupedStaticCLIStaleLock(t, &command)
		out, diagnostic := command.run(ExitSourceDiagnostics, "check", "--cache-root", "cache", "--cache-status")
		if out != "" || !strings.HasPrefix(diagnostic, "cache: cold uncached\n") || !strings.Contains(diagnostic, refusal.code) || !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
			t.Fatalf("Result %s refusal/cache preservation: %q %q", refusal.name, out, diagnostic)
		}
		t.Logf("refusal=%s diagnostic=%q", refusal.name, diagnostic)
		writeTestSource(t, root, "src/main.fy", valid.main)
		groupedStaticCLIExact(t, &command, 0, "cache: hit", "check")
		if full {
			groupedStaticCLIWarm(t, &command, valid.value)
		} else {
			groupedStaticCLIExact(t, &command, 0, "cache: hit", "check", "--locked")
			if !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
				t.Fatal("low restored Result source changed cache bytes")
			}
		}
	}
	t.Logf("Result complete full-vectors=%t processes=%d standalone-products=2 retained=%s", full, command.calls, root)
}

func TestCandidateCompositeResultCLICache(t *testing.T) {
	if os.Getenv("FAIRYC_RESULT_CLI_MAIN_GATE") != "passed" {
		t.Skip("wait for parent main-source Result gate before full host compilation")
	}
	profiles := copiedClosureCLIProfiles(t)
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	cache, err := filepath.Abs(filepath.Join("..", "..", ".cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	attempt := os.Getenv("FAIRYC_RESULT_CLI_REUSE_ARTIFACTS")
	if attempt == "" {
		attempt, err = os.MkdirTemp(cache, "flat-result-cli-")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		relative, err := filepath.Rel(cache, attempt)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			t.Fatal("Result host reuse must stay inside main cache", err)
		}
	}
	t.Setenv("FAIRYC_CLOSURE_CLI_ARTIFACTS", attempt)
	t.Logf("unique Result artifacts=%s", attempt)
	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			started := time.Now()
			image := copiedClosureCLIHost(t, profile)
			receipt, err := os.ReadFile(filepath.Join(filepath.Dir(image), "host.json"))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Result host profile=%s preparation=%s identity=%s", profile.Name, time.Since(started), receipt)
			installGenericCLIPacketCapture(t, image)
			compositeResultCLIAcceptance(t, image, attempt, profile.Name == "optimized")
			t.Logf("Result profile=%s total elapsed=%s", profile.Name, time.Since(started))
		})
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Result native test executable=%s", identity.String())
}
