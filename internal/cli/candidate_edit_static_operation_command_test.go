package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const atlasStaticOperationEditSource = `module main;import answer as a;
fn i32 leaf(){discard a::algorithm<i64>(14);return a::algorithm<i32>(21);}
fn i32 main(){return leaf();}`

const atlasStaticOperationEditBody = `{assert(a::algorithm<bool>(true));return a::algorithm<i32>(22);}`

func TestAtlasStaticOperationEditOracle(t *testing.T) {
	root := atlasStaticOperationWorkspace(t, atlasStaticOperationEditSource, atlasStaticOperationProvider)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("operation edit source oracle: %d %s", code, &diagnostic)
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
		t.Fatal("missing ordinary edit subject")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", atlasStaticOperationEditBody)
	args := []string{"replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(atlasStaticOperationEditSource))}
	var output bytes.Buffer
	if code := runAtlasEdit(args, root, &output, &diagnostic); code != ExitSuccess {
		t.Fatalf("operation overlay oracle: %d %s", code, &diagnostic)
	}
	var report sourceedit.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || !report.Verified || report.Applied || report.BodyStartByte < 0 || report.BodyEndByte < report.BodyStartByte || report.BodyEndByte > len(atlasStaticOperationEditSource) {
		t.Fatalf("operation edit report: %v %s", err, &output)
	}
	overlay := atlasStaticOperationEditSource[:report.BodyStartByte] + atlasStaticOperationEditBody + atlasStaticOperationEditSource[report.BodyEndByte:]
	currentRoot := atlasStaticOperationWorkspace(t, overlay, atlasStaticOperationProvider)
	diagnostic.Reset()
	current, code := prepareAtlasWorkspaceSchema(currentRoot, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("operation overlay source oracle: %d %s", code, &diagnostic)
	}
	for i, workspace := range []preparedAtlasWorkspace{prepared, current} {
		counts := map[string]int{}
		for _, partition := range workspace.partitions {
			for _, specialization := range partition.Specializations() {
				if len(specialization.TypeArguments) != 1 || specialization.Declaration.Module != "answer" {
					t.Fatal("unexpected overlay implementation", specialization)
				}
				counts[specialization.TypeArguments[0].Name]++
			}
		}
		if counts["i32"] != 2 || i == 0 && (counts["i64"] != 2 || counts["bool"] != 0) || i == 1 && (counts["i64"] != 0 || counts["bool"] != 2) || len(counts) != 2 {
			t.Fatalf("fresh bool operation implementations: overlay=%v counts=%v", i == 1, counts)
		}
	}
}

func TestCandidateEditStaticOperationCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditStaticOperationCommand)
}

func verifyCandidateEditStaticOperationCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:    atlasStaticOperationEditSource,
		body:      atlasStaticOperationEditBody,
		invalid:   "{return a::algorithm<bool>(true);}",
		provider:  atlasStaticOperationProvider,
		module:    "answer",
		name:      "transform",
		deniedUID: "example.session.answer",
	})
}
