package cli

import (
	"bytes"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCandidatePortableTestReport(t *testing.T) {
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	inputSource, err := os.ReadFile(filepath.Join(root, "src", "project_compiler_input.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/project_compiler_input.fy", string(inputSource)+`
pub fn Input test_empty(){return Input{.sources=v::empty<syntax::Parsed>(),.owners=v::empty<usize>(),.origins=v::empty<Origin>(),.bindings=v::empty<catalog::Binding>(),.names=bytes::empty()};}
`)

	writeTestSource(t, root, "src/main.fy", candidatePortableTestReportProbe)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "portable test report", &diagnostics)
	if code != 0 {
		t.Fatalf("lower: %d %s", code, &diagnostics)
	}
	host, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	var source cbackend.Unit
	for _, id := range []target.ID{target.X8664WindowsMSVC, target.X8664LinuxGNU} {
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Profile{Target: id, Dialect: cbackend.DialectC11})
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range unit.ForeignSymbols {
			if strings.Contains(symbol, "windows_") {
				t.Fatal("Windows dependency in shared report", symbol)
			}
		}
		if id == host {
			source = unit
		}
	}
	support, err := nativesupport.Source(host, source.ForeignSymbols, nativesupport.Options{SDKStorage: true})
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveNativeToolchain(host, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: source.Source, RuntimeSource: support})
		if err != nil {
			t.Fatal(err)
		}
		result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Timeout: 30 * time.Second})
		if err != nil || result.ExitCode != 0 || result.Stderr != "" {
			t.Fatalf("report: %+v %v", result, err)
		}
		lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		if len(lines) != 2 {
			t.Fatal(result.Stdout)
		}
		for i, line := range lines {
			var report stableTestExecutionReport
			if err := json.Unmarshal([]byte(line), &report); err != nil {
				t.Fatal(err)
			}
			wantTarget := "x86_64-linux-gnu"
			if i == 1 {
				wantTarget = "x86_64-windows-msvc"
			}
			if report.Target == nil || *report.Target != wantTarget || report.Count != 2 || report.Failed != 1 || report.Passed != 1 || report.Engine != "native" || report.Lock != "verified" {
				t.Fatalf("report %s", line)
			}
			if report.Tests[0].TrapCategory == nil || *report.Tests[0].TrapCategory != "assertion-failed" || report.Tests[0].StdoutBase64 == nil || *report.Tests[0].StdoutBase64 != "AP8=" || report.Tests[1].StderrBase64 == nil || *report.Tests[1].StderrBase64 != "ZQ==" {
				t.Fatal(line)
			}
		}
	}
}

const candidatePortableTestReportProbe = `module main;
import project_test_options as options;
import native_test_result as result;
import project_compiler_input as input;
import module_catalog as catalog;
import syntax as syntax;
import std::vec as v;
import std::bytes as bytes;
fn bytes::Buffer copied(ref u8[] data){return switch(bytes::copy(data)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(console) fn i32 main(){
 var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();assert(options::test_add(ref mut argv,b"--native") && options::test_add(ref mut argv,b"--json") && options::test_add(ref mut argv,b"--capture-output"));
 let selected=switch(options::parse_test_options(ref argv)){case .ok(p):{p}case .err(_):{return 1;}};
 let projected=input::test_empty();
 var v::Vec<options::TestIdentity> tests=v::empty<options::TestIdentity>();
 assert(v::push<options::TestIdentity>(ref mut tests,options::identity(copied(b"tests::a::fails"),copied(b"tests/a.fy"),copied(b"client"),0,0)).is_ok);
 assert(v::push<options::TestIdentity>(ref mut tests,options::identity(copied(b"tests::a::passes"),copied(b"tests/a.fy"),copied(b"client"),0,0)).is_ok);
 var v::Vec<usize> chosen=v::empty<usize>();assert(v::push<usize>(ref mut chosen,0).is_ok && v::push<usize>(ref mut chosen,1).is_ok);
 var v::Vec<result::Captured> results=v::empty<result::Captured>();
 let failed=switch(result::copy(6,b"\0\xff",b"")){case .ok(p):{p}case .err(_):{return 2;}};
 let passed=switch(result::copy(0,b"",b"e")){case .ok(p):{p}case .err(_):{return 3;}};
 assert(v::push<result::Captured>(ref mut results,failed).is_ok);assert(v::push<result::Captured>(ref mut results,passed).is_ok);
 assert(options::test_native_report(ref selected,ref tests,ref chosen,ref projected,ref results,3,b"x86_64-linux-gnu")==5);
 assert(options::test_report(ref selected,ref tests,ref chosen,ref projected,ref results,3)==5);return 0;
}
`
