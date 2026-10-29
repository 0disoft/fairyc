package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxCaptureSelection(t *testing.T) {
	for _, name := range []string{"open", "drop", "collect"} {
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
	for _, header := range []string{"spawn.h", "sys/wait.h", "unistd.h", "fcntl.h", "poll.h"} {
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
typedef intptr_t ssize_t;
#define POLLIN 1
#define POLLNVAL 32
struct pollfd {int fd;short events;short revents;};
static int live_fd[32],fd_flags[32],pipes_created,duplications,allocation_calls,closed_stdio;
static int collecting,capture_case,poll_calls,read_calls;
static size_t wanted[2],copied[2];
static ssize_t capture_read(int fd,void *data,size_t count) {
    assert(collecting && live_fd[fd] && (fd_flags[fd]&O_NONBLOCK) && (fd==3 || fd==5) && count>0 && count<=4096);
    ++read_calls;int stream=fd==3 ? 0 : 1;
    if(capture_case==5) {errno=EIO;return -1;}
    if(capture_case==13) {return (ssize_t)count+1;}
    if(capture_case==10 && read_calls==1) {errno=EINTR;return -1;}
    if(capture_case==6 || capture_case==7 || capture_case==8 && poll_calls==0 || capture_case==9 || capture_case==14 ||
        capture_case==2 && stream==0 && copied[1]<8192) {errno=EAGAIN;return -1;}
    size_t available=wanted[stream]-copied[stream];if(count>available) {count=available;}
    for(size_t i=0;i<count;++i) {((uint8_t *)data)[i]=(uint8_t)((copied[stream]+i)*37+(size_t)stream);}
    copied[stream]+=count;return (ssize_t)count;
}
static int capture_poll(struct pollfd *fds,size_t count,int timeout) {
    assert(collecting && count==2 && timeout>0 && timeout<=10);++poll_calls;
    if(capture_case==6) {fds[0].revents=POLLNVAL;return 1;}
    if(capture_case==7) {errno=EIO;return -1;}
    if(capture_case==8 && poll_calls==1) {fake_now+=3000000;errno=EINTR;return -1;}
    if(capture_case==9) {fake_now+=(uint64_t)timeout*1000000;return 0;}
    for(size_t i=0;i<count;++i) {if(fds[i].fd>=0) {fds[i].revents=POLLIN;}}
    return 2;
}
static int capture_clock(int clock,struct timespec *value) {
    if(collecting && capture_case==4) {errno=EIO;return -1;}
    if(collecting && (capture_case==3 || capture_case==14)) {fake_now+=1000000;}
    return probe_clock(clock,value);
}
static int capture_waitid(int type,id_t pid,siginfo_t *info,int flags) {
    int code=probe_waitid(type,pid,info,flags);
    if(code==0 && collecting) {
        if(capture_case==14) {errno=EINTR;return -1;}
        info->si_pid=capture_case==9 || (copied[0]==wanted[0] && copied[1]==wanted[1]) ? 4321 : 0;
    }
    return code;
}
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
#define read capture_read
#define poll capture_poll
#undef clock_gettime
#define clock_gettime capture_clock
#undef waitid
#define waitid capture_waitid
`

const linuxCaptureBoundaryProbe = `
static int descriptors(void) {int count=0;for(int i=0;i<32;++i) {count+=live_fd[i];}return count;}
static void reset_capture(int mode,int closed) {
    scenario=mode;closed_stdio=closed;pipes_created=0;duplications=0;allocation_calls=0;
    memset(live_fd,0,sizeof live_fd);memset(fd_flags,0,sizeof fd_flags);
    if(!closed) {live_fd[0]=1;live_fd[1]=1;live_fd[2]=1;}
}
static void collect_cases(void) {
    static uint8_t out[65538],err[10002];
    const uint8_t data[]="x\0\0a b",cwd[]="/tmp";
    for(capture_case=0;capture_case<=14;++capture_case) {
        reset_capture(0,0);void *raw=NULL;
        assert(fairyc_private_linux_capture_open_v1(data,sizeof data,3,cwd,sizeof cwd,&raw)==0);
        collecting=1;poll_calls=0;read_calls=0;fake_now=1995000000;clock_calls=0;
        kills=0;group_kills=0;wait_calls=0;queries=0;copied[0]=0;copied[1]=0;
        wanted[0]=2;wanted[1]=2;size_t a=2,b=2;
        if(capture_case==1) {wanted[0]=5;a=4;}
        if(capture_case==2) {wanted[0]=9;wanted[1]=9000;a=9;b=9000;}
        if(capture_case==3) {wanted[0]=1000000;a=65536;}
        if(capture_case==11) {wanted[1]=5;b=4;}
        if(capture_case==12 || capture_case==9) {wanted[0]=0;wanted[1]=0;a=0;b=0;}
        memset(out,165,sizeof out);memset(err,165,sizeof err);
        size_t out_len=99,err_len=99;uint32_t kind=99,value=99,pid=99;
        assert(fairyc_private_linux_capture_collect_v1(raw,0,out,a,&out_len,err,b,&err_len,&kind,&value,&pid)==EINVAL);
        assert(kind==0 && value==0 && out_len==0 && err_len==0 && pid==0 && read_calls==0);
        assert(fairyc_private_linux_capture_collect_v1(raw,25,out,1048577,&out_len,err,0,&err_len,&kind,&value,&pid)==EINVAL);
        assert(fairyc_private_linux_capture_collect_v1(raw,25,out,1048576,&out_len,err,1,&err_len,&kind,&value,&pid)==EINVAL);
        assert(fairyc_private_linux_capture_collect_v1(raw,25,NULL,1,&out_len,err,b,&err_len,&kind,&value,&pid)==EINVAL);
        uint32_t code=fairyc_private_linux_capture_collect_v1(raw,capture_case==3 || capture_case==14 ? 5 : 25,
            a==0 ? NULL : out+1,a,&out_len,b==0 ? NULL : err+1,b,&err_len,&kind,&value,&pid);
        assert(pid==4321 && out_len<=a && err_len<=b && out[0]==165 && err[0]==165 && out[a+1]==165 && err[b+1]==165);
        for(size_t i=0;i<out_len;++i) {assert(out[i+1]==(uint8_t)(i*37));}
        for(size_t i=0;i<err_len;++i) {assert(err[i+1]==(uint8_t)(i*37+1));}
        if(capture_case==1 || capture_case==11) {assert(code==0 && kind==4 && value==(capture_case==1 ? 1u : 2u));}
        else if(capture_case==3 || capture_case==9) {assert(code==0 && kind==3 && value==0);}
        else if(capture_case==4 || capture_case==5 || capture_case==6 || capture_case==7 || capture_case==13 || capture_case==14) {
            assert(code==(uint32_t)(capture_case==6 ? EBADF : capture_case==14 ? EINTR : EIO) && kind==0);
        } else {assert(code==0 && kind==1 && value==41 && out_len==a && err_len==b);}
        if(code==0) {assert(kills==1 && group_kills==1 && wait_calls==1);}
        if(capture_case==2) {assert(copied[1]==9000 && copied[0]==9 && poll_calls>=2);}
        if(capture_case==14) {assert(clock_calls<10 && kills==0 && group_kills==0);}
        assert(fairyc_private_linux_capture_collect_v1(raw,25,out,a,&out_len,err,b,&err_len,&kind,&value,&pid)==EINVAL);
        collecting=0;assert(fairyc_private_linux_capture_drop_v1(raw)==0);
        assert(descriptors()==3 && allocations==0);
    }
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
    collect_cases();
    return 0;
}
`

const linuxCaptureNativeProbe = `
#include <assert.h>
#include <stdio.h>
#include <sys/prctl.h>
static void write_all(int fd,const void *raw,size_t size) {
    const uint8_t *data=raw;
    while(size>0) {ssize_t n=write(fd,data,size);if(n<0 && errno==EINTR) {continue;}assert(n>0);data+=n;size-=(size_t)n;}
}
static void *launch(const char *image,const char *mode) {
    char packed[8192];size_t a=strlen(image)+1,b=strlen(mode)+1;assert(a+b<=sizeof packed);
    memcpy(packed,image,a);memcpy(packed+a,mode,b);void *raw=NULL;
    assert(fairyc_private_linux_capture_open_v1((uint8_t *)packed,a+b,2,(const uint8_t *)"/tmp",5,&raw)==0);return raw;
}
int main(int argc,char **argv) {
    const uint8_t binary[]={0,1,255,0,126},error[]={255,0,7};
    if(argc>1) {
        alarm(3);
        if(strcmp(argv[1],"zero")==0) {return 41;}
        if(strcmp(argv[1],"bytes")==0) {write_all(1,binary,sizeof binary);write_all(2,error,sizeof error);return 127;}
        if(strcmp(argv[1],"silent")==0) {for(;;) {pause();}}
        if(strcmp(argv[1],"tail")==0 || strcmp(argv[1],"escaped")==0) {
            int ready[2];assert(pipe2(ready,O_CLOEXEC)==0);pid_t descendant=fork();assert(descendant>=0);
            if(descendant==0) {
                alarm(3);assert(close(ready[0])==0);
                if(strcmp(argv[1],"escaped")==0) {assert(setsid()>0);}
                write_all(ready[1],"r",1);assert(close(ready[1])==0);for(;;) {pause();}
            }
            assert(close(ready[1])==0);char marker;assert(read(ready[0],&marker,1)==1 && close(ready[0])==0);
            uint32_t id=(uint32_t)descendant;write_all(2,&id,sizeof id);write_all(1,binary,sizeof binary);return 17;
        }
        uint8_t chunk[4096];
        if(strcmp(argv[1],"flood")==0) {
            for(int channel=2;channel>=1;--channel) {
                for(size_t i=0;i<sizeof chunk;++i) {chunk[i]=(uint8_t)(i*37+(size_t)(channel-1));}
                for(int i=0;i<32;++i) {write_all(channel,chunk,sizeof chunk);}
            }
            return 41;
        }
        memset(chunk,120,sizeof chunk);for(;;) {write_all(1,chunk,sizeof chunk);write_all(2,chunk,sizeof chunk);}
    }
    char image[4096];ssize_t size=readlink("/proc/self/exe",image,sizeof image-1);assert(size>0);image[size]=0;
    char packed[8192];memcpy(packed,image,(size_t)size+1);memcpy(packed+size+1,"child",6);
    void *raw=NULL;assert(fairyc_private_linux_capture_open_v1((uint8_t *)packed,(size_t)size+7,2,(const uint8_t *)"/tmp",5,&raw)==0);
    fy_linux_capture *c=raw;uint32_t pid=c->pid;int a=c->readers[0],b=c->readers[1];
    assert(a>=3 && b>=3 && (fcntl(a,F_GETFL)&O_NONBLOCK) && (fcntl(b,F_GETFL)&O_NONBLOCK));
    assert(fairyc_private_linux_capture_drop_v1(raw)==0);
    errno=0;assert(fcntl(a,F_GETFD)==-1 && errno==EBADF);errno=0;assert(fcntl(b,F_GETFD)==-1 && errno==EBADF);
    int status;errno=0;assert(waitpid((pid_t)pid,&status,WNOHANG)==-1 && errno==ECHILD);
    assert(prctl(PR_SET_CHILD_SUBREAPER,1,0,0,0)==0);
    const char *modes[]={"bytes","flood","limit","silent","zero","tail","escaped"};
    static uint8_t output[131074],errors[131074];
    for(size_t i=0;i<sizeof modes/sizeof modes[0];++i) {
        raw=launch(image,modes[i]);c=raw;a=c->readers[0];b=c->readers[1];uint32_t descendant=0;
        if(i>=5) {struct pollfd ready={b,POLLIN,0};assert(poll(&ready,1,1000)>0);assert(read(b,&descendant,sizeof descendant)==(ssize_t)sizeof descendant && descendant>1);}
        size_t out_cap=i==0 || i>=5 ? 5 : i==1 ? 131072 : i==2 ? 8 : 0;
        size_t err_cap=i==0 ? 3 : i==1 ? 131072 : i==2 ? 8 : 0;
        memset(output,165,sizeof output);memset(errors,165,sizeof errors);size_t out_len=0,err_len=0;uint32_t kind=0,value=0;
        assert(fairyc_private_linux_capture_collect_v1(raw,i==3 || i==6 ? 25 : 2000,
            out_cap==0 ? NULL : output+1,out_cap,&out_len,err_cap==0 ? NULL : errors+1,err_cap,&err_len,&kind,&value,&pid)==0);
        assert(out_len<=out_cap && err_len<=err_cap && output[0]==165 && output[out_cap+1]==165 && errors[0]==165 && errors[err_cap+1]==165);
        if(i==0) {assert(kind==1 && value==127 && out_len==5 && err_len==3 && memcmp(output+1,binary,5)==0 && memcmp(errors+1,error,3)==0);}
        if(i==1) {assert(kind==1 && value==41 && out_len==131072 && err_len==131072);for(size_t n=0;n<out_len;++n) {assert(output[n+1]==(uint8_t)(n*37) && errors[n+1]==(uint8_t)(n*37+1));}}
        if(i==2) {assert(kind==4 && (value==1 || value==2));}
        if(i==3 || i==6) {assert(kind==3 && value==0);}
        if(i==4) {assert(kind==1 && value==41 && out_len==0 && err_len==0);}
        if(i==5) {assert(kind==1 && value==17 && out_len==5);}
        if(i==6) {siginfo_t owned;assert(waitid(P_PID,(id_t)descendant,&owned,WEXITED|WNOWAIT|WNOHANG)==0);assert(kill((pid_t)descendant,SIGKILL)==0);}
        if(i>=5) {assert(waitpid((pid_t)descendant,&status,0)==(pid_t)descendant && WIFSIGNALED(status) && WTERMSIG(status)==SIGKILL);}
        assert(fairyc_private_linux_capture_drop_v1(raw)==0);
        errno=0;assert(fcntl(a,F_GETFD)==-1 && errno==EBADF);errno=0;assert(fcntl(b,F_GETFD)==-1 && errno==EBADF);
    }
    errno=0;assert(waitpid(-1,&status,WNOHANG)==-1 && errno==ECHILD);
    return 0;
}
`
