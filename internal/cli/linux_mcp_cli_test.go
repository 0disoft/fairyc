package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxAtlasMCPCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;import linux_cli_data as linux;@effects(filesystem,console,environment) fn i32 main(){return mcp::arguments_with_help(linux::help());}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasMCP(t, image)
	}, "linux_cli_data")
}

func verifyLinuxAtlasMCP(t *testing.T, image string) {
	t.Helper()
	help := linuxFetchHelp(t)
	for _, alias := range []string{"--help", "-h", "help"} {
		code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), []string{"experimental", "atlas-mcp", alias}, "")
		if code != 0 || out != help || diagnostic != "" {
			t.Fatalf("Linux MCP help %q: (%d,%q,%q)", alias, code, out, diagnostic)
		}
	}
	verifyCandidateMCPCommandWithHelp(t, image, help)
}

// Reuse the actual full Linux CLI per-profile image, with no additional
// compiler/companion build. This does not run in the Windows command probe.
func checkLinuxAtlasMCP(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-mcp", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasMCP(t, image)
	})
}
