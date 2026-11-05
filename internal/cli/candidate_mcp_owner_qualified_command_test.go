package cli

import "testing"

func TestCandidateMCPOwnerQualifiedCommand(t *testing.T) {
	const main = `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console)fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPOwnerQualifiedCommand)
}

func verifyCandidateMCPOwnerQualifiedCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range atlasOwnerQualifiedFixtures() {
		switch fixture.name {
		case "associated-parameter", "shared", "exclusive", "value-record-dot", "value-resource-dot", "imported-shared":
		default:
			continue
		}
		t.Run(fixture.name, func(t *testing.T) {
			root := atlasOwnerQualifiedWorkspace(t, fixture)
			module := "main"
			member := workspaceSessionMemberSource
			if fixture.member != "" {
				module, member = "answer", fixture.member
			}
			verifyCandidateMCPWorkspaceFixture(t, image, root, fixture.source, member, module, fixture.declaration)
		})
	}
}
