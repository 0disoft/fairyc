package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/sourceedit"
)

func TestAtlasEditBatchChecksAllBodiesWithoutWrites(t *testing.T) {
	const uid = "example.session.app"
	const first = "module main; fn i32 left(){return 10;} fn i32 right(){return 20;} fn i32 main(){return left()+right();}\n"
	const second = "module other; fn i32 third(){return 30;}\n"
	root := writeWorkspaceSessionFixture(t, first)
	writeWorkspaceFile(t, root, "src/other.fy", second)
	left := atlasFunctionSymbolForPackage(t, root, uid, "left")
	right := atlasFunctionSymbolForPackage(t, root, uid, "right")
	third := atlasFunctionSymbolForPackage(t, root, uid, "third")
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{
		"edits/left.txt": "{return 100;}", "edits/right.txt": "{return 2;}", "edits/third.txt": "{return 7;}",
	} {
		writeWorkspaceFile(t, root, path, text)
	}
	type edit struct {
		Symbol string `json:"symbol"`
		Body   string `json:"bodyFile"`
		Hash   string `json:"expectedSourceSha256"`
	}
	edits := []edit{
		{right.Full, "edits/right.txt", sourceedit.Digest([]byte(first))},
		{third.Full, "edits/third.txt", sourceedit.Digest([]byte(second))},
		{left.Visible, "edits/left.txt", sourceedit.Digest([]byte(first))},
	}
	writeRequest := func(edits []edit) {
		t.Helper()
		data, err := json.Marshal(map[string][]edit{"edits": edits})
		if err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, "edits/request.json", string(data))
	}
	args := []string{"experimental", "atlas-edit", "preview-batch", "--package-uid", uid, "--edit-file", "edits/request.json"}
	writeRequest(edits)
	code, stdout, diagnostics := runForTest(args, root)
	var result sourceedit.BatchReport
	if code != ExitSuccess || diagnostics != "" || json.Unmarshal([]byte(stdout), &result) != nil ||
		result.Schema != "fairyc.structural-body-edit-batch/v1" || !result.Verified || len(result.Edits) != 3 {
		t.Fatalf("preview = %d %s %s", code, stdout, diagnostics)
	}
	before := []string{first, second, first}
	afterMain := strings.NewReplacer("return 10;", "return 100;", "return 20;", "return 2;").Replace(first)
	afterOther := strings.Replace(second, "return 30;", "return 7;", 1)
	after := []string{afterMain, afterOther, afterMain}
	for i, report := range result.Edits {
		wantName := []string{"right", "third", "left"}[i]
		if report.Applied || !report.Verified || report.Function != wantName || report.BeforeSHA256 != sourceedit.Digest([]byte(before[i])) ||
			report.AfterSHA256 != sourceedit.Digest([]byte(after[i])) {
			t.Fatalf("preview edit %d = %+v", i, report)
		}
	}
	if code, output, _ := runForTest(append(append([]string(nil), args...), "--apply"), root); code != ExitInvalidInvocation || output != "" {
		t.Fatalf("batch --apply = %d %s", code, output)
	}
	for _, body := range []string{"{return true;}", "{return 1;} fn i32 injected(){return 9;} fn i32 carrier(){return 0;}"} {
		writeWorkspaceFile(t, root, "edits/third.txt", body)
		if code, output, _ := runForTest(args, root); code != ExitInvalidInvocation || output != "" {
			t.Fatalf("invalid batch = %d %s", code, output)
		}
	}
	writeWorkspaceFile(t, root, "edits/third.txt", "{return 7;}")
	writeRequest(append(edits, edits[0]))
	if code, output, _ := runForTest(args, root); code != ExitInvalidInvocation || output != "" {
		t.Fatalf("duplicate batch = %d %s", code, output)
	}
	for path, want := range map[string]string{"src/main.fy": first, "src/other.fy": second, "fairy.lock": string(lock)} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || string(data) != want {
			t.Fatalf("batch changed %s: %v", path, err)
		}
	}
}
