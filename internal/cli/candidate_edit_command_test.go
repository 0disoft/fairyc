package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/sourceedit"
)

func TestCandidateEditCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console) fn i32 main(){return edit::arguments();}`
	candidateEditCommandProbe(t, main, func(t *testing.T, image string) {
		t.Run("preview", func(t *testing.T) { verifyCandidateEditCommand(t, image) })
		verifyCandidateEditApply(t, image)
	})
}

func verifyCandidateEditCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditCommandWithHelp(t, image, usage)
}

func verifyCandidateEditCommandWithHelp(t *testing.T, image, help string) {
	t.Helper()
	root, args := editFixture(t)
	var reference, diagnostic bytes.Buffer
	if code := runAtlasEdit(args[2:], root, &reference, &diagnostic); code != 0 {
		t.Fatal(code, &diagnostic)
	}
	var want sourceedit.Report
	if json.Unmarshal(reference.Bytes(), &want) != nil {
		t.Fatal(&reference)
	}
	t.Setenv("PATH", "")
	before := remoteTree(t, root)
	code, out, stderr := fetchCLIProcess(t, image, root, args, "")
	var got sourceedit.Report
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want || len(out) > 8192 {
		t.Fatalf("actual preview: %d %s %s; want %+v", code, out, stderr, want)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("preview changed the workspace")
	}
	for _, body := range []string{"{return false;}", "{return 1;", "{return 1;}fn void injected(){}"} {
		writeWorkspaceFile(t, root, "edits/body.txt", body)
		snapshot := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(snapshot, remoteTree(t, root)) {
			t.Fatalf("invalid body %q: %d %s %s", body, code, out, stderr)
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", "{return 1;}")
	for _, path := range []string{"../outside", "/absolute", "edits\\body.txt", "edits/../body.txt", "edits/Body.txt", "src/body.txt", "libs/answer/src/body.txt", "absent/body.txt"} {
		bad := append([]string{}, args...)
		bad[8] = path
		snapshot := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, bad, "")
		if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(snapshot, remoteTree(t, root)) {
			t.Fatalf("body path %q: %d %s %s", path, code, out, stderr)
		}
	}
	// Lexical prefix alone must not reject a sibling of a production root.
	writeWorkspaceFile(t, root, "libs/answer/src2/body.txt", "{return 1;}")
	sibling := append([]string{}, args...)
	sibling[8] = "libs/answer/src2/body.txt"
	code, out, stderr = fetchCLIProcess(t, image, root, sibling, "")
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want {
		t.Fatalf("source-root sibling: %d %s %s", code, out, stderr)
	}
	if err := os.RemoveAll(filepath.Join(root, "libs", "answer", "src2")); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{sourceedit.MaxBody - 1, sourceedit.MaxBody, sourceedit.MaxBody + 1} {
		writeWorkspaceFile(t, root, "edits/body.txt", "{return 1;}"+strings.Repeat(" ", n-len("{return 1;}")))
		snapshot := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if n <= sourceedit.MaxBody {
			if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want {
				t.Fatalf("body byte bound %d: %d %s %s", n, code, out, stderr)
			}
		} else if code != 7 || out != "" || stderr == "" {
			t.Fatalf("oversize body: %d %s %s", code, out, stderr)
		}
		if !reflect.DeepEqual(snapshot, remoteTree(t, root)) {
			t.Fatal("bounded body read changed files")
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", "{return 1;}")
	if err := os.Symlink(filepath.Join(root, "edits"), filepath.Join(root, "linked")); err != nil {
		t.Fatal("body-path symlink control requires Windows developer mode:", err)
	}
	linked := append([]string{}, args...)
	linked[8] = "linked/body.txt"
	code, out, stderr = fetchCLIProcess(t, image, root, linked, "")
	if code != 7 || out != "" || stderr == "" {
		t.Fatalf("linked body directory: %d %s %s", code, out, stderr)
	}
	if err := os.Remove(filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	code, out, stderr = fetchCLIProcess(t, image, root, args, "stdout")
	if code != 4 || out != "" || !strings.Contains(stderr, "write output failed") {
		t.Fatalf("broken preview output: %d %s %s", code, out, stderr)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("preview/error path left changes")
	}
	code, out, stderr = fetchCLIProcess(t, image, t.TempDir(), []string{"experimental", "atlas-edit", "--help"}, "")
	if code != 0 || out != help || stderr != "" {
		t.Fatalf("edit help without workspace: %d %q %q", code, out, stderr)
	}
}
