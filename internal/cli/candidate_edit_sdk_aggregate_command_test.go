package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

// Neither the SDK Vec<bool> nor the client Vec<bool> instance exists before the
// edit. Flag's cleanup continues to use the original authenticated declaration.
const editSDKUserAggregateBody = `{let flag=Flag<bool>{.value=true};let cell=Vec<bool>{.value=true};var values=v::empty<bool>();assert(v::push<bool>(ref mut values,cell.value).is_ok);return if(*v::at<bool>(ref values,0)){43}else{1};}`

func TestAtlasSDKUserAggregateEditOracle(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasSDKUserAggregateCommandSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("mixed edit source oracle: %d %s", code, &diagnostic)
	}
	var symbol string
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if p.Header().PackageUID == "example.session.app" && s.Declaration.Name == "leaf" {
				symbol = s.Visible
			}
		}
	}
	if symbol == "" {
		t.Fatal("missing mixed edit leaf")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", editSDKUserAggregateBody)
	var report bytes.Buffer
	args := []string{"replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(atlasSDKUserAggregateCommandSource))}
	if code := runAtlasEdit(args, root, &report, &diagnostic); code != 0 {
		t.Fatalf("mixed edit preview oracle: %d %s", code, &diagnostic)
	}
	var result sourceedit.Report
	if err := json.Unmarshal(report.Bytes(), &result); err != nil || !result.Verified || result.Applied || result.BodyStartByte < 0 || result.BodyEndByte < result.BodyStartByte || result.BodyEndByte > len(atlasSDKUserAggregateCommandSource) {
		t.Fatalf("mixed edit oracle report: %v %s", err, &report)
	}
	overlay := atlasSDKUserAggregateCommandSource[:result.BodyStartByte] + editSDKUserAggregateBody + atlasSDKUserAggregateCommandSource[result.BodyEndByte:]
	newRoot := writeWorkspaceSessionFixture(t, overlay)
	diagnostic.Reset()
	current, code := prepareAtlasWorkspaceSchema(newRoot, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("mixed edit overlay oracle: %d %s", code, &diagnostic)
	}
	for _, p := range []struct {
		prepared preparedAtlasWorkspace
		fresh    bool
	}{{prepared, false}, {current, true}} {
		sdk := map[string]bool{}
		client := false
		for _, partition := range p.prepared.partitions {
			for _, instance := range partition.Specializations() {
				if len(instance.TypeArguments) != 1 || instance.TypeArguments[0].Name != "bool" {
					continue
				}
				if instance.Declaration.Module == "std::vec" {
					sdk[instance.Declaration.Name] = true
				} else if instance.Declaration.Module == "main" && instance.Declaration.Name == "Vec" && instance.Aggregate != nil {
					client = true
				}
			}
		}
		want := map[string]bool{}
		if p.fresh {
			want = map[string]bool{"Vec": true, "take_slot": true, "empty": true, "push": true, "at": true}
		}
		if !reflect.DeepEqual(sdk, want) || client != p.fresh {
			t.Fatalf("mixed edit fresh bool implementations: fresh=%v SDK=%v client Vec=%v", p.fresh, sdk, client)
		}
	}
}

func TestCandidateEditSDKUserAggregateCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditSDKUserAggregateCommand)
}

func verifyCandidateEditSDKUserAggregateCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:  atlasSDKUserAggregateCommandSource,
		body:    editSDKUserAggregateBody,
		invalid: "{let value=Vec<bool>{.value=42};return 1;}",
		module:  "std::vec",
		name:    "at",
	})
}
