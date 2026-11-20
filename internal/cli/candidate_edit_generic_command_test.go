package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

func TestCandidateEditGenericCommand(t *testing.T) {
	const main = `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditGenericCommand)
}

func verifyCandidateEditGenericCommand(t *testing.T, image string) {
	const source = `module main;
fn T transform<T>(T value){return value;}
fn T unused<T>(T value){return value;}
fn i32 main(){if(transform<bool>(true)){return transform<i32>(42);}return 0;}
`
	const replacement = "{let T copy=value;return copy;}"
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("prepare edit oracle: %d %s", code, &diagnostic)
	}
	symbols := map[string]string{}
	for _, partition := range prepared.partitions {
		if partition.Header().PackageUID == "example.session.app" {
			for _, s := range partition.Symbols() {
				symbols[s.Declaration.Name] = s.Visible
			}
		}
	}
	args := func(subject string) []string {
		t.Helper()
		if symbols[subject] == "" {
			t.Fatalf("missing edit selector for %s", subject)
		}
		return []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.session.app",
			"--symbol", symbols[subject], "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(source))}
	}
	assertRejected := func(subject, body, reason string) {
		t.Helper()
		writeWorkspaceFile(t, root, "edits/body.txt", body)
		before := remoteTree(t, root)
		code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, append(args(subject), "--apply"))
		if code != ExitInvalidInvocation || out != "" || !strings.Contains(stderr, reason) || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("reject %s: %d %q %q", subject, code, out, stderr)
		}
	}

	assertRejected("unused", replacement, "requires a current concrete specialization")
	assertRejected("transform", "{return 42;}", "failed semantic/MIR validation")
	writeWorkspaceFile(t, root, "edits/body.txt", replacement)
	before := remoteTree(t, root)
	for _, apply := range []bool{false, true} {
		arguments := args("transform")
		if apply {
			arguments = append(arguments, "--apply")
		}
		var oracle bytes.Buffer
		diagnostic.Reset()
		if code := runAtlasEdit(args("transform")[2:], root, &oracle, &diagnostic); code != ExitSuccess {
			t.Fatalf("Go preview: %d %s", code, &diagnostic)
		}
		var want, got sourceedit.Report
		if err := json.Unmarshal(oracle.Bytes(), &want); err != nil || !want.Verified {
			t.Fatalf("Go preview report: %v %s", err, &oracle)
		}
		want.Applied = apply
		code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, arguments)
		if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want {
			t.Fatalf("edit apply=%v: %d %q %q, want %+v", apply, code, out, stderr, want)
		}
		if !apply && !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("preview changed workspace files")
		}
	}
	current, err := os.ReadFile(filepath.Join(root, "src", "main.fy"))
	if err != nil || string(current) != strings.Replace(source, "{return value;}", replacement, 1) {
		t.Fatalf("applied source: %v %s", err, current)
	}
	before["src/main.fy"] = sourceedit.Digest(current)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("apply changed another file or left staging files")
	}
}
