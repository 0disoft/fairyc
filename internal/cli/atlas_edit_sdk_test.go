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

func TestAtlasEditSDKVectorOverlay(t *testing.T) {
	const source = `module main;import std::vec as v;fn i32 leaf(){return 0;}fn i32 main(){return leaf();}`
	const body = `{var values=v::empty<bool>();assert(v::push<bool>(ref mut values,true).is_ok);return if(*v::at<bool>(ref values,0)){43}else{1};}`
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("SDK edit selector preparation: %d %s", code, &diagnostic)
	}
	var symbol string
	for _, partition := range prepared.partitions {
		if partition.Header().PackageUID == "example.session.app" {
			for _, candidate := range partition.Symbols() {
				if candidate.Declaration.Name == "leaf" {
					symbol = candidate.Visible
				}
			}
		}
	}
	if symbol == "" {
		t.Fatal("missing SDK edit root selector")
	}
	args := []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(source))}
	for _, invalid := range []string{
		`{var values=v::empty<bool>();discard v::push<bool>(ref mut values,43);return 43;}`,
		`{let usize size=unsafe{storage_size<i32>()};return 43;}`,
	} {
		writeWorkspaceFile(t, root, "edits/body.txt", invalid)
		before := remoteTree(t, root)
		code, out, diagnostic := runForTest(append(append([]string{}, args...), "--apply"), root)
		if code != 2 || out != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("invalid or unauthorized SDK overlay: %d %q %q", code, out, diagnostic)
		}
	}
	writeWorkspaceFile(t, root, "edits/body.txt", body)
	before := remoteTree(t, root)
	code, out, stderr := runForTest(args, root)
	var preview sourceedit.Report
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &preview) != nil || !preview.Verified || preview.Applied || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("SDK overlay preview: %d %s %s", code, out, stderr)
	}
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, stderr = runForTest(append(append([]string{}, args...), "--apply"), root)
	var applied sourceedit.Report
	preview.Applied = true
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &applied) != nil || applied != preview {
		t.Fatalf("SDK overlay apply: %d %s %s", code, out, stderr)
	}
	current, err := os.ReadFile(filepath.Join(root, "src/main.fy"))
	if err != nil || string(current) != strings.Replace(source, "{return 0;}", body, 1) {
		t.Fatalf("SDK overlay changed unexpected bytes: %v %s", err, current)
	}
	currentLock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, currentLock) {
		t.Fatal("SDK overlay changed graph lock", err)
	}
	before["src/main.fy"] = sourceedit.Digest(current)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("SDK overlay changed unrelated files or leaked staging/lock")
	}
}
