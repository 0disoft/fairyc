package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The shared query/command executes with Windows workspace I/O. Full Linux
// entry emission separately verifies Linux host selection; native replay is
// attached to that entry's existing per-profile image below.
func TestLinuxAtlasImpactCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;import linux_cli_data as linux;@effects(filesystem,console,environment) fn i32 main(){return atlas::arguments_with_help(linux::help());}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasImpact(t, image)
	}, "linux_cli_data")
}

func verifyLinuxAtlasImpact(t *testing.T, image string) {
	t.Helper()
	help := linuxFetchHelp(t)
	for _, alias := range []string{"--help", "-h", "help"} {
		code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), []string{"experimental", "atlas-impact", alias}, "")
		if code != 0 || out != help || diagnostic != "" {
			t.Fatalf("Linux Atlas help %q: (%d,%q,%q)", alias, code, out, diagnostic)
		}
	}
	code, _, diagnostic := fetchCLIProcess(t, image, t.TempDir(), []string{"experimental", "atlas-impact", "--help"}, "stdout")
	if code != 4 || diagnostic != "" {
		t.Fatalf("Linux Atlas help output failure: %d %q", code, diagnostic)
	}
	verifyCandidateAtlasCommand(t, image)
}

func checkLinuxAtlasImpact(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-impact", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasImpact(t, image)
	})
}
