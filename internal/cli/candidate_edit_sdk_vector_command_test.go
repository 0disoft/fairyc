package cli

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const editSDKVectorSource = `module main;import std::vec as v;
fn i32 leaf(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}
fn i32 main(){return leaf();}`

// Bool is admitted storage and is absent from the SDK's ordinary view signatures.
// The overlay must therefore create both its Vec type and its callable instances.
const editSDKVectorBody = `{var values=v::empty<bool>();assert(v::push<bool>(ref mut values,true).is_ok);return if(*v::at<bool>(ref values,0)){43}else{1};}`

func TestAtlasSDKVectorEditOracle(t *testing.T) {
	source := "module main;import std::vec as v;fn i32 leaf()" + editSDKVectorBody + "fn i32 main(){return leaf();}"
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		candidateAtlasAggregateOracleFailure(t, root)
		t.Fatalf("edited SDK vector oracle: %d %s", code, &diagnostic)
	}
	found := map[string]bool{}
	for _, partition := range prepared.partitions {
		for _, instance := range partition.Specializations() {
			if instance.Declaration.Module == "std::vec" && len(instance.TypeArguments) == 1 && instance.TypeArguments[0].Name == "bool" {
				found[instance.Declaration.Name] = true
			}
		}
	}
	want := map[string]bool{"Vec": true, "take_slot": true, "empty": true, "push": true, "at": true, "write_slot": true}
	if !reflect.DeepEqual(found, want) {
		t.Fatalf("edited body must request fresh bool implementations: got %v want %v", found, want)
	}
}

func TestCandidateEditSDKVectorCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditSDKVectorCommand)
}

func verifyCandidateEditSDKVectorCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:  editSDKVectorSource,
		body:    editSDKVectorBody,
		invalid: "{return v::missing();}",
		module:  "std::vec",
		name:    "at",
	})
}
