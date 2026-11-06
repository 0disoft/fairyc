package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxAtlasPackCommand(t *testing.T) {
	main := `module main;
import project_pack_cli as packs;import linux_cli_data as linux;
import std::bytes as bytes;import std::process as process;
@effects(filesystem,console,environment) fn i32 main(){
 var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] prefix=ref raw;
 if(bytes::equal(prefix[..n],b"default-help")){return packs::arguments();}
 return packs::arguments_with_help(linux::help());
}`
	candidateAtlasProbeSize(t, "project_pack_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasPack(t, image)
		verifyLinuxAtlasCommandHelp(t, image, "default-help", "atlas-pack", usage, linuxPackHelpActions())
	}, "linux_cli_data")
}

func linuxPackHelpActions() [][]string {
	return [][]string{nil, {"overview"}, {"show"}, {"around"}}
}

func verifyLinuxAtlasPack(t *testing.T, image string) {
	t.Helper()
	verifyLinuxAtlasCommandHelp(t, image, "experimental", "atlas-pack", linuxFetchHelp(t), linuxPackHelpActions())
	verifyCandidatePackCommand(t, image)
}

func checkLinuxAtlasPack(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-pack", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasPack(t, image)
	})
}
