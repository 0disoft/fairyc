package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxClockSelection(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_monotonic_ms_v1(")
	if err != nil || source != linuxClock {
		t.Fatalf("clock source: %v", err)
	}
	source, err = LinuxSource("fairyc_private_linux_monotonic_ms_v1( fairyc_private_linux_root_file_read_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxClock {
		t.Fatalf("feature macro order: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_monotonic_unknown_v1("); err == nil {
		t.Fatal("unknown clock admitted")
	}
}

func TestLinuxClockAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC clock substitute; native Linux replay is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			// UCRT supplies timespec; only the Linux clock syscall is substituted.
			source := linuxClockStub + strings.ReplaceAll(linuxClock, "#include <time.h>\n", "") + linuxClockBoundaryProbe
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, source)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "clock.exe")
		})
	}
}

func TestLinuxClockAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires GNU/Linux amd64, GCC and Clang; execution stays batched")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxClock+`
#include <assert.h>
int main(void){size_t first=0,second=0;assert(fairyc_private_linux_monotonic_ms_v1(&first)==0 && fairyc_private_linux_monotonic_ms_v1(&second)==0 && second>=first);assert(fairyc_private_linux_monotonic_ms_v1(NULL)==EINVAL);return 0;}
`, "typedef int fy_clock_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "clock")
			})
		}
	}
}

const linuxClockStub = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <time.h>
#ifndef EOVERFLOW
#define EOVERFLOW 75
#endif
#define CLOCK_MONOTONIC 1
static int scenario,calls;
static int probe_clock_gettime(int clock_id,struct timespec *output){
    assert(clock_id==CLOCK_MONOTONIC);++calls;output->tv_sec=7;output->tv_nsec=999999999;
    if(scenario==1){errno=EIO;return -1;}if(scenario==2){output->tv_sec=-1;}if(scenario==3){output->tv_nsec=-1;}if(scenario==4){output->tv_nsec=1000000000;}
    if(scenario==5 || scenario==6){output->tv_sec=(int64_t)(SIZE_MAX/1000);output->tv_nsec=(long)((SIZE_MAX%1000+(scenario==6?1u:0u))*1000000u);}
    if(scenario==7){output->tv_sec=(int64_t)(SIZE_MAX/1000+1);output->tv_nsec=0;}if(scenario==8){errno=EINTR;return -1;}
    if(scenario==9){output->tv_nsec=999999;}return 0;
}
#define clock_gettime probe_clock_gettime
`

const linuxClockBoundaryProbe = `
int main(void){
    size_t current=99;scenario=0;assert(fairyc_private_linux_monotonic_ms_v1(&current)==0 && current==7999 && calls==1);
    assert(fairyc_private_linux_monotonic_ms_v1(NULL)==EINVAL && calls==1);
    for(scenario=1;scenario<=9;++scenario){current=99;int before=calls;uint32_t code=fairyc_private_linux_monotonic_ms_v1(&current);assert(calls==before+1);
        if(scenario==1){assert(code==EIO && current==0);}else if(scenario==5){assert(code==0 && current==SIZE_MAX);}else if(scenario==8){assert(code==EINTR && current==0);}else if(scenario==9){assert(code==0 && current==7000);}else{assert(code==EOVERFLOW && current==0);}
    }return 0;
}
`
