package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const atlasCopiedClosureEditSource = `module main;
fn i32 choose<T>(T marker){discard marker;let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}
fn i32 leaf(){discard choose<i32>(1);let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}
fn i32 main(){return leaf();}`

const atlasCopiedClosureEditBody = `{discard choose<bool>(true);let i32 offset=20;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(22);}`

func TestAtlasCopiedClosureEditOracle(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasCopiedClosureEditSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("closure edit source oracle: %d %s", code, &diagnostic)
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
		t.Fatal("missing ordinary closure edit subject")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", atlasCopiedClosureEditBody)
	args := []string{"replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(atlasCopiedClosureEditSource))}
	var output bytes.Buffer
	if code := runAtlasEdit(args, root, &output, &diagnostic); code != ExitSuccess {
		t.Fatalf("closure overlay oracle: %d %s", code, &diagnostic)
	}
	var report sourceedit.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || !report.Verified || report.Applied || report.BodyStartByte < 0 || report.BodyEndByte < report.BodyStartByte || report.BodyEndByte > len(atlasCopiedClosureEditSource) {
		t.Fatalf("closure edit report: %v %s", err, &output)
	}
	overlay := atlasCopiedClosureEditSource[:report.BodyStartByte] + atlasCopiedClosureEditBody + atlasCopiedClosureEditSource[report.BodyEndByte:]
	currentRoot := writeWorkspaceSessionFixture(t, overlay)
	diagnostic.Reset()
	current, code := prepareAtlasWorkspaceSchema(currentRoot, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("closure overlay source oracle: %d %s", code, &diagnostic)
	}
	for i, workspace := range []preparedAtlasWorkspace{prepared, current} {
		implementations, helpers := 0, 0
		for _, partition := range workspace.partitions {
			for _, declaration := range partition.Declarations() {
				if strings.HasPrefix(declaration.Key.Name, "_fy_closure_") {
					helpers++
					if declaration.Public {
						t.Fatal("closure helper became public")
					}
				}
			}
			for _, specialization := range partition.Specializations() {
				implementations++
				want := "i32"
				if i == 1 {
					want = "bool"
				}
				if specialization.Declaration.Name != "choose" || len(specialization.TypeArguments) != 1 || specialization.TypeArguments[0].Name != want {
					t.Fatal("closure overlay kept an old concrete owner", specialization)
				}
			}
		}
		if implementations != 1 || helpers != 2 {
			t.Fatalf("closure overlay inventory: overlay=%d owners=%d helpers=%d", i, implementations, helpers)
		}
	}
}

func TestCandidateEditCopiedClosureCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditCopiedClosureCommand)
}

func verifyCandidateEditCopiedClosureCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:    atlasCopiedClosureEditSource,
		body:      atlasCopiedClosureEditBody,
		invalid:   "{let add=fn [] i32(){return true;};return add();}",
		module:    "answer",
		name:      "value",
		deniedUID: "example.session.answer",
	})
}
