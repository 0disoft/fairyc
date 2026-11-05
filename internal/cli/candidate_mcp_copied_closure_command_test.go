package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateMCPCopiedClosureCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console)fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPCopiedClosureCommand)
}

func verifyCandidateMCPCopiedClosureCommand(t *testing.T, image string) {
	t.Helper()
	for _, family := range []string{"copied-capture", "generic-owner"} {
		t.Run(family, func(t *testing.T) {
			source := atlasCopiedClosureSource(t, family)
			root := writeWorkspaceSessionFixture(t, source)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("MCP closure selector oracle: %d %s", code, &diagnostic)
			}
			name := ""
			for _, partition := range prepared.partitions {
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				for _, symbol := range partition.Symbols() {
					if strings.HasPrefix(symbol.Declaration.Name, "_fy_closure_") {
						name = symbol.Declaration.Name
						break
					}
				}
			}
			if name == "" {
				t.Fatal("missing private closure helper selector")
			}
			verifyCandidateMCPWorkspaceFixture(t, image, root, source, workspaceSessionMemberSource, "main", name)
		})
	}
}
