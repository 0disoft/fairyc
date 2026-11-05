package cli

import "testing"

func TestCandidateMCPSDKVectorCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console) fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPSDKVectorCommand)
}

func verifyCandidateMCPSDKVectorCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateMCPSDKCommandFixture(t, image, candidateAtlasSDKVectorPopSource, "std::vec", "take_slot")
}
