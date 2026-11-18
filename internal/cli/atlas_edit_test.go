package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sourceedit"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const editSource = "module main;import answer as a;\n// keep this comment\nfn i32 helper(){return 0;}\nfn i32 main(){return a::value()+helper();}\n"

func editFixture(t *testing.T) (string, []string) {
	root := writeWorkspaceSessionFixture(t, editSource)
	s := atlasFunctionSymbolForPackage(t, root, "example.session.app", "helper")
	writeWorkspaceFile(t, root, "edits/body.txt", "{return 1;}")
	return root, []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.session.app", "--symbol", s.Visible, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(editSource))}
}

func TestAtlasEditPreviewApplyAndRejectEscapes(t *testing.T) {
	root, args := editFixture(t)
	lockBefore, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, stderr := runForTest(args, root)
	var report sourceedit.Report
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &report) != nil || !report.Verified || report.Applied || report.BodyStartByte <= 0 || report.BodyEndByte <= report.BodyStartByte || strings.Contains(out, root) {
		t.Fatalf("preview %d %s %s", code, out, stderr)
	}
	before, _ := os.ReadFile(filepath.Join(root, "src/main.fy"))
	if string(before) != editSource {
		t.Fatal("preview wrote source")
	}
	for _, body := range []string{"{return true;}", "{return 1;", "{return 1;} fn i32 injected(){return 9;} fn i32 carrier(){return 0;}"} {
		writeWorkspaceFile(t, root, "edits/body.txt", body)
		if code, out, _ := runForTest(append(append([]string{}, args...), "--apply"), root); code == 0 || out != "" {
			t.Fatalf("invalid edit accepted: %s", body)
		}
		actual, _ := os.ReadFile(filepath.Join(root, "src/main.fy"))
		if !bytes.Equal(actual, before) {
			t.Fatal("rejected edit changed source")
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", "{return 1;}")
	code, out, stderr = runForTest(append(append([]string{}, args...), "--apply"), root)
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &report) != nil || !report.Applied {
		t.Fatalf("apply %d %s %s", code, out, stderr)
	}
	after, _ := os.ReadFile(filepath.Join(root, "src/main.fy"))
	expected := strings.Replace(editSource, "{return 0;}", "{return 1;}", 1)
	if string(after) != expected {
		t.Fatalf("changed bytes outside body: %s", after)
	}
	lockAfter, _ := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("edit refreshed graph lock")
	}
	if code, _, _ := runForTest([]string{"check", "--locked"}, root); code != ExitPackageFailure {
		t.Fatal("edit did not invalidate old lock")
	}
	if code, out, _ := runForTest(args, root); code == 0 || out != "" {
		t.Fatal("stale selector/source accepted")
	}
}

func TestAtlasEditPlanRejectsDependencyDriftAndLinks(t *testing.T) {
	root, _ := editFixture(t)
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := sourceedit.PlanBody(w, "main", "helper", sourceedit.Digest([]byte(editSource)), []byte("{return 1;}"))
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", strings.Replace(workspaceSessionMemberSource, "42", "43", 1))
	if err := p.Apply(); err == nil {
		t.Fatal("changed dependency admitted")
	}
	actual, _ := os.ReadFile(filepath.Join(root, "src/main.fy"))
	if string(actual) != editSource {
		t.Fatal("drift rejection changed source")
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	for _, path := range []string{"../outside", "/absolute", "src/../main.fy", "src\\main.fy"} {
		if _, err := sourceedit.ReadFile(opened, path, 100); err == nil {
			t.Fatal("nonportable edit path accepted", path)
		}
	}
	if err := os.Symlink(filepath.Join(root, "edits/body.txt"), filepath.Join(root, "edits/link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := sourceedit.ReadFile(opened, "edits/link.txt", 100); err == nil {
		t.Fatal("body symlink admitted")
	}
}

func TestAtlasEditNative(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("existing native hosts only")
	}
	if runtime.GOOS == "windows" {
		if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
			t.Skipf("MSVC unavailable: %v", err)
		}
	}
	root, args := editFixture(t)
	if code, out, err := runForTest(append(args, "--apply"), root); code != 0 || err != "" {
		t.Fatalf("apply %d %s %s", code, out, err)
	}
	if code, out, err := runForTest([]string{"check"}, root); code != 0 || err != "" {
		t.Fatalf("recheck %d %s %s", code, out, err)
	}
	checkLockedNativeWorkspaceStatus(t, root, 43)
}
