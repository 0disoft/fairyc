package cli

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

type inlineTrapCase struct {
	name, expression, category string
	line                       int
}

type inlineTrapFixture struct {
	cases                []inlineTrapCase
	provider, assertions string
}

var inlineArithmeticTraps = inlineTrapFixture{
	cases: []inlineTrapCase{
		{"divzero", "divide(10,0)", "division-by-zero", 5},
		{"signed", "divide(-2147483648,-1)", "signed-division-overflow", 5},
		{"overflow", "add(2147483647)", "integer-overflow", 3},
		{"zero_divzero", "zero_divide()", "division-by-zero", 8},
		{"zero_signed", "zero_signed()", "signed-division-overflow", 9},
		{"zero_overflow", "zero_add()", "integer-overflow", 10},
	},
	provider: inlineCLIProvider + `@inline pub fn i32 zero_divide(){return 10/0;}
@inline pub fn i32 zero_signed(){return -2147483648/-1;}
@inline pub fn i32 zero_add(){return 2147483647+1;}
@inline pub fn i32 zero_value(){return 40+2;}
`,
	assertions: "assert(a::zero_value()==42);assert(a::add(40)==42);",
}

func inlineTrapCall(name, expression string, generic bool) (declaration, call string) {
	if generic {
		return "fn i32 " + name + "<T>(T marker){discard marker;return a::" + expression + ";}", name + "<bool>(true)"
	}
	return "", "a::" + expression
}

func (fixture inlineTrapFixture) workspace(t *testing.T, index int, generic bool) string {
	t.Helper()
	root := inlineCLIWorkspace(t)
	writeTestSource(t, root, "libs/answer/src/answer.fy", fixture.provider)
	item := fixture.cases[index]
	decl, call := inlineTrapCall("invoke", item.expression, generic)
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+decl+"fn i32 main(){return "+call+";}")
	var tests strings.Builder
	tests.WriteString("module tests::inline;import answer as a;")
	for i, item := range fixture.cases {
		decl, call := inlineTrapCall("invoke_"+item.name, item.expression, generic)
		tests.WriteString(decl)
		fmt.Fprintf(&tests, "@test fn void case_%d_%s(){discard %s;}", i, item.name, call)
	}
	fmt.Fprintf(&tests, "@test fn void case_%d_after(){%s}", len(fixture.cases), fixture.assertions)
	writeTestSource(t, root, "tests/inline.fy", tests.String())
	return root
}

func checkInlineTrapDiagnostic(t *testing.T, diagnostic, category string, wantLine int) {
	t.Helper()
	var line, column int
	prefix := "trap: " + category + "\n"
	_, err := fmt.Sscanf(strings.TrimPrefix(diagnostic, prefix), "  at libs/answer/src/answer.fy:%d:%d\n", &line, &column)
	want := fmt.Sprintf("%s  at libs/answer/src/answer.fy:%d:%d\n", prefix, line, column)
	if err != nil || line != wantLine || column <= 0 || diagnostic != want {
		t.Fatalf("missing inline-provider trap category/location: %q", diagnostic)
	}
}

func (fixture inlineTrapFixture) checkReport(t *testing.T, output string, native, release bool) {
	t.Helper()
	var report stableTestExecutionReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatal(err, output)
	}
	engine := "interpreter"
	if native {
		engine = "native"
	}
	count := len(fixture.cases)
	if report.Engine != engine || report.SchemaVersion != "fairyc.test-report/v1" || report.Count != count+1 || report.Failed != count || report.Passed != 1 || report.Status != "failed" || report.Lock != "verified" || len(report.Tests) != count+1 {
		t.Fatal(output)
	}
	if native {
		profile := "development"
		if release {
			profile = "release"
		}
		if report.Target == nil || *report.Target != "x86_64-windows-msvc" || report.Profile == nil || *report.Profile != profile {
			t.Fatal(output)
		}
	} else if report.Target != nil || report.Profile != nil {
		t.Fatal(output)
	}
	for i, item := range fixture.cases {
		row := report.Tests[i]
		if row.Identity != fmt.Sprintf("tests::inline::case_%d_%s", i, item.name) || row.Outcome != "failed" || row.TrapCategory == nil || *row.TrapCategory != item.category {
			t.Fatal(output)
		}
	}
	last := report.Tests[count]
	if last.Identity != fmt.Sprintf("tests::inline::case_%d_after", count) || last.Outcome != "passed" || last.TrapCategory != nil {
		t.Fatal("test after traps was not executed", output)
	}
}

type inlineTrapOracle struct {
	diagnostics [2][]string
	reports     [2][3]string
}

var inlineTrapTestFlags = [][]string{nil, {"--native"}, {"--native", "--release"}}

func (fixture inlineTrapFixture) reference(t *testing.T) inlineTrapOracle {
	t.Helper()
	var oracle inlineTrapOracle
	for family := 0; family < 2; family++ {
		oracle.diagnostics[family] = make([]string, len(fixture.cases))
		var last string
		for i, item := range fixture.cases {
			root := fixture.workspace(t, i, family == 1)
			code, out, diagnostic := runForTest([]string{"run"}, root)
			if code != ExitProgramTrap || out != "" {
				t.Fatalf("Go inline trap: exit=%d stdout=%q stderr=%q", code, out, diagnostic)
			}
			checkInlineTrapDiagnostic(t, diagnostic, item.category, item.line)
			oracle.diagnostics[family][i] = diagnostic
			last = root
		}
		for i, flags := range inlineTrapTestFlags {
			outCode, out, diagnostic := runForTest(append([]string{"test", "--json", "--locked"}, flags...), last)
			if outCode != ExitProgramTrap || diagnostic != "" {
				t.Fatalf("Go inline test traps: exit=%d stdout=%q stderr=%q", outCode, out, diagnostic)
			}
			fixture.checkReport(t, out, i != 0, i == 2)
			oracle.reports[family][i] = out
		}
	}
	return oracle
}

func TestCandidateInlineTrapReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC inline trap reference; native Linux remains deferred")
	}
	inlineArithmeticTraps.reference(t)
}
