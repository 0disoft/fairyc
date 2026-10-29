package candidatehost

import (
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxBuildScratch(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_build_scratch_open_v1( fairyc_private_linux_build_scratch_drop_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxBuildScratch {
		t.Fatalf("build staging selection: %v", err)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native GNU build staging requires Linux amd64; selection passed")
	}
	tc, err := cdriver.ResolveNativeToolchain(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	probe := source + `
#include <assert.h>
int main(void){
    char root[4096];assert(getcwd(root,sizeof root)!=NULL);
    int sentinel=open("outside",O_WRONLY|O_CREAT|O_EXCL,0600);assert(sentinel>=0);assert(write(sentinel,"keep",4)==4);assert(close(sentinel)==0);
    uint8_t path[4096];size_t length=0,handle=0;
    assert(fairyc_private_linux_build_scratch_open_v1((const uint8_t *)root,strlen(root)+1,path,sizeof path,&length,&handle)==0);
    assert(handle!=0 && length>0 && path[length]==0);
    struct fy_linux_build_scratch *stage=(struct fy_linux_build_scratch *)(uintptr_t)handle;
    struct stat info;assert(fstat(stage->fd,&info)==0 && (info.st_mode&0777)==0700);
    assert(mkdirat(stage->fd,"nested",0700)==0);
    int file=openat(stage->fd,"nested/part.o",O_WRONLY|O_CREAT|O_EXCL,0600);assert(file>=0);assert(close(file)==0);
    assert(symlinkat(root,stage->fd,"link")==0);
    assert(fairyc_private_linux_build_scratch_drop_v1(handle)==0);
    assert(lstat((const char *)path,&info)<0 && errno==ENOENT);
    char kept[4];file=open("outside",O_RDONLY);assert(file>=0 && read(file,kept,4)==4 && memcmp(kept,"keep",4)==0);assert(close(file)==0);
    assert(unlink("outside")==0 && fairyc_private_linux_build_scratch_drop_v1(0)==0);return 0;
}
`
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: probe, RuntimeSource: "\n"})
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{WorkingDirectory: t.TempDir(), RejectOutput: true, Timeout: 10 * time.Second})
		if err != nil || outcome.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", profile.Name, outcome, err)
		}
	}
}
