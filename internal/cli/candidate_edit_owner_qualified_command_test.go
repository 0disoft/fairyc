package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const atlasOwnerQualifiedEditSource = `module main;import answer as a;
struct Cell{i32 value;}resource Token{i32 value;cleanup{}}
fn i32 Cell::answer(){return a::value();}
fn i32 Cell::get(ref Cell this){return this.value;}
fn void Cell::set(ref mut Cell this,i32 value){this.value=value;}
fn i32 Token::take(Token this){return this.value;}
fn i32 leaf(){let cell=Cell{.value=40};let view=ref cell;return view.get()+2;}
fn i32 main(){return leaf();}`

const atlasOwnerQualifiedEditBody = `{var cell=Cell{.value=0};{let view=ref mut cell;view.set(Cell::answer());}let token=Token{.value=cell.value};return token.take();}`

func TestAtlasOwnerQualifiedEditOracle(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasOwnerQualifiedEditSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("owner edit original oracle: %d %s", code, &diagnostic)
	}
	var symbol string
	for _, partition := range prepared.partitions {
		for _, s := range partition.Symbols() {
			if partition.Header().PackageUID == "example.session.app" && s.Declaration.Name == "leaf" {
				symbol = s.Visible
			}
		}
	}
	if symbol == "" {
		t.Fatal("missing ordinary owner-call edit subject")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", atlasOwnerQualifiedEditBody)
	args := []string{"replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(atlasOwnerQualifiedEditSource))}
	var output bytes.Buffer
	if code := runAtlasEdit(args, root, &output, &diagnostic); code != ExitSuccess {
		t.Fatalf("owner edit overlay oracle: %d %s", code, &diagnostic)
	}
	var report sourceedit.Report
	if json.Unmarshal(output.Bytes(), &report) != nil || !report.Verified || report.Applied || report.Function != "leaf" || report.BodyStartByte < 0 || report.BodyEndByte < report.BodyStartByte || report.BodyEndByte > len(atlasOwnerQualifiedEditSource) {
		t.Fatal("invalid ordinary owner-call edit report", &output)
	}
	overlay := atlasOwnerQualifiedEditSource[:report.BodyStartByte] + atlasOwnerQualifiedEditBody + atlasOwnerQualifiedEditSource[report.BodyEndByte:]
	currentRoot := writeWorkspaceSessionFixture(t, overlay)
	diagnostic.Reset()
	current, code := prepareAtlasWorkspaceSchema(currentRoot, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("owner edit current oracle: %d %s", code, &diagnostic)
	}
	for i, workspace := range []preparedAtlasWorkspace{prepared, current} {
		index, err := atlasmap.NewWorkspaceCallIndexV3(workspace.partitions, workspace.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
		if err != nil || len(index.Calls()) != []int{3, 5}[i] {
			t.Fatalf("owner edit call inventory: overlay=%d index=%v err=%v", i, index, err)
		}
	}
}

func TestCandidateEditOwnerQualifiedCommand(t *testing.T) {
	const main = `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditOwnerQualifiedCommand)
}

func verifyCandidateEditOwnerQualifiedCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:    atlasOwnerQualifiedEditSource,
		body:      atlasOwnerQualifiedEditBody,
		invalid:   "{let token=Token{.value=42};discard token.take();return token.take();}",
		module:    "main",
		name:      "Cell::get",
		deniedUID: "example.session.app",
	})
}
