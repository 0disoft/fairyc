package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxAtlasSymbolsCommand(t *testing.T) {
	// A test-only prefix selects the unchanged default entry without moving any
	// option positions. Both real entry signatures share one compiled image.
	main := `module main;
import project_catalog_cli as catalog;import linux_cli_data as linux;
import std::bytes as bytes;import std::process as process;
@effects(filesystem,console,environment) fn i32 main(){
 var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] prefix=ref raw;
 if(bytes::equal(prefix[..n],b"default-help")){return catalog::arguments();}
 return catalog::arguments_with_help(linux::help());
}`
	candidateAtlasProbeSize(t, "project_catalog_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasSymbols(t, image)
		for _, alias := range []string{"--help", "-h", "help"} {
			for _, action := range [][]string{nil, {"list"}} {
				args := append([]string{"default-help", "atlas-symbols"}, action...)
				args = append(args, alias)
				code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
				if code != 0 || out != usage || diagnostic != "" {
					t.Fatalf("default catalog help %v: (%d,%q,%q)", args, code, out, diagnostic)
				}
			}
		}
	}, "linux_cli_data")
}

func verifyLinuxAtlasSymbols(t *testing.T, image string) {
	t.Helper()
	help := linuxFetchHelp(t)
	for _, alias := range []string{"--help", "-h", "help"} {
		for _, action := range [][]string{nil, {"list"}} {
			args := append([]string{"experimental", "atlas-symbols"}, action...)
			args = append(args, alias)
			code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
			if code != 0 || out != help || diagnostic != "" {
				t.Fatalf("Linux catalog help %v: (%d,%q,%q)", args, code, out, diagnostic)
			}
		}
	}
	verifyCandidateCatalogCommand(t, image)
}

func checkLinuxAtlasSymbols(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-symbols", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasSymbols(t, image)
	})
}
