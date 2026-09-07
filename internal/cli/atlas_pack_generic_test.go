package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestExperimentalAtlasPackV2PreservesGenericFacts(t *testing.T) {
	root := writeAtlasGenericWorkspace(t)
	symbol := atlasGenericFunctionSymbol(t, root, "example.session.app", "id")
	for _, kind := range []string{"overview", "show", "around"} {
		args := atlasPackOverviewArguments("example.session.app")
		if kind != "overview" {
			args = atlasPackRelationArguments(kind, "example.session.app", symbol.Visible)
		}
		args = append(args, "--pack-version", "2")
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("generic pack %s: %d %s %s", kind, code, output, stderr)
		}
		var pack struct {
			PackVersion string `json:"pack_version"`
			Checksum    string `json:"checksum"`
			Pack        struct {
				MapVersion    string `json:"map_version"`
				Compatibility struct {
					GenericInterfaceHash string `json:"generic_interface_hash"`
				} `json:"compatibility"`
				Facts struct {
					Show *struct {
						Subject atlasmap.QueryNode `json:"subject"`
					} `json:"show"`
				} `json:"facts"`
				Relations      []atlasmap.QueryRelation        `json:"relations"`
				Completeness   []semanticsnapshot.Completeness `json:"completeness"`
				Documentation  []json.RawMessage               `json:"documentation"`
				SourceExcerpts []json.RawMessage               `json:"source_excerpts"`
			} `json:"pack"`
		}
		if err := json.Unmarshal([]byte(output), &pack); err != nil {
			t.Fatal(err)
		}
		if pack.PackVersion != contextpack.PackVersionV2 || pack.Pack.MapVersion != atlasmap.LogicalSchemaV2 || pack.Pack.Compatibility.GenericInterfaceHash == "" {
			t.Fatal("wrong generic pack identity")
		}
		if _, err := contextpack.ParseChecksumV2(pack.Checksum); err != nil {
			t.Fatal(err)
		}
		if len(pack.Pack.Documentation) != 0 || len(pack.Pack.SourceExcerpts) != 0 || strings.Contains(output, root) || strings.Contains(output, atlasGenericRootSource) {
			t.Fatal("generic pack acquired text")
		}
		if kind == "show" {
			node := pack.Pack.Facts.Show.Subject
			if node.Function != nil || len(node.Specializations) != 2 || len(pack.Pack.Relations) != 2 {
				t.Fatal("generic observations missing")
			}
			for _, relation := range pack.Pack.Relations {
				if relation.SourceSpecialization == "" || relation.Target == nil || relation.Target.Specialization == "" {
					t.Fatal("concrete endpoint erased")
				}
			}
		}
		code, locked, stderr := runForTest(append(append([]string{"experimental", "atlas-pack"}, args...), "--locked"), root)
		if code != ExitSuccess || stderr != "" || locked != output {
			t.Fatal("locked pack differs")
		}
		code, tiny, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, replaceAtlasPackArgument(args, "--max-output-bytes", "1")...), root)
		if code != ExitInvalidInvocation || tiny != "" || stderr == "" {
			t.Fatal("output bound failed")
		}
	}
}

func TestExperimentalAtlasPackVersionPolicy(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	symbol := atlasFunctionSymbolForPackage(t, root, "example.session.app", "main")
	for _, kind := range []string{"overview", "show", "around"} {
		args := atlasPackOverviewArguments("example.session.app")
		if kind != "overview" {
			args = atlasPackRelationArguments(kind, "example.session.app", symbol.Visible)
		}
		args = append([]string{"experimental", "atlas-pack"}, args...)
		code, legacy, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatal("legacy pack failed")
		}
		code, explicit, stderr := runForTest(append(args, "--pack-version", "1"), root)
		if code != ExitSuccess || stderr != "" || explicit != legacy {
			t.Fatal("explicit v1 changed default output")
		}
		for _, suffix := range [][]string{{"--pack-version"}, {"--pack-version", "0"}, {"--pack-version", "02"}, {"--pack-version", "2", "--pack-version", "2"}} {
			code, output, stderr := runForTest(append(args, suffix...), root)
			if code != ExitInvalidInvocation || output != "" || stderr == "" {
				t.Fatal("invalid pack version accepted")
			}
		}
	}
	args := append(atlasPackRelationArguments("show", "example.session.app", symbol.Visible)[1:], "--pack-version", "2")
	if _, err := parseAtlasRelationOptions("atlas-evidence show", "show", args); err == nil {
		t.Fatal("shared relation parser activated unversioned evidence consumer")
	}
}
