package candidatehost

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxHostSymbolSelection(t *testing.T) {
	for _, header := range []string{"ordinary header", "fairyc_private_linux_host_path_v1"} {
		if source, err := LinuxSource(header); err != nil || source != "" {
			t.Fatalf("unused Linux support selected: %v", err)
		}
	}
	for _, header := range []string{
		"fairyc_private_linux_host_path_v1(",
		"extern uint32_t fairyc_private_linux_host_path_v1 \n (",
	} {
		if source, err := LinuxSource(header); err != nil || source != linuxPaths {
			t.Fatalf("missing Linux path adapter: %v", err)
		}
	}
	for _, header := range []string{
		"fairyc_private_linux_unknown_v1(",
		"fairyc_private_linux_host_path_v1( fairyc_private_linux_unknown_v1(",
		"fairyc_private_windows_current_path_v1(",
	} {
		if _, err := LinuxSource(header); err == nil {
			t.Fatalf("unsupported host admitted: %s", header)
		}
	}
}

// Exercise the production adapter with deterministic POSIX-call substitutes.
// This proves bounds/error handling on Windows, not native Linux OS behavior.
func TestLinuxPathAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("deterministic adapter probe uses MSVC; native Linux has a separate probe")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxPathPOSIXStub + strings.ReplaceAll(linuxPaths, "#include <unistd.h>\n", "") + linuxPathBoundaryProbe
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, source)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "paths.exe")
		})
	}
}

func TestLinuxPathAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Linux amd64, GCC and Clang")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxPaths+linuxPathNativeProbe, "typedef int fy_path_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "paths")
			})
		}
	}
}

func runLinuxPathProbe(t *testing.T, image []byte, name string) {
	t.Helper()
	exe := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(exe, image, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("path adapter execution: %v %s", err, output)
	}
}

const linuxPathPOSIXStub = `
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
typedef intptr_t ssize_t;
static int scenario;
static char oversized[4097];
static char *probe_getcwd(char *data,size_t capacity) {
    if(scenario==1) { errno=ENOENT; return NULL; }
    if(capacity<6) { errno=ERANGE; return NULL; }
    memcpy(data,"/work",6); return data;
}
static char *probe_getenv(const char *name) {
    assert(strcmp(name,"TMPDIR")==0);
    if(scenario==4) { return oversized; }
    if(scenario==5) { return NULL; }
    if(scenario==6) { return ""; }
    return "relative temp";
}
static ssize_t probe_readlink(const char *path,char *data,size_t capacity) {
    assert(strcmp(path,"/proc/self/exe")==0);
    if(scenario==2) { errno=ENOENT; return -1; }
    if(scenario==3) { memset(data,'x',capacity); return (ssize_t)capacity; }
    assert(capacity>=11); memcpy(data,"/bin/fairyc",11); return 11;
}
#define getcwd probe_getcwd
#define getenv probe_getenv
#define readlink probe_readlink
`

const linuxPathBoundaryProbe = `
int main(void) {
    uint8_t data[4098]; uint32_t length=99;
    memset(data,85,sizeof data); memset(oversized,'x',4096); oversized[4096]=0;
    assert(fairyc_private_linux_host_path_v1(1,data,4096,&length)==0 && length==5 && strcmp((char *)data,"/work")==0);
    assert(fairyc_private_linux_host_path_v1(2,data,4096,&length)==0 && length==13 && strcmp((char *)data,"relative temp")==0);
    assert(fairyc_private_linux_host_path_v1(3,data,4096,&length)==0 && length==11 && data[11]==0);
    scenario=1; assert(fairyc_private_linux_host_path_v1(1,data,4096,&length)==ENOENT && length==0);
    scenario=2; assert(fairyc_private_linux_host_path_v1(3,data,4096,&length)==ENOENT && length==0);
    scenario=3; assert(fairyc_private_linux_host_path_v1(3,data,4096,&length)==ERANGE && length==0 && data[4096]==85);
    scenario=4; assert(fairyc_private_linux_host_path_v1(2,data,4096,&length)==ERANGE && length==0 && data[4096]==85);
    oversized[4095]=0; assert(fairyc_private_linux_host_path_v1(2,data,4096,&length)==0 && length==4095 && data[4095]==0 && data[4096]==85);
    scenario=5; assert(fairyc_private_linux_host_path_v1(2,data,4096,&length)==0 && length==0 && data[0]==0);
    scenario=6; assert(fairyc_private_linux_host_path_v1(2,data,4096,&length)==0 && length==0);
    assert(fairyc_private_linux_host_path_v1(99,data,4096,&length)==EINVAL && length==0);
    assert(fairyc_private_linux_host_path_v1(1,data,0,&length)==EINVAL && length==0);
    assert(fairyc_private_linux_host_path_v1(1,NULL,4096,&length)==EINVAL && length==0);
    assert(fairyc_private_linux_host_path_v1(1,data,4096,NULL)==EINVAL);
    return 0;
}
`

const linuxPathNativeProbe = `
#include <assert.h>
int main(void) {
    uint8_t data[4096]; char expected[4096]; uint32_t length=0;
    assert(getcwd(expected,sizeof expected)!=NULL);
    assert(fairyc_private_linux_host_path_v1(1,data,sizeof data,&length)==0 && strcmp((char *)data,expected)==0 && length==strlen(expected));
    ssize_t size=readlink("/proc/self/exe",expected,sizeof expected);
    assert(size>0 && (size_t)size<sizeof expected); expected[size]=0;
    assert(fairyc_private_linux_host_path_v1(3,data,sizeof data,&length)==0 && strcmp((char *)data,expected)==0);
    assert(unsetenv("TMPDIR")==0);
    assert(fairyc_private_linux_host_path_v1(2,data,sizeof data,&length)==0 && length==0);
    assert(setenv("TMPDIR","relative /\xed\x95\x9c \xf0\x9f\x98\x80",1)==0);
    assert(fairyc_private_linux_host_path_v1(2,data,sizeof data,&length)==0 && strcmp((char *)data,getenv("TMPDIR"))==0);
    assert(fairyc_private_linux_host_path_v1(3,data,1,&length)==ERANGE && length==0);
    return 0;
}
`
