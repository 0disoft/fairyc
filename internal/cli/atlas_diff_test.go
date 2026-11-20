package cli

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestAtlasDiffWorkspace(t *testing.T) {
	const source = "module main; fn i32 main() { return 7; }\n"
	baseline := writeWorkspaceSessionFixture(t, source)
	current := writeWorkspaceSessionFixture(t, source)
	writeWorkspaceFile(t, current, "libs/answer/src/answer.fy",
		"\nmodule answer; pub fn i32 value() { return 43; }\n")
	standalone := writeWorkspaceSessionFixture(t, "\n"+source)
	manifest, _, _ := strings.Cut(workspaceSessionRootManifest, "[workspace]")
	writeWorkspaceFile(t, standalone, "fairy.toml", manifest)
	for _, root := range []string{baseline, current, standalone} {
		if code, _, stderr := runForTest([]string{"check"}, root); code != ExitSuccess {
			t.Fatalf("check: %d %s", code, stderr)
		}
	}
	args := []string{"experimental", "atlas-diff", "--baseline", baseline, "--workspace",
		"--max-packages", "4", "--max-declarations", "4", "--max-output-bytes", "100000"}
	read := func(args []string, root string) map[string]atlasmap.WorkspacePackageDiff {
		t.Helper()
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" || strings.Contains(stdout, root) || strings.Contains(stdout, baseline) {
			t.Fatalf("workspace diff: %d stdout=%q stderr=%q", code, stdout, stderr)
		}
		var report atlasmap.WorkspaceDiffResult
		if err := json.Unmarshal([]byte(stdout), &report); err != nil ||
			report.Schema != "fairyc.atlas-workspace-diff/v1" || len(report.Packages) != 2 {
			t.Fatalf("decode %s: %v", stdout, err)
		}
		result := make(map[string]atlasmap.WorkspacePackageDiff, len(report.Packages))
		for i, item := range report.Packages {
			if i != 0 && report.Packages[i-1].Coordinate >= item.Coordinate {
				t.Fatal("package rows are not sorted by coordinate")
			}
			context := item.After
			if context == nil {
				context = item.Before
			}
			result[context.PackageUID] = item
		}
		return result
	}
	beforeFiles, currentFiles := remoteTree(t, baseline), remoteTree(t, current)
	const app, answer = "example.session.app", "example.session.answer"
	rows := read(args, current)
	if rows[app].Action != "unchanged" || !slices.Equal(rows[app].BeforeDependencies, []string{rows[answer].Coordinate}) ||
		!slices.Equal(rows[app].AfterDependencies, rows[app].BeforeDependencies) {
		t.Fatalf("source identity change affected dependents: %+v", rows[app])
	}
	want := []atlasmap.DiffChange{{Module: "answer", Kind: "function", Name: "value", Action: "modified", Facts: []string{"executable-body"}}}
	if rows[answer].Action != "modified" || !reflect.DeepEqual(rows[answer].Changes, want) {
		t.Fatalf("dependency body change: %+v", rows[answer])
	}
	removed := read(args, standalone)
	if removed[answer].Action != "removed" || removed[answer].After != nil || len(removed[answer].Changes) != 1 ||
		removed[answer].Changes[0].Action != "removed" || removed[app].Action != "modified" ||
		len(removed[app].Changes) != 0 || len(removed[app].AfterDependencies) != 0 {
		t.Fatalf("package/edge removal: %+v", removed)
	}
	added := read(replaceWorkspaceSessionArgument(args, "--baseline", standalone), current)
	if added[answer].Action != "added" || added[answer].Before != nil || len(added[answer].Changes) != 1 ||
		added[answer].Changes[0].Action != "added" || added[app].Action != "modified" || len(added[app].Changes) != 0 {
		t.Fatalf("package/edge addition: %+v", added)
	}
	for flag, limit := range map[string]string{"--max-packages": "3", "--max-declarations": "3", "--max-output-bytes": "1"} {
		capped := replaceWorkspaceSessionArgument(args, flag, limit)
		if code, stdout, stderr := runForTest(capped, current); code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "limit") {
			t.Fatalf("%s: %d stdout=%q stderr=%q", flag, code, stdout, stderr)
		}
	}
	conflicting := append(slices.Clone(args), "--package-uid", app)
	if code, stdout, stderr := runForTest(conflicting, current); code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "exactly one") {
		t.Fatalf("conflicting scope: %d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !reflect.DeepEqual(remoteTree(t, baseline), beforeFiles) || !reflect.DeepEqual(remoteTree(t, current), currentFiles) {
		t.Fatal("workspace Atlas diff wrote to disk")
	}
}

func TestAtlasDiffComparesLockedFacts(t *testing.T) {
	const base = "module main; import answer as answer;\n" +
		"fn i32 local() { return 1; }\n" +
		"fn i32 removed() { return 0; }\n" +
		"fn i32 literal() { return 10; }\n" +
		"fn i32 main() { return answer::value(); }\n"
	const update = "module main; import answer as answer;\n\n" +
		"pub fn i32 local() { return 1; }\n" +
		"fn i32 added() { return 0; }\n" +
		"fn i32 literal() { return 11; }\n" +
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
		{Module: "main", Kind: "function", Name: "literal", Action: "modified", Facts: []string{"executable-body"}},
		{Module: "main", Kind: "function", Name: "local", Action: "modified", Facts: []string{"declaration"}},
		{Module: "main", Kind: "function", Name: "main", Action: "modified", Facts: []string{"calls", "executable-body"}},
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
