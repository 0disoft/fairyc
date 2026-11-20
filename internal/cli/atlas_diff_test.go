package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestAtlasDiffComparesLockedFacts(t *testing.T) {
	const base = "module main; import answer as answer;\n" +
		"fn i32 local() { return 1; }\n" +
		"fn i32 removed() { return 0; }\n" +
		"fn i32 main() { return answer::value(); }\n"
	const update = "module main; import answer as answer;\n\n" +
		"pub fn i32 local() { return 1; }\n" +
		"fn i32 added() { return 0; }\n" +
		"fn i32 main() { return local(); }\n"
	before := writeWorkspaceSessionFixture(t, base)
	shifted := writeWorkspaceSessionFixture(t, "\n"+base)
	after := writeWorkspaceSessionFixture(t, update)
	for _, root := range []string{before, shifted, after} {
		if code, _, diag := runForTest([]string{"check"}, root); code != ExitSuccess {
			t.Fatalf("prepare locked fixture: %d %s", code, diag)
		}
	}
	args := []string{"experimental", "atlas-diff", "--baseline", before,
		"--package-uid", "example.session.app", "--max-declarations", "16", "--max-output-bytes", "100000"}
	beforeFiles, afterFiles := remoteTree(t, before), remoteTree(t, after)
	read := func(root string) atlasmap.DiffResult {
		t.Helper()
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || strings.Contains(stdout, root) || strings.Contains(stdout, before) {
			t.Fatalf("atlas-diff: %d stdout=%q stderr=%q", code, stdout, stderr)
		}
		var report atlasmap.DiffResult
		if err := json.Unmarshal([]byte(stdout), &report); err != nil || report.Schema != "fairyc.atlas-diff/v1" {
			t.Fatalf("decode %s: %v", stdout, err)
		}
		return report
	}
	if got := read(shifted); len(got.Changes) != 0 || got.Before.PackageInstance == got.After.PackageInstance {
		t.Fatalf("position-only changes affected comparison: %+v", got)
	}
	want := []atlasmap.DiffChange{
		{Module: "main", Kind: "function", Name: "added", Action: "added"},
		{Module: "main", Kind: "function", Name: "local", Action: "modified", Facts: []string{"declaration"}},
		{Module: "main", Kind: "function", Name: "main", Action: "modified", Facts: []string{"calls"}},
		{Module: "main", Kind: "function", Name: "removed", Action: "removed"},
	}
	if got := read(after); !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("changes = %+v, want %+v", got.Changes, want)
	}
	for _, flag := range []string{"--max-declarations", "--max-output-bytes"} {
		capped := replaceWorkspaceSessionArgument(args, flag, "1")
		if code, stdout, stderr := runForTest(capped, after); code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "limit") {
			t.Fatalf("%s: %d stdout=%q stderr=%q", flag, code, stdout, stderr)
		}
	}
	if !reflect.DeepEqual(remoteTree(t, before), beforeFiles) || !reflect.DeepEqual(remoteTree(t, after), afterFiles) {
		t.Fatal("Atlas diff changed a workspace")
	}
	writeWorkspaceFile(t, before, "src/main.fy", base+"\n")
	if code, stdout, _ := runForTest(args, after); code != ExitPackageFailure || stdout != "" {
		t.Fatalf("stale baseline: %d stdout=%q", code, stdout)
	}
}
