package cli

import "testing"

func TestCandidateMCPStaticOperationCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console)fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPStaticOperationCommand)
}

func verifyCandidateMCPStaticOperationCommand(t *testing.T, image string) {
	t.Helper()
	root := atlasStaticOperationWorkspace(t, atlasStaticOperationUsed, atlasStaticOperationProvider)
	verifyCandidateMCPWorkspaceFixture(t, image, root, atlasStaticOperationUsed, atlasStaticOperationProvider, "answer", "transform")
}
