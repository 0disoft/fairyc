package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestWhyEmittedExplicitSourceSet(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main; import helpers as h; fn i32 main() { return h::get(); }")
	writeTestSource(t, root, "helpers.fy", "module helpers; pub fn i32 get() { return 42; }")
	args := []string{"experimental", "why-emitted", "main.fy", "--source", "helpers.fy", "--symbol"}
	code, stdout, stderr := runForTest(append(args, "helpers::get"), root)
	var report cbackend.EmissionExplanation
	if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(stdout), &report) != nil ||
		report.Schema != "fairyc.why-emitted/v1" || !report.Emitted || len(report.Path) != 1 ||
		report.Path[0] != (cbackend.EmissionStep{From: "main::main", To: "helpers::get", Reason: "call"}) {
		t.Fatalf("explain: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if code, stdout, stderr := runForTest(append(args, "missing"), root); code != ExitInvalidInvocation ||
		stdout != "" || !strings.Contains(stderr, "expected one internal function") {
		t.Fatalf("unknown selector: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
