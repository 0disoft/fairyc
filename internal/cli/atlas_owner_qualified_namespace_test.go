package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

func atlasOwnerQualifiedNamespaceSource(ordinary bool) string {
	source := `module main;struct Cell{i32 value;}struct Other{i32 value;}
fn i32 Cell::get(ref Cell this){return this.value;}
fn i32 Other::get(ref Other this){return this.value;}`
	call := ""
	if ordinary {
		source += "fn i32 get(){return 1;}"
		call = "+get()"
	}
	return source + `fn i32 main(){let a=Cell{.value=20};let b=Other{.value=21};let left=ref a;let right=ref b;return left.get()+right.get()` + call + ";}"
}

func TestAtlasOwnerQualifiedDistinctNamespaces(t *testing.T) {
	for _, ordinary := range []bool{false, true} {
		t.Run(fmt.Sprint("ordinary=", ordinary), func(t *testing.T) {
			source := atlasOwnerQualifiedNamespaceSource(ordinary)
			root := writeWorkspaceSessionFixture(t, source)
			if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess {
				t.Fatalf("ordinary owner namespaces must be valid: %d %s", code, diagnostic)
			}
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("owner namespaces lost during v3 preparation: %d %s", code, &diagnostic)
			}
			want := map[string]bool{"Cell::get": false, "Other::get": false}
			if ordinary {
				want["get"] = false
			}
			for _, partition := range prepared.partitions {
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				for _, symbol := range partition.Symbols() {
					if _, ok := want[symbol.Declaration.Name]; ok {
						if want[symbol.Declaration.Name] {
							t.Fatal("duplicate qualified function identity")
						}
						want[symbol.Declaration.Name] = true
					}
				}
			}
			for name, seen := range want {
				if !seen {
					t.Fatal("missing distinct function identity", name)
				}
			}
			index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
			if err != nil || len(index.Calls()) != len(want) {
				t.Fatalf("distinct same-leaf calls: index=%v err=%v", index, err)
			}
		})
	}
}

func TestAtlasEditOrdinaryFunctionBesideSameLeafMethods(t *testing.T) {
	source := atlasOwnerQualifiedNamespaceSource(true)
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("same-leaf edit oracle: %d %s", code, &diagnostic)
	}
	symbols := map[string]string{}
	for _, partition := range prepared.partitions {
		if partition.Header().PackageUID == "example.session.app" {
			for _, symbol := range partition.Symbols() {
				symbols[symbol.Declaration.Name] = symbol.Visible
			}
		}
	}
	if symbols["get"] == "" || symbols["Cell::get"] == "" || symbols["Other::get"] == "" {
		t.Fatal("missing same-leaf edit subjects")
	}
	const body = "{return 2;}"
	writeWorkspaceFile(t, root, "edits/body.txt", body)
	args := []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.session.app", "--symbol", symbols["get"], "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(source))}
	before := remoteTree(t, root)
	code, out, stderr := runForTest(args, root)
	var preview sourceedit.Report
	if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &preview) != nil || !preview.Verified || preview.Applied || preview.Function != "get" || preview.BodyStartByte < 0 || preview.BodyEndByte < preview.BodyStartByte || preview.BodyEndByte > len(source) || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("ordinary same-leaf preview: %d %s %s", code, out, stderr)
	}
	for _, method := range []string{"Cell::get", "Other::get"} {
		denied := replaceWorkspaceSessionArgument(args, "--symbol", symbols[method])
		code, out, stderr := runForTest(append(denied, "--apply"), root)
		if code != ExitInvalidInvocation || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("method must stay noneditable: %s %d %s %s", method, code, out, stderr)
		}
	}
	code, out, stderr = runForTest(append(args, "--apply"), root)
	var applied sourceedit.Report
	want := preview
	want.Applied = true
	if code != ExitSuccess || stderr != "" || json.Unmarshal([]byte(out), &applied) != nil || applied != want {
		t.Fatalf("ordinary same-leaf apply: %d %s %s", code, out, stderr)
	}
	current, err := os.ReadFile(filepath.Join(root, "src/main.fy"))
	expected := source[:preview.BodyStartByte] + body + source[preview.BodyEndByte:]
	if err != nil || string(current) != expected {
		t.Fatalf("same-leaf edit changed method or unrelated bytes: %v %s", err, current)
	}
	before["src/main.fy"] = sourceedit.Digest(current)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("same-leaf edit changed lock, method or staging files")
	}
}
