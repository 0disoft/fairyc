package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/contextpack"
)

func TestExperimentalAtlasPackV3ShowsDiscoveredResource(t *testing.T) {
	root := writeAtlasAggregateWorkspace(t)
	symbol := atlasAggregateSymbol(t, root, "example.session.app", "Token", "resource")
	for _, kind := range []string{"overview", "show", "around"} {
		args := atlasPackOverviewArguments("example.session.app")
		if kind != "overview" {
			args = atlasPackRelationArguments(kind, "example.session.app", symbol.Full)
		}
		args = append(args, "--pack-version", "3")
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("pack %s: %d %s %s", kind, code, output, stderr)
		}
		var pack struct {
			PackVersion string `json:"pack_version"`
			Checksum    string `json:"checksum"`
			Pack        struct {
				MapVersion string `json:"map_version"`
				Facts      struct {
					Show *struct {
						Subject atlasmap.QueryNode `json:"subject"`
					} `json:"show"`
				} `json:"facts"`
				Relations      []atlasmap.QueryRelation `json:"relations"`
				Documentation  []json.RawMessage        `json:"documentation"`
				SourceExcerpts []json.RawMessage        `json:"source_excerpts"`
			} `json:"pack"`
		}
		if err := json.Unmarshal([]byte(output), &pack); err != nil {
			t.Fatal(err)
		}
		if pack.PackVersion != contextpack.PackVersionV3 || pack.Pack.MapVersion != atlasmap.LogicalSchemaV3 {
			t.Fatal("v3 identity lost")
		}
		if _, err := contextpack.ParseChecksumV3(pack.Checksum); err != nil {
			t.Fatal(err)
		}
		if kind == "show" {
			node := pack.Pack.Facts.Show.Subject
			if node.Function != nil || len(node.Specializations) != 2 || len(pack.Pack.Relations) != 2 {
				t.Fatal("aggregate facts absent")
			}
			for _, value := range node.Specializations {
				if value.Aggregate == nil || value.Cleanup == nil || value.Function != nil {
					t.Fatal("aggregate/cleanup collapsed")
				}
			}
			for _, relation := range pack.Pack.Relations {
				if relation.SourceBody != "cleanup" || relation.SourceSpecialization == "" {
					t.Fatal("cleanup source lost")
				}
			}
		}
		if len(pack.Pack.Documentation) != 0 || len(pack.Pack.SourceExcerpts) != 0 || strings.Contains(output, root) || strings.Contains(output, atlasAggregateSource) {
			t.Fatal("pack acquired text")
		}
		code, locked, stderr := runForTest(append(append([]string{"experimental", "atlas-pack"}, args...), "--locked"), root)
		if code != ExitSuccess || stderr != "" || locked != output {
			t.Fatal("locked output differs")
		}
		for _, replacement := range []struct{ flag, value string }{{"--max-output-bytes", "1"}, {"--max-selection-bytes", "1"}, {"--pack-version", "03"}} {
			code, output, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, replaceAtlasPackArgument(args, replacement.flag, replacement.value)...), root)
			if code != ExitInvalidInvocation || output != "" || stderr == "" {
				t.Fatal("invalid pack request leaked partial output")
			}
		}
		code, output, _ = runForTest(append([]string{"experimental", "atlas-pack"}, replaceAtlasPackArgument(args, "--pack-version", "2")...), root)
		if code != ExitSourceDiagnostics || output != "" {
			t.Fatal("legacy pack promoted aggregates")
		}
	}
}
