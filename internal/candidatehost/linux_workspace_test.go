package candidatehost

import (
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxWorkspaceObservation(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_workspace_path_kind_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxWorkspace {
		t.Fatalf("workspace observation selection: %v", err)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native Linux workspace observation requires Linux amd64; selection passed")
	}
	toolchain, err := cdriver.ResolveNativeToolchain(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	probe := source + `
#include <assert.h>
int main(void) {
    uint32_t mode=99;
    assert(fairyc_private_linux_workspace_path_kind_v1(NULL,&mode)==EINVAL && mode==0);
    assert(fairyc_private_linux_workspace_path_kind_v1((const uint8_t *)"",&mode)==EINVAL);
    assert(fairyc_private_linux_workspace_path_kind_v1((const uint8_t *)".",&mode)==0 && S_ISDIR(mode));
    assert(symlink(".","link")==0);
    assert(fairyc_private_linux_workspace_path_kind_v1((const uint8_t *)"link",&mode)==0 && S_ISLNK(mode));
    assert(unlink("link")==0);
    assert(fairyc_private_linux_workspace_path_kind_v1((const uint8_t *)"missing",&mode)==ENOENT);
    return 0;
}
`
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: probe, RuntimeSource: "typedef int fy_workspace_runtime;\n"})
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{WorkingDirectory: t.TempDir(), RejectOutput: true, Timeout: 10 * time.Second})
		if err != nil || outcome.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", profile.Name, outcome, err)
		}
	}
}
