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

const editAggregateSource = `module root;import maker as m;
// Preserve declarations outside the selected body.
struct Local<T>{T value;}
fn T identity<T>(T value){return value;}
pub fn i32 public_leaf(){return 42;}
fn i32 leaf(m::Cell<i32> value){return value.value;}
@effects(console)fn i32 main(){let c=m::Cell<i32>{.value=42};return leaf(c);}
`

const editAggregateBody = "{let copy=value;let flag=m::Cell<bool>{.value=true};assert(flag.value);return copy.value+1;}"

func TestCandidateEditAggregateCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditAggregateCommand)
}

func verifyCandidateEditAggregateCommand(t *testing.T, image string) {
	t.Helper()
	root := candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{root: editAggregateSource})
	prepared, _ := candidateAtlasAggregateCommandOracle(t, root)
	symbols := map[string]string{}
	for _, p := range prepared.partitions {
		if p.Header().PackageUID == "example.aggregate.root" {
			for _, s := range p.Symbols() {
				symbols[s.Declaration.Name] = s.Visible
			}
		}
	}
	if len(symbols) != 5 || symbols["leaf"] == "" {
		t.Fatal("missing aggregate edit source declarations", symbols)
	}
	writeWorkspaceFile(t, root, "edits/body.txt", editAggregateBody)
	args := []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.aggregate.root", "--symbol", symbols["leaf"], "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(editAggregateSource))}
	var reference, diagnostic bytes.Buffer
	if code := runAtlasEdit(args[2:], root, &reference, &diagnostic); code != 0 {
		t.Fatalf("aggregate edit Go oracle: %d %s", code, &diagnostic)
	}
	var want sourceedit.Report
	if json.Unmarshal(reference.Bytes(), &want) != nil || !want.Verified || want.Applied {
		t.Fatal("invalid aggregate edit Go report", &reference)
	}
	t.Setenv("PATH", "")
	before := remoteTree(t, root)
	code, out, stderr := fetchCLIProcess(t, image, root, args, "")
	var got sourceedit.Report
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("aggregate preview: %d %s %s, want %+v", code, out, stderr, want)
	}
	for _, body := range []string{"{return false;}", "{return value.missing;}", "{return 1;}fn i32 injected(){return 1;}"} {
		writeWorkspaceFile(t, root, "edits/body.txt", body)
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, append(append([]string{}, args...), "--apply"), "")
		if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("invalid aggregate overlay %s: %d %s %s", body, code, out, stderr)
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", editAggregateBody)
	for _, name := range []string{"Local", "identity", "public_leaf", "main"} {
		invalid := replaceWorkspaceSessionArgument(args, "--symbol", symbols[name])
		before := remoteTree(t, root)
		code, out, stderr := fetchCLIProcess(t, image, root, invalid, "")
		if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("noneditable aggregate subject %s: %d %s %s", name, code, out, stderr)
		}
	}
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	beforeApply := remoteTree(t, root)
	code, out, stderr = fetchCLIProcess(t, image, root, append(append([]string{}, args...), "--apply"), "")
	want.Applied = true
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want {
		t.Fatalf("aggregate apply: %d %s %s, want %+v", code, out, stderr, want)
	}
	after, err := os.ReadFile(filepath.Join(root, "src", "root.fy"))
	if err != nil || string(after) != strings.Replace(editAggregateSource, "{return value.value;}", editAggregateBody, 1) {
		t.Fatalf("aggregate edit changed bytes outside target: %v %s", err, after)
	}
	currentLock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, currentLock) {
		t.Fatal("aggregate edit refreshed lock", err)
	}
	beforeApply["src/root.fy"] = sourceedit.Digest(after)
	if !reflect.DeepEqual(beforeApply, remoteTree(t, root)) {
		t.Fatal("aggregate edit left staging/lock files or changed unrelated bytes")
	}
	snapshot := remoteTree(t, root)
	code, out, stderr = fetchCLIProcess(t, image, root, args, "")
	if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(snapshot, remoteTree(t, root)) {
		t.Fatalf("aggregate edit stale lock: %d %s %s", code, out, stderr)
	}
	// Relock with independent Go only after the native publisher has completed.
	// The candidate must then reject the old source hash without changing bytes.
	candidateAtlasAggregateCommandOracle(t, root)
	snapshot = remoteTree(t, root)
	code, out, stderr = fetchCLIProcess(t, image, root, args, "")
	if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(snapshot, remoteTree(t, root)) {
		t.Fatalf("aggregate edit stale source hash: %d %s %s", code, out, stderr)
	}
}
