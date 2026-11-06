package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxAtlasCallsCommand(t *testing.T) {
	main := `module main;
import project_calls_cli as calls;import linux_cli_data as linux;
import std::bytes as bytes;import std::process as process;
@effects(filesystem,console,environment) fn i32 main(){
 var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] prefix=ref raw;
 if(bytes::equal(prefix[..n],b"default-help")){return calls::arguments();}
 return calls::arguments_with_help(linux::help());
}`
	candidateAtlasProbeSize(t, "project_calls_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasCalls(t, image)
		verifyLinuxAtlasCommandHelp(t, image, "default-help", "atlas-calls", usage, [][]string{nil, {"list"}})
	}, "linux_cli_data")
}

// Help has the same ordering and exact output requirements for both entries.
// The test-only default prefix never enters the real Linux dispatcher.
func verifyLinuxAtlasCommandHelp(t *testing.T, image, prefix, command, help string, actions [][]string) {
	t.Helper()
	for _, alias := range []string{"--help", "-h", "help"} {
		for _, action := range actions {
			args := append([]string{prefix, command}, action...)
			args = append(args, alias)
			code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
			if code != 0 || out != help || diagnostic != "" {
				t.Fatalf("help %v: (%d,%q,%q)", args, code, out, diagnostic)
			}
		}
	}
}

func verifyLinuxAtlasCalls(t *testing.T, image string) {
	t.Helper()
	verifyLinuxAtlasCommandHelp(t, image, "experimental", "atlas-calls", linuxFetchHelp(t), [][]string{nil, {"list"}})
	verifyCandidateCallsCommand(t, image)
}

func checkLinuxAtlasCalls(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-calls", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasCalls(t, image)
	})
}
