package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxCaptureSelection(t *testing.T) {
	for _, name := range []string{"open", "drop"} {
		source, err := LinuxSource("fairyc_private_linux_capture_" + name + "_v1(")
		if err != nil || source != linuxProcess+"\n"+linuxCapture {
			t.Fatalf("capture %s: %v", name, err)
		}
	}
	if _, err := LinuxSource("fairyc_private_linux_capture_unknown_v1("); err == nil {
		t.Fatal("unknown capture admitted")
	}
	source, err := LinuxSource("fairyc_private_linux_host_path_v1( fairyc_private_linux_capture_open_v1(")
	if err != nil || source != linuxProcess+"\n"+linuxCapture+"\n"+linuxPaths {
		t.Fatalf("capture/path order: %v", err)
	}
}

func TestLinuxCaptureAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxProcess + "\n" + linuxCapture
	for _, header := range []string{"spawn.h", "sys/wait.h", "unistd.h", "fcntl.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxProcessPOSIXStub+linuxCapturePOSIXStub+source+linuxCaptureBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "capture.exe")
		})
	}
}

func TestLinuxCaptureAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires GNU/Linux amd64, GCC and Clang")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxProcess+"\n"+linuxCapture+linuxCaptureNativeProbe, "typedef int fy_capture_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "capture")
			})
		}
	}
}

const linuxCapturePOSIXStub = `
#define O_CLOEXEC 524288
#define O_NONBLOCK 2048
#define F_DUPFD_CLOEXEC 1030
#define F_GETFL 3
#define F_SETFL 4
static int live_fd[32],fd_flags[32],pipes_created,duplications,allocation_calls,closed_stdio;
static int capture_fd(void) { for(int i=0;i<32;++i) { if(!live_fd[i]) { live_fd[i]=1;fd_flags[i]=O_CLOEXEC;return i; } } assert(0);return -1; }
static int capture_pipe(int pair[2],int flags) {
    assert(flags==O_CLOEXEC);++pipes_created;
    if(scenario==100 || scenario==102 && pipes_created==2) {errno=EMFILE;return -1;}
    pair[0]=capture_fd();pair[1]=capture_fd();return 0;
}
static int capture_close(int fd) {
    assert(fd>=0 && fd<32 && live_fd[fd]);live_fd[fd]=0;
    if(scenario==109) {errno=EINTR;return -1;} return 0;
}
static int capture_fcntl(int fd,int command,...) {
    assert(fd>=0 && fd<32 && live_fd[fd]);
    if(command==F_DUPFD_CLOEXEC) {
        if(scenario==101) {errno=EMFILE;return -1;}
        for(int i=3;i<32;++i) {if(!live_fd[i]) {live_fd[i]=1;fd_flags[i]=fd_flags[fd];return i;}}
        assert(0);return -1;
    }
    if(command==F_GETFL) {if(scenario==103) {errno=EIO;return -1;}return fd_flags[fd];}
    assert(command==F_SETFL);if(scenario==104) {errno=EIO;return -1;}
    fd_flags[fd]|=O_NONBLOCK;return 0;
}
static int capture_dup2(posix_spawn_file_actions_t *p,int from,int to) {
    assert(live_fd[from] && !(fd_flags[from]&O_NONBLOCK));++duplications;
    if(scenario==105 && to==1 || scenario==106 && to==2) {return EBADF;}
    return probe_dup2(p,from,to);
}
static int capture_closefrom(posix_spawn_file_actions_t *p,int low) { assert(duplications==2); return probe_closefrom(p,low); }
static void *capture_alloc(size_t count,size_t size) {
    ++allocation_calls;if(scenario==108 && allocation_calls==2) {return NULL;}
    return probe_calloc(count,size);
}
#define pipe2 capture_pipe
#define close capture_close
#define fcntl capture_fcntl
#undef calloc
#define calloc capture_alloc
#undef posix_spawn_file_actions_adddup2
#define posix_spawn_file_actions_adddup2 capture_dup2
#undef posix_spawn_file_actions_addclosefrom_np
#define posix_spawn_file_actions_addclosefrom_np capture_closefrom
`

const linuxCaptureBoundaryProbe = `
static int descriptors(void) {int count=0;for(int i=0;i<32;++i) {count+=live_fd[i];}return count;}
static void reset_capture(int mode,int closed) {
    scenario=mode;closed_stdio=closed;pipes_created=0;duplications=0;allocation_calls=0;
    memset(live_fd,0,sizeof live_fd);memset(fd_flags,0,sizeof fd_flags);
    if(!closed) {live_fd[0]=1;live_fd[1]=1;live_fd[2]=1;}
}
int main(void) {
    const uint8_t data[]="x\0\0a b",cwd[]="/tmp";
    for(int closed=0;closed<2;++closed) {
        reset_capture(0,closed);void *raw=NULL;
        assert(fairyc_private_linux_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,&raw)==0 && raw!=NULL);
        fy_linux_capture *c=raw;assert(c->pid==4321 && c->readers[0]>=3 && c->readers[1]>=3 && c->readers[0]!=c->readers[1]);
        assert((fd_flags[c->readers[0]]&O_NONBLOCK) && (fd_flags[c->readers[1]]&O_NONBLOCK));
        assert(descriptors()==(closed ? 2 : 5) && allocations==1 && actions_live==0 && attrs_live==0);
        assert(fairyc_private_linux_capture_drop_v1(raw)==0 && descriptors()==(closed ? 0 : 3) && allocations==0);
    }
    const int modes[]={1,2,3,4,5,6,17,18,19,100,101,102,103,104,105,106,108,109};
    for(size_t i=0;i<sizeof modes/sizeof modes[0];++i) {
        reset_capture(modes[i],modes[i]==101);void *raw=(void *)(uintptr_t)1;
        assert(fairyc_private_linux_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,&raw)!=0 && raw==NULL);
        assert(descriptors()==(closed_stdio ? 0 : 3) && allocations==0 && actions_live==0 && attrs_live==0);
    }
    assert(fairyc_private_linux_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,NULL)==EINVAL);
    assert(fairyc_private_linux_capture_drop_v1(NULL)==0);
    return 0;
}
`

const linuxCaptureNativeProbe = `
#include <assert.h>
#include <stdio.h>
int main(int argc,char **argv) {
    if(argc>1) {alarm(3);for(;;) { assert(write(1,"x",1)==1);assert(write(2,"y",1)==1); }}
    (void)argv;
    char image[4096];ssize_t size=readlink("/proc/self/exe",image,sizeof image-1);assert(size>0);image[size]=0;
    char packed[8192];memcpy(packed,image,(size_t)size+1);memcpy(packed+size+1,"child",6);
    void *raw=NULL;assert(fairyc_private_linux_capture_open_v1((uint8_t *)packed,(size_t)size+7,2,(const uint8_t *)"/tmp",5,&raw)==0);
    fy_linux_capture *c=raw;uint32_t pid=c->pid;int a=c->readers[0],b=c->readers[1];
    assert(a>=3 && b>=3 && (fcntl(a,F_GETFL)&O_NONBLOCK) && (fcntl(b,F_GETFL)&O_NONBLOCK));
    assert(fairyc_private_linux_capture_drop_v1(raw)==0);
    errno=0;assert(fcntl(a,F_GETFD)==-1 && errno==EBADF);errno=0;assert(fcntl(b,F_GETFD)==-1 && errno==EBADF);
    int status;errno=0;assert(waitpid((pid_t)pid,&status,WNOHANG)==-1 && errno==ECHILD);
    return 0;
}
`
