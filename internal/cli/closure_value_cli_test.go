package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

// Original source goes through the manifest loader and Go CLI. No candidate
// host, synthetic lifted metadata, AST/HIR patch, or replacement lowerer is used.
const closureValueCLIProvider = `module answer;
pub struct Holder{fn i32(i32) action;}
pub fn fn i32(i32) make(i32 seed){
 var i32 a=seed;let i32 b=3;let i32 c=2;let i32 d=4;
 let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){let reduced=a-b*x;return reduced+c-d;};
 a=999;return action;
}
pub fn fn i32(i32) other(){let fn i32(i32) action=fn [] i32(i32 x){return x-1;};return action;}
pub fn fn i32(i32) trapping(){let fn i32(i32) action=fn [] i32(i32 x){let unused=1/0;return x;};return action;}
pub fn Holder boxed(i32 seed){let action=make(seed);return Holder{.action=action};}
pub fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
`

const closureValueCLIMain = `module main;import answer as a;
fn i32 main(){
 let original=a::make(50);let copy=original;let holder=a::boxed(50);let stored=holder;let extracted=stored.action;
 let different=a::other();let unused=a::trapping();let separate=a::make(60);
 assert(a::apply(different,43)==42);assert(a::apply(copy,2)==42);assert(a::apply(separate,2)==52);
 var i32 count=0;let result=extracted({count=count+1;2});assert(count==1);assert(result==42);return result;
}`

const closureValueCLITests = `module tests::closure;import answer as a;
@test fn void a_body_trap(){let action=a::trapping();discard a::apply(action,42);}
@test fn void b_argument_first(){let action=a::make(-2147483648);discard a::apply(action,1/0);}
@test fn void c_overflow(){let action=a::make(-2147483648);discard a::apply(action,1);}
@test fn void z_values(){
 let original=a::make(50);let copy=original;let holder=a::boxed(50);let stored=holder;let extracted=stored.action;
 let other=a::other();let unused=a::trapping();let separate=a::make(60);
 assert(a::apply(copy,2)==42);assert(a::apply(other,43)==42);assert(a::apply(separate,2)==52);
 var i32 count=0;assert(extracted({count=count+1;2})==42);assert(count==1);
}
`

func closureValueCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, closureValueCLIMain)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", closureValueCLIProvider)
	writeWorkspaceFile(t, root, "tests/closure.fy", closureValueCLITests)
	t.Logf("workspace=%s original provider=%x main=%x tests=%x", root, sha256.Sum256([]byte(closureValueCLIProvider)), sha256.Sum256([]byte(closureValueCLIMain)), sha256.Sum256([]byte(closureValueCLITests)))
	return root
}

func closureValueCLICheck(t *testing.T, root string) {
	t.Helper()
	code, out, diag := runForTest([]string{"check"}, root)
	if code != ExitSuccess || !strings.HasPrefix(out, "check ok:") || diag != "" {
		t.Fatalf("check: (%d, %q, %q)", code, out, diag)
	}
}

func closureValueCLITestReport(t *testing.T, root string, native, release, successOnly bool) {
	t.Helper()
	args := []string{"test", "--json", "--locked"}
	engine, profile := "interpreter", "development"
	if native {
		args = append(args, "--native", "--capture-output")
		engine = "native"
	}
	if release {
		args = append(args, "--release")
		profile = "release"
	}
	wantCode, count, passed, failed, status := ExitProgramTrap, 4, 1, 3, "failed"
	if successOnly {
		args = append(args, "--exact", "tests::closure::z_values")
		wantCode, count, passed, failed, status = ExitSuccess, 1, 1, 0, "ok"
	}
	t.Logf("invoking %v", args)
	code, out, diag := runForTest(args, root)
	var report stableTestExecutionReport
	if native {
		report = portableNativeReport(t, code, wantCode, out, diag, profile, count)
	} else if code != wantCode || diag != "" {
		t.Fatalf("%v: (%d, %q, %q)", args, code, out, diag)
	} else if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	wantSchema := stableTestReportSchema
	if native {
		wantSchema = stableCapturedTestReportSchema
	}
	if report.SchemaVersion != wantSchema || report.Engine != engine || report.Count != count || len(report.Tests) != count || report.Passed != passed || report.Failed != failed || report.Status != status || report.Lock != "verified" {
		t.Fatalf("test report: %+v", report)
	}
	if native {
		if report.Target == nil || *report.Target != string(nativeWorkspaceTarget()) || report.Profile == nil || *report.Profile != profile {
			t.Fatalf("native identity: %+v", report)
		}
	} else if report.Target != nil || report.Profile != nil {
		t.Fatalf("interpreter identity: %+v", report)
	}
	categories := []string{"division-by-zero", "division-by-zero", "integer-overflow", ""}
	names := []string{"a_body_trap", "b_argument_first", "c_overflow", "z_values"}
	if successOnly {
		categories, names = categories[3:], names[3:]
	}
	for i, outcome := range report.Tests {
		if native && (outcome.StdoutBase64 == nil || *outcome.StdoutBase64 != "" || outcome.StderrBase64 == nil || *outcome.StderrBase64 != "") {
			t.Fatalf("native test leaked program/trap streams: %+v", outcome)
		}
		wantOutcome := "failed"
		if categories[i] == "" {
			wantOutcome = "passed"
		}
		if outcome.Identity != "tests::closure::"+names[i] || outcome.Outcome != wantOutcome || categories[i] == "" && outcome.TrapCategory != nil || categories[i] != "" && (outcome.TrapCategory == nil || *outcome.TrapCategory != categories[i]) {
			t.Fatalf("ordered test %d: %+v", i, outcome)
		}
	}
	t.Logf("%v: status=%d %s", args, code, strings.TrimSpace(out))
}

func TestClosureValueCLIStableSource(t *testing.T) {
	root := closureValueCLIWorkspace(t)
	closureValueCLICheck(t, root)
	code, out, diag := runForTest([]string{"check", "--locked"}, root)
	if code != ExitSuccess || !strings.HasPrefix(out, "check ok:") || diag != "" {
		t.Fatalf("locked check: (%d, %q, %q)", code, out, diag)
	}
	closureValueCLITestReport(t, root, false, false, true)
	closureValueCLITestReport(t, root, false, false, false)
}

func TestClosureValueCLIWindowsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 MSVC acceptance; Linux is batched separately")
	}
	root := closureValueCLIWorkspace(t)
	closureValueCLICheck(t, root)
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		profile := "low"
		if release {
			profile = "optimized"
		}
		t.Run(profile, func(t *testing.T) {
			for _, command := range []string{"build", "run"} {
				t.Run(command, func(t *testing.T) {
					args := []string{command, "--locked"}
					if release {
						args = append(args, "--release")
					}
					code, out, diag := runForTest(args, root)
					if command == "build" {
						if code != ExitSuccess || !strings.HasPrefix(out, "built: build/app.exe (x86_64-windows-msvc, ") || diag != "" {
							t.Fatalf("%v: (%d, %q, %q)", args, code, out, diag)
						}
						image, err := os.ReadFile(filepath.Join(root, "build", "app.exe"))
						if err != nil || len(image) < 2 || string(image[:2]) != "MZ" {
							t.Fatal("build did not publish a PE image", err)
						}
						t.Logf("%v: status=%d PE-sha256=%x", args, code, sha256.Sum256(image))
					} else if code != 42 || out != "" || diag != "" {
						t.Fatalf("%v: (%d, %q, %q)", args, code, out, diag)
					} else {
						t.Logf("%v: original program status=%d, empty streams", args, code)
					}
				})
			}
			t.Run("test", func(t *testing.T) {
				started := time.Now()
				defer func() {
					deadline, bounded := t.Deadline()
					t.Logf("native test wall=%s context-error=%v outer-deadline=%s bounded=%v remaining=%s", time.Since(started), t.Context().Err(), deadline.Format(time.RFC3339Nano), bounded, time.Until(deadline))
				}()
				// Preserve execution assertions when isolating this command.
				closureValueCLITestReport(t, root, true, release, false)
			})
		})
	}
	practicalFileEquals(t, filepath.Join(root, "fairy.lock"), string(lock))
}

func TestClosureValueCLIWindowsRunTraps(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 MSVC acceptance; Linux is batched separately")
	}
	for _, tc := range []struct{ name, body string }{
		{"body-unused-local", "let action=a::trapping();return a::apply(action,42);"},
		{"argument-before-overflow", "let action=a::make(-2147483648);return a::apply(action,1/0);"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := closureValueCLIWorkspace(t)
			writeWorkspaceFile(t, root, "src/main.fy", "module main;import answer as a;fn i32 main(){"+tc.body+"}")
			closureValueCLICheck(t, root)
			for _, release := range []bool{false, true} {
				args := []string{"run", "--locked"}
				if release {
					args = append(args, "--release")
				}
				code, out, diag := runForTest(args, root)
				if code != ExitProgramTrap || out != "" || !strings.HasPrefix(diag, "trap: division-by-zero\n  at ") || !strings.HasSuffix(diag, "\n") {
					t.Fatalf("%v trap: (%d, %q, %q)", args, code, out, diag)
				}
				t.Logf("%v: original status=%d %s", args, code, strings.TrimSpace(diag))
			}
		})
	}
}

func closureValueCLICache(t *testing.T, root string, hit bool) workspaceCacheReport {
	t.Helper()
	args := append([]string{"experimental", "check-workspace-cache"}, workspaceCacheArguments()...)
	args = append(args, "--locked")
	code, out, diag := runForTest(args, root)
	if code != ExitSuccess || diag != "" {
		t.Fatalf("cache: (%d, %q, %q)", code, out, diag)
	}
	r := decodeWorkspaceCacheReport(t, out)
	if r.SchemaVersion != workspaceCacheSchema || r.Cache.Storage != workspacecompiler.StorageDisk || r.Cache.Packages != 2 || r.Cache.StoreFailed != 0 || r.Cache.RetainedEntries != 2 || r.Instances != 2 || r.SourceFiles != 2 || r.MIRFunctions < 9 {
		t.Fatalf("cache report: %+v", r)
	}
	if hit {
		if r.Cache.Outcome != workspacecompiler.OutcomeAllHit || r.Cache.Hits != 2 || r.Cache.VerifiedHits != 2 || r.Cache.Misses != 0 || r.Cache.Lowered || r.Cache.Stored != 0 || r.Cache.ReassemblyFallback {
			t.Fatalf("not a restored all-hit: %+v", r.Cache)
		}
	} else if r.Cache.Outcome != workspacecompiler.OutcomeFallback || r.Cache.Hits != 0 || r.Cache.Misses != 2 || !r.Cache.Lowered || r.Cache.Stored != 2 {
		t.Fatalf("not a cold two-package publication: %+v", r.Cache)
	}
	t.Logf("persistent CLI: %s", strings.TrimSpace(out))
	return r
}

func closureValueCLIGenerations(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	records := map[string][32]byte{}
	err := filepath.WalkDir(filepath.Join(root, "build", "fairyc-cache-test", "v1"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			records[path] = sha256.Sum256(data)
		}
		return nil
	})
	if err != nil || len(records) != 2 {
		t.Fatalf("actual two-package disk seed: %v %v", records, err)
	}
	return records
}

func TestClosureValueCLIPersistentCache(t *testing.T) {
	root := closureValueCLIWorkspace(t)
	closureValueCLICheck(t, root)
	cold := closureValueCLICache(t, root, false)
	before := closureValueCLIGenerations(t, root)
	warm := closureValueCLICache(t, root, true)
	if cold.CompilerBuild != warm.CompilerBuild {
		t.Fatal("compiler cache identity changed")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestClosureValueCLICacheChild$", "-test.count=1", "-test.v")
	command.Env = append(os.Environ(), "FAIRYC_CLOSURE_CLI_CHILD="+root, "FAIRYC_CLOSURE_CLI_BUILD="+cold.CompilerBuild)
	output, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("closure-value-cli-new-process-ok")) {
		t.Fatalf("new process: %v\n%s", err, output)
	}
	t.Logf("new process: %s", output)
	if !reflect.DeepEqual(before, closureValueCLIGenerations(t, root)) {
		t.Fatal("warm/new-process hit rewrote retained generation bytes")
	}
	// Corrupt an actual generation. Repair must be visible, then return to hits.
	for path := range before {
		if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		break
	}
	args := append([]string{"experimental", "check-workspace-cache"}, workspaceCacheArguments()...)
	code, out, diag := runForTest(append(append([]string{}, args...), "--locked"), root)
	if code != ExitSuccess || diag != "" {
		t.Fatalf("corruption recovery: (%d, %q, %q)", code, out, diag)
	}
	repaired := decodeWorkspaceCacheReport(t, out)
	if repaired.CompilerBuild != cold.CompilerBuild || repaired.Cache.Outcome != workspacecompiler.OutcomeFallback || repaired.Cache.Hits != 1 || repaired.Cache.Misses != 1 || repaired.Cache.RecordRejected == 0 || !repaired.Cache.Lowered || repaired.Cache.Stored != 2 || repaired.Cache.StoreFailed != 0 || repaired.Cache.RetainedEntries != 2 {
		t.Fatalf("corrupt generation was not repaired: %+v", repaired)
	}
	t.Logf("corruption repair: %s", strings.TrimSpace(out))
	closureValueCLICache(t, root, true)
	before = closureValueCLIGenerations(t, root)
	// Invalid original source cannot execute or replace previously good records.
	writeWorkspaceFile(t, root, "src/main.fy", "module main;fn i32 main(){return true;}")
	code, out, diag = runForTest(args, root)
	if code != ExitSourceDiagnostics || out != "" || !strings.Contains(diag, "E1009") {
		t.Fatalf("invalid source cache refusal: (%d, %q, %q)", code, out, diag)
	}
	if !reflect.DeepEqual(before, closureValueCLIGenerations(t, root)) {
		t.Fatal("invalid source changed good disk records")
	}
	writeWorkspaceFile(t, root, "src/main.fy", closureValueCLIMain)
	closureValueCLICheck(t, root) // Refresh only the disposable fixture lock.
	closureValueCLICache(t, root, true)
}

func TestClosureValueCLICacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_CLOSURE_CLI_CHILD")
	if root == "" {
		t.Skip("launched only by persistent-cache parent")
	}
	r := closureValueCLICache(t, root, true)
	if r.CompilerBuild != os.Getenv("FAIRYC_CLOSURE_CLI_BUILD") {
		t.Fatal("child did not use the seeded compiler identity")
	}
	closureValueCLITestReport(t, root, false, false, true)
	t.Log("closure-value-cli-new-process-ok")
}

func TestClosureValueCLIRefusals(t *testing.T) {
	const literal = "fn [] i32(i32 x){return x;}"
	for _, tc := range []struct{ name, source, diagnostic string }{
		{"mutable", "fn i32 main(){var fn i32(i32) action=" + literal + ";return action(1);}", "unique immutable let"},
		{"reassign", "fn i32 main(){let fn i32(i32) action=" + literal + ";action=action;return 0;}", "E1008"},
		{"signature", "fn void take(fn bool(i32) action){discard action;}fn i32 main(){return 0;}", "type fn bool(i32)"},
		{"field-invoke", "struct Box{fn i32(i32) action;}fn i32 take(Box box){return box.action(1);}fn i32 main(){return 0;}", "unknown method action"},
		{"wrapper", "fn void take(Option<fn i32(i32)> action){discard action;}fn i32 main(){return 0;}", "type Option<fn i32(i32)>"},
		{"reprC", "@repr(C) struct Box{fn i32(i32) action;}fn i32 main(){return 0;}", "repr(C) struct containing a closure value"},
		{"foreign", "unsafe extern C fn void take(fn i32(i32) action);fn i32 main(){return 0;}", "foreign closure-bearing parameter"},
		{"pointer-carrier", "struct Box{fn i32(i32) action;}fn void take(ptr Box box){discard box;}fn i32 main(){return 0;}", "type ptr Box"},
		{"capture-closure", "fn i32 main(){let fn i32(i32) action=" + literal + ";let fn i32(i32) other=fn [i32 action] i32(i32 x){return action+x;};return 0;}", "closure values cannot be captured"},
		{"five-captures", "fn i32 main(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let i32 e=5;let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d,i32 e] i32(i32 x){return x;};return 0;}", "copied closure requires 0..4"},
		{"seventeen-locals", "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){" + closureValueCLITooManyLocals() + "return x;};return 0;}", "copied closure requires 0..4"},
		{"sixty-five-nodes", "fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let a=1;let b=2;return " + strings.Repeat("x+", 31) + "x;};return 0;}", "at most 64 nodes"},
		{"generic-transport", "fn fn i32(i32) forward<T>(fn i32(i32) action,T marker){discard marker;return action;}fn i32 main(){let fn i32(i32) action=" + literal + ";let copy=forward<i32>(action,1);return copy(2);}", "monomorphic source function"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := closureValueCLIWorkspace(t)
			writeWorkspaceFile(t, root, "src/main.fy", "module main;"+tc.source)
			for _, command := range []string{"check", "build", "run", "test"} {
				code, out, diag := runForTest([]string{command}, root)
				if code != ExitSourceDiagnostics || out != "" || !strings.Contains(diag, tc.diagnostic) {
					t.Fatalf("%s original refusal: (%d, %q, %q)", command, code, out, diag)
				}
				t.Logf("%s: original status=%d %s", command, code, strings.TrimSpace(diag))
			}
			if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
				t.Fatalf("refused source created build output: %v", err)
			}
		})
	}
}

func closureValueCLITooManyLocals() string {
	var body strings.Builder
	for i := 0; i < 17; i++ {
		fmt.Fprintf(&body, "let a%d=x;", i)
	}
	return body.String()
}
