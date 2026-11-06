package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateLinuxHTTPSSelection(t *testing.T) {
	for _, name := range []string{"open", "headers", "read", "close"} {
		source, err := LinuxSource("fairyc_private_linux_https_" + name + "_v1(")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(source, "#ifndef _GNU_SOURCE\n") {
			t.Fatal("GNU selection must precede every libc header")
		}
		for _, asset := range []string{linuxClock, linuxHTTPSProtocol, linuxHTTPSWorker, linuxHTTPSReader, linuxHTTPSParent} {
			if strings.Count(source, asset) != 1 {
				t.Fatal("missing or duplicated HTTPS asset")
			}
		}
	}
	source, err := LinuxSource("fairyc_private_linux_https_open_v1( fairyc_private_linux_https_read_v1( fairyc_private_linux_monotonic_ms_v1( fairyc_private_linux_root_file_read_v1(")
	if err != nil || strings.Count(source, linuxClock) != 1 || strings.Count(source, linuxHTTPSParent) != 1 || !strings.Contains(source, linuxRootedFile) {
		t.Fatalf("mixed source selection: %v", err)
	}
	for _, header := range []string{"", "fairyc_private_linux_monotonic_ms_v1("} {
		source, err := LinuxSource(header)
		if err != nil || strings.Contains(source, "curl") || strings.Contains(source, "fy_linux_https_") {
			t.Fatalf("unrequested producer selected: %v", err)
		}
	}
	for _, header := range []string{"fairyc_private_linux_https_unknown_v1(", "fairyc_private_linux_https_open_v1( fairyc_private_windows_https_open_v1("} {
		if _, err := LinuxSource(header); err == nil {
			t.Fatalf("invalid symbols admitted: %s", header)
		}
	}
}

func TestLinuxHTTPSParentBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; real child/TLS execution stays batched")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	parent := linuxHTTPSParent
	for _, header := range []string{"fcntl.h", "limits.h", "poll.h", "signal.h", "stdlib.h", "sys/prctl.h", "sys/wait.h", "unistd.h"} {
		parent = strings.ReplaceAll(parent, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile,
				linuxHTTPSParentTypes+linuxHTTPSProtocol+linuxHTTPSReader+linuxHTTPSParentStubs+parent+linuxHTTPSParentCases)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxHTTPSProbe(t, artifact.Bytes, "https-parent.exe")
		})
	}
}

const linuxHTTPSParentTypes = `
#include <assert.h>
#include <errno.h>
#include <limits.h>
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <setjmp.h>
#ifndef EPROTO
#define EPROTO 71
#endif
#ifndef EOVERFLOW
#define EOVERFLOW 75
#endif
#ifndef ECHILD
#define ECHILD 10
#endif
#ifndef ENOTSUP
#define ENOTSUP 95
#endif
typedef int pid_t;typedef unsigned int id_t;typedef int64_t ssize_t;
typedef struct {int si_pid;} siginfo_t;
struct sigaction {void (*sa_handler)(int);int sa_flags;};
struct pollfd {int fd;short events,revents;};
#define SIG_DFL ((void (*)(int))0)
#define SIG_IGN ((void (*)(int))1)
#define SIGCHLD 17
#define SIGKILL 9
#define SA_NOCLDWAIT 2
#define P_PID 1
#define WEXITED 4
#define WNOWAIT 0x1000000
#define WNOHANG 1
#define WIFEXITED(s) (((s)&127)==0)
#define WEXITSTATUS(s) (((s)>>8)&255)
#define O_CLOEXEC 0x80000
#define O_NONBLOCK 0x800
#define F_GETFL 3
#define F_SETFL 4
#define POLLIN 1
#define POLLERR 8
#define POLLHUP 16
#define POLLNVAL 32
#define PR_SET_PDEATHSIG 1
`

const linuxHTTPSParentStubs = `
static size_t now_ms,clock_calls,clock_fault_at,clock_jump_at;
static int mode,live_memory,live_child,fds[16],close_calls,kill_calls,wait_calls,fork_calls;
static int poll_calls,read_calls,waitid_calls,write_calls,exited,child_mode;
static uint8_t pipe_bytes[1024],written_bytes[1024];static size_t pipe_length,pipe_position,read_step,written_count;
static jmp_buf child_exit;
static uint32_t fairyc_private_linux_monotonic_ms_v1(size_t *out){
    ++clock_calls;if(clock_calls==clock_fault_at){*out=0;return EIO;}
    if(clock_calls==clock_jump_at){now_ms=0;}*out=now_ms;return 0;
}
static void *probe_calloc(size_t count,size_t size){if(mode==1){return NULL;}void *p=calloc(count,size);assert(p!=NULL);++live_memory;return p;}
static void probe_free(void *p){assert(p!=NULL && live_memory==1);--live_memory;free(p);}
static int probe_sigaction(int signal,const struct sigaction *set,struct sigaction *out){
    assert(signal==SIGCHLD && set==NULL);if(mode==2){errno=EIO;return -1;}out->sa_handler=mode==3?SIG_IGN:SIG_DFL;out->sa_flags=mode==4?SA_NOCLDWAIT:0;return 0;
}
static int probe_pipe2(int *out,int flags){assert(flags==O_CLOEXEC);if(mode==5){errno=EMFILE;return -1;}out[0]=7;out[1]=8;fds[7]=fds[8]=1;return 0;}
static int probe_fcntl(int fd,int command,...){
    assert(fd==7 && fds[7]);if(mode==6 && command==F_GETFL || mode==7 && command==F_SETFL){errno=EIO;return -1;}
    if(command==F_GETFL){return 0;}assert(command==F_SETFL);va_list ap;va_start(ap,command);assert(va_arg(ap,int)==O_NONBLOCK);va_end(ap);return 0;
}
static int probe_close(int fd){assert(fd>=0 && fd<16);++close_calls;if(!fds[fd]){errno=EBADF;return -1;}fds[fd]=0;if(mode==9 && fd==8 || mode==10 && fd==7){errno=EINTR;return -1;}return 0;}
static pid_t probe_getpid(void){return 1000;}
static pid_t probe_getppid(void){return mode==31?99:1000;}
static pid_t probe_fork(void){++fork_calls;if(mode==8){errno=EAGAIN;return -1;}if(mode==11){now_ms+=30000;}if(child_mode){return 0;}assert(!live_child);live_child=1;return 2000;}
static int probe_poll(struct pollfd *p,size_t count,int timeout){
    assert(count==1 && p->fd==7 && fds[7] && p->events==POLLIN && timeout>0 && timeout<=30000);++poll_calls;
    if(mode==12){now_ms+=(size_t)timeout;return 0;}if(mode==13 && poll_calls==1){errno=EINTR;return -1;}
    if(mode==38){now_ms+=1000;errno=EINTR;return -1;}
    if(mode==14){errno=EIO;return -1;}if(mode==15){p->revents=POLLNVAL;return 1;}
    if(mode==16){p->revents=0;return 1;}if(mode==17){now_ms+=(size_t)timeout;}
    p->revents=pipe_position<pipe_length?POLLIN:POLLHUP;return 1;
}
static ssize_t probe_read(int fd,void *data,size_t capacity){
    assert(fd==7 && fds[7] && capacity>0);++read_calls;
    if(mode==18 && read_calls==1){errno=EAGAIN;return -1;}if(mode==19 && read_calls==1){errno=EINTR;return -1;}
    if(mode==20){errno=EIO;return -1;}
    if(mode==29){return (ssize_t)(capacity+1);}if(mode==39){now_ms+=1000;errno=EAGAIN;return -1;}
    size_t count=pipe_length-pipe_position;if(count>capacity){count=capacity;}if(count>read_step){count=read_step;}
    memcpy(data,pipe_bytes+pipe_position,count);pipe_position+=count;if(count==0){exited=1;}if(mode==21){now_ms+=30000;}return (ssize_t)count;
}
static ssize_t probe_write(int fd,const void *data,size_t size){
    assert((fd==3 || fd==8) && data!=NULL && size>0);++write_calls;if(mode==35 && write_calls==1){errno=EINTR;return -1;}
    if(mode==36){errno=EIO;return -1;}if(mode==37){return 0;}size_t count=size>2?2:size;
    assert(written_count+count<=sizeof written_bytes);memcpy(written_bytes+written_count,data,count);written_count+=count;return (ssize_t)count;
}
static int probe_waitid(int which,id_t pid,siginfo_t *info,int flags){
    assert(which==P_PID && pid==2000 && flags==(WEXITED|WNOWAIT|WNOHANG));++waitid_calls;
    if(mode==22){live_child=0;errno=ECHILD;return -1;}if(mode==23 && waitid_calls==1){errno=EINTR;return -1;}
    info->si_pid=exited?2000:0;return 0;
}
static int probe_kill(pid_t pid,int signal){assert(pid==2000 && live_child && signal==SIGKILL);++kill_calls;exited=1;if(mode==24){errno=EIO;return -1;}if(mode==25){errno=ESRCH;return -1;}return 0;}
static pid_t probe_waitpid(pid_t pid,int *status,int flags){
    assert(pid==2000 && live_child && exited);++wait_calls;assert(flags==(mode==24?WNOHANG:0));
    if(mode==26 && wait_calls==1){errno=EINTR;return -1;}live_child=0;*status=mode==27?256:0;
    if(mode==28){errno=EIO;return -1;}return pid;
}
static int probe_prctl(int option,int signal){assert(option==PR_SET_PDEATHSIG && signal==SIGKILL);if(mode==30){errno=EIO;return -1;}return 0;}
static int probe_dup2(int from,int to){assert(from==8 && to==3 && fds[8]);if(mode==32){errno=EIO;return -1;}fds[3]=1;return 3;}
static int probe_close_range(unsigned int first,unsigned int last,int flags){assert(first==4 && last==UINT_MAX && flags==0);if(mode==33){errno=EIO;return -1;}for(size_t i=4;i<16;++i){fds[i]=0;}return 0;}
static __declspec(noreturn) void probe_exit(int code){assert(child_mode);longjmp(child_exit,code==0?1:2);}
static uint32_t fy_linux_https_transfer(const char *address,size_t maximum,size_t timeout,fy_linux_https_writer writer,void *sink){
    assert(strcmp(address,"https://packages.invalid/a.zip")==0 && maximum==100 && timeout>0 && timeout<=30000);
    for(size_t i=0;i<16;++i){assert(fds[i]==(i==3));}
    fy_linux_https_response response;memset(&response,0,sizeof response);response.write=writer;response.sink=sink;response.maximum=maximum;
    assert(fy_linux_https_header("HTTP/2 200\r\n",1,12,&response)==12);assert(fy_linux_https_line(&response,"\r\n",2)==0);
    char data[2]={'a',0};assert(fy_linux_https_body(data,1,2,&response)==2);return fy_linux_https_send(&response,3,0,NULL,0);
}
#define calloc probe_calloc
#define free probe_free
#define sigaction(...) probe_sigaction(__VA_ARGS__)
#define pipe2 probe_pipe2
#define fcntl probe_fcntl
#define close probe_close
#define getpid probe_getpid
#define getppid probe_getppid
#define fork probe_fork
#define poll probe_poll
#define read probe_read
#define write probe_write
#define waitid probe_waitid
#define waitpid probe_waitpid
#define kill probe_kill
#define prctl probe_prctl
#define dup2 probe_dup2
#define close_range probe_close_range
#define _exit probe_exit
`

const linuxHTTPSParentCases = `
static const uint8_t address[]="https://packages.invalid/a.zip";
static size_t child_raw;
static void reset(int value){
    assert(live_memory==0 && live_child==0);for(size_t i=0;i<16;++i){assert(fds[i]==0);}
    mode=value;now_ms=100;clock_calls=0;clock_fault_at=clock_jump_at=SIZE_MAX;
    close_calls=kill_calls=wait_calls=fork_calls=poll_calls=read_calls=waitid_calls=write_calls=exited=child_mode=0;
    pipe_length=pipe_position=written_count=0;read_step=SIZE_MAX;
}
static void bytes(const void *data,size_t count){assert(pipe_length+count<=sizeof pipe_bytes);memcpy(pipe_bytes+pipe_length,data,count);pipe_length+=count;}
static void frame(uint32_t kind,uint32_t count,uint32_t code){uint32_t words[3]={kind,count,code};bytes(words,12);}
static void good(void){
    uint8_t metadata[144]={0};uint32_t status=200;size_t absent=SIZE_MAX;memcpy(metadata,&status,4);memcpy(metadata+8,&absent,8);frame(1,144,0);bytes(metadata,144);
    const uint8_t data[3]={'a',0,255};frame(2,3,0);bytes(data,3);frame(3,0,0);
}
static uint32_t open_request(size_t *raw){return fairyc_private_linux_https_open_v1(address,sizeof address,100,30000,raw);}
static void clean(size_t raw,uint32_t expected){assert(fairyc_private_linux_https_close_v1(raw)==expected);assert(live_memory==0 && live_child==0);for(size_t i=0;i<16;++i){assert(fds[i]==0);}}
static void read_request(size_t raw,uint32_t expected){uint8_t data[8]={0};size_t copied=99;assert(fairyc_private_linux_https_read_v1(raw,data,sizeof data,&copied)==expected);if(expected==0){assert(copied==3 && data[0]=='a' && data[1]==0 && data[2]==255);}else{assert(copied==0);}}
int main(void){
    reset(0);size_t raw=99;assert(fairyc_private_linux_https_open_v1(NULL,0,100,1,&raw)==EINVAL && raw==0 && clock_calls==0);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address,100,1,NULL)==EINVAL);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address,0,1,&raw)==EINVAL);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address,16777217,1,&raw)==EINVAL);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address,100,0,&raw)==EINVAL);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address,100,30001,&raw)==EINVAL);
    assert(fairyc_private_linux_https_open_v1(address,sizeof address-1,100,1,&raw)==EINVAL);
    clock_fault_at=1;assert(open_request(&raw)==EIO && raw==0);clock_fault_at=SIZE_MAX;now_ms=SIZE_MAX-10;assert(open_request(&raw)==EOVERFLOW && raw==0);
    for(int scenario=1;scenario<=11;++scenario){
        reset(scenario);uint32_t expected=scenario==1?ENOMEM:scenario==3 || scenario==4?ENOTSUP:scenario==5?EMFILE:scenario==8?EAGAIN:scenario==9?EINTR:scenario==10?0:scenario==11?ETIMEDOUT:EIO;
        assert(open_request(&raw)==expected);if(raw!=0){clean(raw,scenario==10?EINTR:0);}else{assert(live_memory==0 && live_child==0);}
    }
    for(int scenario=0;scenario<=29;++scenario){
        if(scenario>0 && scenario<12){continue;}reset(scenario);good();assert(open_request(&raw)==0);
        if(scenario==0 || scenario==13 || scenario==18 || scenario==19 || scenario>=22){read_step=1;}
        uint32_t expected=scenario==12 || scenario==17 || scenario==21?ETIMEDOUT:scenario==14 || scenario==16 || scenario==20?EIO:scenario==15?EBADF:scenario==29?EPROTO:0;
        if(scenario!=24 && scenario!=25){read_request(raw,expected);}
        clean(raw,scenario==22?ECHILD:scenario==24 || scenario==27 || scenario==28?EIO:0);
        if(scenario==22){assert(kill_calls==0);}if(scenario==12){assert(poll_calls==1);}if(scenario==17){assert(read_calls==0);}
    }
    reset(0);good();assert(open_request(&raw)==0);uint8_t encoding[128];uint32_t status=99;size_t count=99,declared=99;
    assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&count,&declared)==0 && status==200 && count==0 && declared==SIZE_MAX);
    size_t at=pipe_position;assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&count,&declared)==0 && pipe_position==at);
    assert(fairyc_private_linux_https_read_v1(raw,encoding,0,&count)==EINVAL && count==0);
    assert(fairyc_private_linux_https_read_v1(raw,encoding,65537,&count)==EINVAL);
    assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,129,&count,&declared)==EINVAL);
    clock_fault_at=clock_calls+1;read_request(raw,EIO);clock_fault_at=SIZE_MAX;clock_jump_at=clock_calls+1;read_request(raw,EIO);clean(raw,0);
    reset(0);good();assert(open_request(&raw)==0);assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&count,&declared)==0);
    fy_linux_https_request *request=(fy_linux_https_request *)(uintptr_t)raw;now_ms=request->deadline-1;
    assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&count,&declared)==0);now_ms=request->deadline;
    assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&count,&declared)==ETIMEDOUT && status==0 && count==0);clean(raw,0);
    reset(14);good();assert(open_request(&raw)==0);read_request(raw,EIO);mode=10;clean(raw,EINTR);
    reset(0);good();assert(open_request(&raw)==0);read_request(raw,0);now_ms=30100;clean(raw,ETIMEDOUT);
    reset(0);good();assert(open_request(&raw)==0);read_request(raw,0);clock_fault_at=clock_calls+1;clean(raw,EIO);
    for(int scenario=38;scenario<=39;++scenario){reset(scenario);good();assert(open_request(&raw)==0);read_request(raw,ETIMEDOUT);assert((scenario==38?poll_calls:read_calls)==30);clean(raw,0);}
    reset(0);clock_fault_at=2;assert(open_request(&raw)==EIO && raw!=0 && fork_calls==0);clean(raw,0);
    reset(0);clock_fault_at=3;assert(open_request(&raw)==EIO && raw!=0 && fork_calls==0);clean(raw,0);
    reset(0);clock_fault_at=4;assert(open_request(&raw)==EIO && raw!=0 && fork_calls==1);clean(raw,0);
    /* Execute the actual child branch through a non-returning _exit substitute.
     * The parent test above independently exercises its own namespace copy. */
    for(int scenario=0;scenario<=33;++scenario){
        if(scenario>0 && scenario<30){continue;}reset(scenario);child_mode=1;
        for(size_t i=0;i<7;++i){fds[i]=1;}int outcome=setjmp(child_exit);if(outcome==0){(void)open_request(&child_raw);assert(0);}
        assert(outcome==(scenario==0?1:2) && written_count>=12);
        if(scenario!=0){uint32_t words[3];memcpy(words,written_bytes,12);assert(words[0]==3 && words[1]==0 && words[2]==(uint32_t)(scenario==31?ECANCELED:EIO));}
        assert(live_memory==1 && live_child==0);fy_linux_https_request *p=(fy_linux_https_request *)(uintptr_t)child_raw;
        /* longjmp models process death; release only the independent fixture. */
        probe_free(p);memset(fds,0,sizeof fds);
    }
    reset(35);int descriptor=3;uint8_t data[5]={0};assert(fy_linux_https_pipe_output(&descriptor,data,5)==0 && write_calls==4);
    reset(36);assert(fy_linux_https_pipe_output(&descriptor,data,5)==EIO);reset(37);assert(fy_linux_https_pipe_output(&descriptor,data,5)==EIO);
    assert(fairyc_private_linux_https_close_v1(0)==EINVAL);return 0;
}
`
