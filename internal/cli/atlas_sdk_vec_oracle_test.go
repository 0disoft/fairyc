package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func TestAtlasPrepareSDKVectorWorkspace(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}`)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("prepare SDK vector workspace: exit %d: %s", code, &diagnostic)
	}
	if len(prepared.partitions) == 0 {
		t.Fatal("prepare SDK vector workspace returned no partitions")
	}
	declarations := map[string]bool{}
	for _, p := range prepared.partitions {
		data, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := atlasmap.DecodeV3(data, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range decoded.Declarations() {
			if d.Key.Module == "std::vec" {
				declarations[d.Key.Name] = true
			}
		}
	}
	for _, name := range []string{"Vec", "empty", "push", "at"} {
		if !declarations[name] {
			t.Fatal("missing authenticated SDK declaration", name)
		}
	}
	result, err := workspacecompiler.CheckGeneric(prepared.workspace)
	if err != nil || !result.Verified || !result.VerificationCapabilities.AllowsStorageTransfers() {
		t.Fatalf("lost SDK MIR verification admission: %v %v", result.Verified, err)
	}
	build, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	result.VerificationCapabilities = admission.Capabilities{}
	if _, _, err := atlasmap.PrepareWorkspaceV3(prepared.workspace, result, build); err == nil || !strings.Contains(err.Error(), "storage-layout") {
		t.Fatalf("snapshot accepted storage MIR without its admission: %v", err)
	}
}
