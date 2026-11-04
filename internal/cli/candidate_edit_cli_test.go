package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/sourceedit"
)

// This runs in the complete unmodified candidate CLI, without fault wrappers
// or installed companions. Go independently supplies the applied report, exact
// source bytes and graph lock after a subsequent ordinary check.
func verifyCandidateEditCLIApply(t *testing.T, image string) {
	t.Helper()
	reference, referenceArgs := editFixture(t)
	code, output, diagnostic := runForTest(append(referenceArgs, "--apply"), reference)
	var want sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &want) != nil || !want.Applied {
		t.Fatal(code, output, diagnostic)
	}
	updated, err := os.ReadFile(filepath.Join(reference, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	if code, _, diagnostic := runForTest([]string{"check"}, reference); code != 0 || diagnostic != "" {
		t.Fatal(code, diagnostic)
	}
	lock, err := os.ReadFile(filepath.Join(reference, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}

	root, args := editFixture(t)
	before := remoteTree(t, root)
	code, output, diagnostic = fetchCLIProcess(t, image, root, append(args, "--apply"), "")
	var got sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &got) != nil || got != want {
		t.Fatalf("full CLI application: %d %s %s; want %+v", code, output, diagnostic, want)
	}
	before["src/main.fy"] = sourceedit.Digest(updated)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("full CLI application changed other inputs or leaked staging/lock")
	}
	for _, command := range [][]string{append(args, "--apply"), {"check", "--locked"}} {
		code, output, diagnostic = fetchCLIProcess(t, image, root, command, "")
		if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("old-lock refusal %v: %d %s %s", command, code, output, diagnostic)
		}
	}
	for _, command := range [][]string{{"check"}, {"check", "--locked"}} {
		code, output, diagnostic = fetchCLIProcess(t, image, root, command, "")
		if code != 0 || diagnostic != "" {
			t.Fatalf("post-edit %v: %d %s %s", command, code, output, diagnostic)
		}
	}
	before["fairy.lock"] = sourceedit.Digest(lock)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("post-edit graph lock differs from the Go reference or another input changed")
	}
}
