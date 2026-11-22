package cli

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
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
		"resource Token { i32 value; cleanup { let i32 marker = 5; } }\n" +
		"fn i32 main() { return answer::value(); }\n"
	const update = "module main; import answer as answer;\n\n" +
		"pub fn i32 local() { return 1; }\n" +
		"fn i32 added() { return 0; }\n" +
		"fn i32 literal() { return 11; }\n" +
		"resource Token { i32 value; cleanup { let i32 marker = 6; } }\n" +
		"fn i32 main() { return local(); }\n"
	before := writeWorkspaceSessionFixture(t, base)
	shifted := writeWorkspaceSessionFixture(t, "\n"+base)
	after := writeWorkspaceSessionFixture(t, update)
	emptyCleanup := writeWorkspaceSessionFixture(t, strings.Replace(base, "cleanup { let i32 marker = 5; }", "cleanup {}", 1))
	for _, root := range []string{before, shifted, after, emptyCleanup} {
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
		if slices.Contains(report.Omissions, "executable-body-comparison-incomplete") {
			t.Fatalf("missing executable comparison: %s", stdout)
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
		{Module: "main", Kind: "resource", Name: "Token", Action: "modified", Facts: []string{"executable-body"}},
	}
	if got := read(after); !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("changes = %+v, want %+v", got.Changes, want)
	}
	want = []atlasmap.DiffChange{
		{Module: "main", Kind: "resource", Name: "Token", Action: "modified", Facts: []string{"declaration", "executable-body"}},
	}
	if got := read(emptyCleanup); !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("cleanup removal = %+v, want %+v", got.Changes, want)
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

func TestAtlasDiffConcreteGenericImplementations(t *testing.T) {
	const dependency = "module answer; pub fn i32 value<T>(T x) { let T observed = x; return 42; }"
	const source = `module main; import answer as answer;
struct Cell<T> { T value; }
resource Token<T> {
    Cell<T> payload;
    cleanup { let i32 observed = answer::value<T>(this.payload.value); let i32 marker = 5; }
}
fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; }
fn i32 main() { let Token<i32> a = make<i32>(42); return a.payload.value; }
`
	makeWorkspace := func(app, lib string) string {
		t.Helper()
		root := writeWorkspaceSessionFixture(t, app)
		writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", lib)
		if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
			t.Fatalf("prepare lock: %v", err)
		}
		return root
	}
	read := func(before, root string) map[string]atlasmap.WorkspacePackageDiff {
		t.Helper()
		args := []string{"experimental", "atlas-diff", "--baseline", before, "--workspace", "--max-packages", "4",
			"--max-declarations", "24", "--max-output-bytes", "100000"}
		code, stdout, stderr := runForTest(args, root)
		var report atlasmap.WorkspaceDiffResult
		if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(stdout), &report) != nil || len(report.Packages) != 2 ||
			strings.Contains(stdout, "_fy_generic_") {
			t.Fatalf("atlas diff: %d stdout=%q stderr=%q", code, stdout, stderr)
		}
		result := make(map[string]atlasmap.WorkspacePackageDiff, len(report.Packages))
		for _, row := range report.Packages {
			if slices.Contains(row.Omissions, "executable-body-comparison-incomplete") {
				t.Fatalf("missing concrete MIR projection for %s: %s", row.After.PackageUID, stdout)
			}
			result[row.After.PackageUID] = row
		}
		return result
	}
	const app, lib = "example.session.app", "example.session.answer"
	before := makeWorkspace(source, dependency)
	shifted := read(before, makeWorkspace("\n"+source, "\n"+dependency))
	if shifted[app].Action != "unchanged" || shifted[lib].Action != "unchanged" {
		t.Fatalf("position-only changes affected generic comparison: %+v", shifted)
	}
	changed := read(before, makeWorkspace(strings.Replace(source, "marker = 5", "marker = 6", 1), strings.Replace(dependency, "return 42", "return 43", 1)))
	for _, tc := range []struct {
		uid, module, kind, name string
	}{
		{app, "main", "resource", "Token"},
		{lib, "answer", "function", "value"},
	} {
		want := []atlasmap.DiffChange{
			{Module: tc.module, Kind: semanticsnapshot.DeclarationKind(tc.kind), Name: tc.name, Action: "modified", Facts: []string{"template-body-syntax"}},
			{Module: tc.module, Kind: semanticsnapshot.DeclarationKind(tc.kind), Name: tc.name, Action: "modified", Facts: []string{"executable-body"}, TypeArguments: []string{"i32"}},
		}
		if !reflect.DeepEqual(changed[tc.uid].Changes, want) {
			t.Fatalf("%s generic body changes = %+v, want %+v", tc.uid, changed[tc.uid].Changes, want)
		}
	}
	more := makeWorkspace(strings.Replace(source, "return a.payload.value", "let Token<u32> b = make<u32>(7); return a.payload.value", 1), dependency)
	for _, tc := range []struct{ baseline, current, action string }{{before, more, "added"}, {more, before, "removed"}} {
		rows := read(tc.baseline, tc.current)
		for _, want := range []atlasmap.DiffChange{
			{Module: "main", Kind: "function", Name: "make", Action: tc.action, TypeArguments: []string{"u32"}},
			{Module: "main", Kind: "resource", Name: "Token", Action: tc.action, TypeArguments: []string{"u32"}},
			{Module: "main", Kind: "struct", Name: "Cell", Action: tc.action, TypeArguments: []string{"u32"}},
		} {
			if !slices.ContainsFunc(rows[app].Changes, func(change atlasmap.DiffChange) bool { return reflect.DeepEqual(change, want) }) {
				t.Fatalf("missing implementation change %+v in %+v", want, rows[app].Changes)
			}
		}
	}
}

func TestAtlasDiffUninstantiatedTemplateBodies(t *testing.T) {
	const app = "example.session.app"
	const original = `module main;
fn T unused<T>(T value) { let i32 marker = 41; return value; }
resource Token<T> {
    T value;
    cleanup { let i32 marker = 51; }
}
fn i32 main() { return 0; }
`
	workspace := func(text string) string {
		t.Helper()
		root := writeWorkspaceSessionFixture(t, text)
		if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
			t.Fatal(err)
		}
		return root
	}
	baseline := workspace(original)
	initial := remoteTree(t, baseline)
	for _, tc := range []struct {
		name string
		text string
		want []atlasmap.DiffChange
	}{
		{"trivia", strings.Replace(original, "marker = 41", "marker /* a comment */ =\n 41", 1), []atlasmap.DiffChange{}},
		{"tokens", strings.NewReplacer("marker = 41", "marker = 42", "marker = 51", "marker = 52").Replace(original), []atlasmap.DiffChange{
			{Module: "main", Kind: "function", Name: "unused", Action: "modified", Facts: []string{"template-body-syntax"}},
			{Module: "main", Kind: "resource", Name: "Token", Action: "modified", Facts: []string{"template-body-syntax"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := workspace(tc.text)
			tree := remoteTree(t, current)
			args := []string{"experimental", "atlas-diff", "--baseline", baseline, "--package-uid", app,
				"--max-declarations", "24", "--max-output-bytes", "100000"}
			code, out, stderr := runForTest(args, current)
			var report atlasmap.DiffResult
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &report) != nil ||
				report.State != "partial" || slices.Contains(report.Omissions, "template-body-syntax-comparison-incomplete") ||
				!reflect.DeepEqual(report.Changes, tc.want) {
				t.Fatalf("template diff: %d stdout=%q stderr=%q", code, out, stderr)
			}
			if !slices.ContainsFunc(report.After.Completeness, func(fact semanticsnapshot.Completeness) bool {
				return fact.Relation == "generic-template-bodies" && fact.Status == semanticsnapshot.CompletenessUnavailable
			}) {
				t.Fatal("template comparison lost semantic coverage limitations")
			}
			if !reflect.DeepEqual(initial, remoteTree(t, baseline)) || !reflect.DeepEqual(tree, remoteTree(t, current)) {
				t.Fatal("template diff modified a workspace")
			}
		})
	}
}

func TestAtlasDiffDenyNewEffect(t *testing.T) {
	const header = `module main;
@effects(filesystem)
fn i32 touch() { return 1; }
@effects(filesystem)
resource Token { i32 value; cleanup { let i32 marker = touch(); } }
fn i32 main() { return 0; }
`
	const parser = "fn i32 decode() { return 7; }"
	const newEffect = "denied new effect \"filesystem\" on \"main::decode\""
	checked := func(text string) string {
		t.Helper()
		root := writeWorkspaceSessionFixture(t, text)
		if code, stdout, stderr := runForTest([]string{"check"}, root); code != ExitSuccess {
			t.Fatalf("prepare locked workspace: %d stdout=%q stderr=%q", code, stdout, stderr)
		}
		return root
	}
	baseline := checked(header + parser)
	args := []string{"experimental", "atlas-diff", "--baseline", baseline,
		"--package-uid", "example.session.app", "--max-declarations", "32", "--max-output-bytes", "100000"}
	guarded := append(slices.Clone(args), "--symbol", "main::decode", "--deny-new-effect", "filesystem")
	for _, tc := range []struct {
		name, text, diagnostic string
	}{
		{"position only", "\n" + header + "\n" + parser, ""},
		{"console added", header + "@effects(console) " + parser, ""},
		{"unrelated function gains filesystem", header + parser + "\n@effects(filesystem) fn i32 other() { return touch(); }", ""},
		{"direct call", header + "@effects(filesystem) fn i32 decode() { return touch(); }", newEffect},
		{"implicit cleanup", header + "@effects(filesystem) fn i32 decode() { let Token token = Token { .value = 7 }; return 7; }", newEffect},
		{"missing current function", header, "in the current workspace"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := checked(tc.text)
			code, stdout, stderr := runForTest(guarded, root)
			if tc.diagnostic != "" {
				if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, tc.diagnostic) {
					t.Fatalf("guard: %d stdout=%q stderr=%q, want %q", code, stdout, stderr, tc.diagnostic)
				}
				if code, stdout, stderr = runForTest(args, root); code != ExitSuccess || !json.Valid([]byte(stdout)) || stderr != "" {
					t.Fatalf("unguarded diff: %d stdout=%q stderr=%q", code, stdout, stderr)
				}
				return
			}
			var report atlasmap.DiffResult
			if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(stdout), &report) != nil ||
				report.Schema != "fairyc.atlas-diff/v1" {
				t.Fatalf("guard: %d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}
	t.Run("missing baseline function", func(t *testing.T) {
		missing := replaceWorkspaceSessionArgument(guarded, "--baseline", checked(header))
		if code, out, diag := runForTest(missing, baseline); code != ExitInvalidInvocation || out != "" || !strings.Contains(diag, "in the baseline") {
			t.Fatalf("missing baseline: %d stdout=%q stderr=%q", code, out, diag)
		}
	})
	t.Run("existing filesystem allowed", func(t *testing.T) {
		old := checked(header + "@effects(filesystem) " + parser)
		current := checked(header + "@effects(filesystem) fn i32 decode() { return touch(); }")
		alreadyDeclared := replaceWorkspaceSessionArgument(guarded, "--baseline", old)
		if code, out, diag := runForTest(alreadyDeclared, current); code != ExitSuccess || !json.Valid([]byte(out)) || diag != "" {
			t.Fatalf("existing filesystem: %d stdout=%q stderr=%q", code, out, diag)
		}
	})
	t.Run("flag validation", func(t *testing.T) {
		workspace := []string{"experimental", "atlas-diff", "--baseline", baseline, "--workspace",
			"--max-packages", "4", "--max-declarations", "32", "--max-output-bytes", "100000"}
		for _, tc := range []struct {
			arguments []string
			message   string
		}{
			{append(slices.Clone(args), "--deny-new-effect", "filesystem"), "must be used together"},
			{append(slices.Clone(args), "--symbol", "main::decode"), "must be used together"},
			{append(slices.Clone(workspace), guarded[len(args):]...), "requires --package-uid"},
			{replaceWorkspaceSessionArgument(guarded, "--symbol", "decode"), "qualified function name"},
			{replaceWorkspaceSessionArgument(guarded, "--deny-new-effect", "console"), "requires filesystem"},
		} {
			if code, out, diag := runForTest(tc.arguments, baseline); code != ExitInvalidInvocation || out != "" || !strings.Contains(diag, tc.message) {
				t.Fatalf("flag validation: %d stdout=%q stderr=%q, want %q", code, out, diag, tc.message)
			}
		}
	})
}
