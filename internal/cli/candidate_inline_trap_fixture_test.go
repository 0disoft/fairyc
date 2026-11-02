package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

var inlineTrapCases = []struct{ name, expression, category string }{
	{"divzero", "divide(10,0)", "division-by-zero"},
	{"signed", "divide(-2147483648,-1)", "signed-division-overflow"},
	{"overflow", "add(2147483647)", "integer-overflow"},
}

func inlineTrapCall(name, expression string, generic bool) (declaration, call string) {
	if generic {
		return "fn i32 " + name + "<T>(T marker){discard marker;return a::" + expression + ";}", name + "<bool>(true)"
	}
	return "", "a::" + expression
}

func inlineTrapWorkspace(t *testing.T, index int, generic bool) string {
	t.Helper()
	root := inlineCLIWorkspace(t)
	item := inlineTrapCases[index]
	decl, call := inlineTrapCall("invoke", item.expression, generic)
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+decl+"fn i32 main(){return "+call+";}")
	var tests strings.Builder
	tests.WriteString("module tests::inline;import answer as a;")
	for i, item := range inlineTrapCases {
		decl, call := inlineTrapCall("invoke_"+item.name, item.expression, generic)
		tests.WriteString(decl)
		fmt.Fprintf(&tests, "@test fn void case_%d_%s(){discard %s;}", i, item.name, call)
	}
	tests.WriteString("@test fn void case_3_after(){assert(a::add(40)==42);}")
	writeTestSource(t, root, "tests/inline.fy", tests.String())
	return root
}

func checkInlineTrapDiagnostic(t *testing.T, diagnostic, category string) {
	t.Helper()
	var line, column int
	prefix := "trap: " + category + "\n"
	_, err := fmt.Sscanf(strings.TrimPrefix(diagnostic, prefix), "  at libs/answer/src/answer.fy:%d:%d\n", &line, &column)
	wantLine := 5
	if category == "integer-overflow" {
		wantLine = 3
	}
	want := fmt.Sprintf("%s  at libs/answer/src/answer.fy:%d:%d\n", prefix, line, column)
	if err != nil || line != wantLine || column <= 0 || diagnostic != want {
		t.Fatalf("missing inline-provider trap category/location: %q", diagnostic)
	}
}

func checkInlineTrapReport(t *testing.T, output string, native, release bool) {
	t.Helper()
	var report stableTestExecutionReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatal(err, output)
	}
	engine := "interpreter"
	if native {
		engine = "native"
	}
	if report.Engine != engine || report.SchemaVersion != "fairyc.test-report/v1" || report.Count != 4 || report.Failed != 3 || report.Passed != 1 || report.Status != "failed" || report.Lock != "verified" || len(report.Tests) != 4 {
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
	for i, item := range inlineTrapCases {
		row := report.Tests[i]
		if row.Identity != fmt.Sprintf("tests::inline::case_%d_%s", i, item.name) || row.Outcome != "failed" || row.TrapCategory == nil || *row.TrapCategory != item.category {
			t.Fatal(output)
		}
	}
	last := report.Tests[3]
	if last.Identity != "tests::inline::case_3_after" || last.Outcome != "passed" || last.TrapCategory != nil {
		t.Fatal("test after traps was not executed", output)
	}
}

type inlineTrapOracle struct {
	diagnostics [2][3]string
	reports     [2][3]string
}

var inlineTrapTestFlags = [][]string{nil, {"--native"}, {"--native", "--release"}}

func inlineTrapReference(t *testing.T) inlineTrapOracle {
	t.Helper()
	var oracle inlineTrapOracle
	for family := 0; family < 2; family++ {
		var last string
		for i, item := range inlineTrapCases {
			root := inlineTrapWorkspace(t, i, family == 1)
			code, out, diagnostic := runForTest([]string{"run"}, root)
			if code != ExitProgramTrap || out != "" {
				t.Fatalf("Go inline trap: exit=%d stdout=%q stderr=%q", code, out, diagnostic)
			}
			checkInlineTrapDiagnostic(t, diagnostic, item.category)
			oracle.diagnostics[family][i] = diagnostic
			last = root
		}
		for i, flags := range inlineTrapTestFlags {
			outCode, out, diagnostic := runForTest(append([]string{"test", "--json", "--locked"}, flags...), last)
			if outCode != ExitProgramTrap || diagnostic != "" {
				t.Fatalf("Go inline test traps: exit=%d stdout=%q stderr=%q", outCode, out, diagnostic)
			}
			checkInlineTrapReport(t, out, i != 0, i == 2)
			oracle.reports[family][i] = out
		}
	}
	return oracle
}

func TestCandidateInlineTrapReference(t *testing.T) {
	inlineTrapReference(t)
}
