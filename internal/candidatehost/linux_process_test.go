package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxProcessSelection(t *testing.T) {
	for _, name := range []string{"start", "wait", "discard", "wait_for"} {
		source, err := LinuxSource("fairyc_private_linux_process_" + name + "_v1(")
		if err != nil || source != linuxProcess {
			t.Fatalf("process selection %s: %v", name, err)
		}
	}
	if source, err := LinuxSource("fairyc_private_linux_host_path_v1( fairyc_private_linux_process_wait_v1("); err != nil || source != linuxProcess+"\n"+linuxPaths {
		t.Fatalf("combined GNU feature-macro order: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_process_unknown_v1("); err == nil {
		t.Fatal("unknown process adapter admitted")
	}
}

func TestLinuxProcessAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("POSIX fault substitutes use MSVC; Linux has a separate native probe")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxProcess
	for _, header := range []string{"spawn.h", "sys/wait.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxProcessPOSIXStub+source+linuxProcessBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "process.exe")
		})
	}
}

func TestLinuxProcessAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native GNU/Linux amd64, GCC and Clang")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxProcess+linuxProcessNativeProbe, "typedef int fy_process_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "process")
			})
		}
	}
}

const linuxProcessPOSIXStub = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
typedef int pid_t;
typedef struct { int active; } posix_spawn_file_actions_t;
typedef struct { int active,group,flags; } posix_spawnattr_t;
typedef unsigned int id_t;
typedef struct { pid_t si_pid; } siginfo_t;
#define P_PID 1
#define WEXITED 4
#define WNOWAIT 8
#define POSIX_SPAWN_SETPGROUP 2
#define CLOCK_MONOTONIC 1
#define TIMER_ABSTIME 1
#define WNOHANG 1
#define SIGKILL 9
#define WIFEXITED(s) (((s)&127)==0)
#define WEXITSTATUS(s) (((s)>>8)&255)
#define WIFSIGNALED(s) (((s)&127)!=0 && ((s)&127)!=127)
#define WTERMSIG(s) ((s)&127)
static int scenario, allocations, actions_live, attrs_live, spawn_calls, wait_calls, queries, kills, group_kills, phase;
static int clock_calls,sleeps;
static uint64_t fake_now;
static int probe_clock(int clock,struct timespec *value) {
    assert(clock==CLOCK_MONOTONIC); ++clock_calls;
    if(scenario==25 || scenario==26 && clock_calls==3) { errno=EIO; return -1; }
    if(scenario==24) { fake_now+=1000000; }
    value->tv_sec=(time_t)(fake_now/1000000000); value->tv_nsec=(long)(fake_now%1000000000);
    if(scenario==30 && clock_calls==3) { value->tv_sec=0; value->tv_nsec=0; }
    if(scenario==32) { value->tv_nsec=1000000000L; }
    if(scenario==33) { value->tv_sec=(time_t)INT64_MAX; }
    return 0;
}
static int probe_sleep(int clock,int flags,const struct timespec *until,struct timespec *remaining) {
    assert(clock==CLOCK_MONOTONIC && flags==TIMER_ABSTIME && remaining==NULL);
    uint64_t wake=(uint64_t)until->tv_sec*1000000000+(uint64_t)until->tv_nsec;
    assert(wake>fake_now && wake-fake_now<=10000000 && wake<=2020000000);
    ++sleeps;
    if(scenario==27) { errno=EIO; return EINVAL; }
    if(scenario==23 && sleeps==1) { fake_now+=3000000; return EINTR; }
    fake_now=wake; return 0;
}
static char **probe_environ;
static void *probe_calloc(size_t n,size_t size) {
    if(scenario==1) { return NULL; }
    void *p=calloc(n,size); if(p!=NULL) { ++allocations; } return p;
}
static void probe_free(void *p) { if(p!=NULL) { --allocations; } free(p); }
static int probe_init(posix_spawn_file_actions_t *p) {
    if(scenario==2) { return ENOMEM; } p->active=1; ++actions_live; return 0;
}
static int probe_chdir(posix_spawn_file_actions_t *p,const char *cwd) {
    assert(p->active && strcmp(cwd,"/tmp")==0); return scenario==3 ? ENOENT : 0;
}
static int probe_closefrom(posix_spawn_file_actions_t *p,int low) {
    assert(p->active && low==3); return scenario==4 ? EINVAL : 0;
}
static int probe_destroy(posix_spawn_file_actions_t *p) { assert(p->active); p->active=0; --actions_live; return 0; }
static int probe_attr_init(posix_spawnattr_t *p) {
    if(scenario==17) { return ENOMEM; }
    memset(p,0,sizeof *p); p->active=1; ++attrs_live; return 0;
}
static int probe_attr_group(posix_spawnattr_t *p,pid_t group) {
    assert(p->active && group==0); p->group=1; return scenario==18 ? EINVAL : 0;
}
static int probe_attr_flags(posix_spawnattr_t *p,short flags) {
    assert(p->active && p->group && flags==POSIX_SPAWN_SETPGROUP); p->flags=flags;
    return scenario==19 ? EINVAL : 0;
}
static int probe_attr_destroy(posix_spawnattr_t *p) { assert(p->active); p->active=0; --attrs_live; return 0; }
static int probe_spawn(pid_t *pid,const char *path,const posix_spawn_file_actions_t *p,
        const posix_spawnattr_t *attributes,char *const argv[],char *const env[]) {
    assert(p->active && attributes->active && attributes->group && attributes->flags==POSIX_SPAWN_SETPGROUP && env==probe_environ);
    assert(strcmp(path,"x")==0 && strcmp(argv[0],"x")==0 && strcmp(argv[1],"")==0 && strcmp(argv[2],"a b")==0 && argv[3]==NULL);
    ++spawn_calls; if(scenario==5) { return ENOENT; } if(scenario==6) { return EACCES; }
    *pid=4321; return 0;
}
static int probe_waitid(int type,id_t pid,siginfo_t *info,int flags) {
    assert(type==P_PID && pid==4321 && (flags & (WEXITED|WNOWAIT))==(WEXITED|WNOWAIT));
    ++queries;
    if(scenario==11 && queries==1 || scenario==15) { errno=EINVAL; return -1; }
    if(scenario==12) { errno=ECHILD; return -1; }
    if(scenario==20 && queries==1) { errno=EINTR; return -1; }
    if(scenario==24) { errno=EINTR; return -1; }
    if(scenario==28) { errno=ECHILD; return -1; }
    if(scenario==29) { errno=EIO; return -1; }
    if(scenario>=21 && scenario!=21 && !(scenario==34 && sleeps>0)) { info->si_pid=0; phase=1; return 0; }
    info->si_pid=4321; phase=1; return 0;
}
static pid_t probe_waitpid(pid_t pid,int *status,int flags) {
    assert(pid==4321 && phase==3); ++wait_calls;
    if(scenario==10 && wait_calls==1) { errno=EINTR; return -1; }
    if(scenario==12 || scenario==14 && flags==0) { errno=ECHILD; return -1; }
    if(scenario==15) { errno=EINVAL; return -1; }
    if(flags==WNOHANG && scenario!=13) { return 0; }
    *status=scenario==8 ? 127<<8 : scenario==9 ? 15 : 41<<8;
    return pid;
}
static int probe_kill(pid_t pid,int sig) {
    assert((pid==4321 || pid==-4321) && sig==SIGKILL);
    if(pid<0) { assert(phase==1); phase=2; ++group_kills; }
    else { assert(phase==2); phase=3; ++kills; }
    if(scenario==14) { errno=ESRCH; return -1; }
    if(scenario==16 || scenario==31) { errno=EPERM; return -1; } return 0;
}
#undef environ
#define environ probe_environ
#define calloc probe_calloc
#define free probe_free
#define posix_spawn_file_actions_init probe_init
#define posix_spawn_file_actions_addchdir_np probe_chdir
#define posix_spawn_file_actions_addclosefrom_np probe_closefrom
#define posix_spawn_file_actions_destroy probe_destroy
#define posix_spawnattr_init probe_attr_init
#define posix_spawnattr_setpgroup probe_attr_group
#define posix_spawnattr_setflags probe_attr_flags
#define posix_spawnattr_destroy probe_attr_destroy
#define posix_spawn probe_spawn
#define waitid probe_waitid
#define waitpid probe_waitpid
#define kill probe_kill
#define clock_gettime probe_clock
#define clock_nanosleep probe_sleep
`

const linuxProcessBoundaryProbe = `
static void deadlines(void) {
    uint32_t kind=99,value=99;
    scenario=21;queries=0;
    assert(fairyc_private_linux_process_wait_for_v1(4321,0,&kind,&value)==EINVAL && kind==0 && value==0);
    assert(fairyc_private_linux_process_wait_for_v1(4321,86400001,&kind,&value)==EINVAL);
    assert(fairyc_private_linux_process_wait_for_v1(1,25,&kind,&value)==EINVAL);
    assert(fairyc_private_linux_process_wait_for_v1(4321,25,NULL,&value)==EINVAL && queries==0);
    for(scenario=21;scenario<=34;++scenario) {
        fake_now=1995000000; clock_calls=0;sleeps=0;queries=0;wait_calls=0;kills=0;group_kills=0;phase=0;
        uint32_t code=fairyc_private_linux_process_wait_for_v1(4321,25,&kind,&value);
        if(scenario==21 || scenario==34) {
            assert(code==0 && kind==1 && value==41 && kills==1 && group_kills==1 && wait_calls==1);
            assert(sleeps==(scenario==21 ? 0 : 1));
        } else if(scenario==22 || scenario==23) {
            assert(code==0 && kind==3 && value==0 && kills==1 && group_kills==1 && wait_calls==1);
            assert(fake_now==2020000000 && sleeps==(scenario==22 ? 3 : 4));
        } else {
            uint32_t expected=scenario==24 ? EINTR : scenario==27 ? EINVAL : scenario==28 ? ECHILD :
                scenario==31 ? EPERM : scenario==32 || scenario==33 ? EOVERFLOW : EIO;
            assert(code==expected && kind==0 && value==0 && kills==0 && wait_calls==0);
            assert(group_kills==(scenario==31 ? 1 : 0));
            if(scenario==24) { assert(fake_now>=2020000000 && clock_calls<30 && sleeps==0); }
        }
    }
    scenario=21;fake_now=1995000000;
    assert(fairyc_private_linux_process_wait_for_v1(4321,86400000,&kind,&value)==0 && kind==1);
}
int main(void) {
    const uint8_t data[]="x\0\0a b",cwd[]="/tmp"; uint32_t pid=99,kind=99,value=99;
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,cwd,sizeof cwd,&pid)==0 && pid==4321 && allocations==0 && actions_live==0 && attrs_live==0);
    const uint32_t errors[]={0,ENOMEM,ENOMEM,ENOENT,EINVAL,ENOENT,EACCES};
    for(scenario=1;scenario<=6;++scenario) {
        assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,cwd,sizeof cwd,&pid)==errors[scenario]);
        assert(pid==0 && allocations==0 && actions_live==0 && attrs_live==0);
    }
    for(scenario=17;scenario<=19;++scenario) {
        assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,cwd,sizeof cwd,&pid)==(uint32_t)(scenario==17 ? ENOMEM : EINVAL));
        assert(pid==0 && allocations==0 && actions_live==0 && attrs_live==0);
    }
    scenario=0; int before=spawn_calls;
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,2,cwd,sizeof cwd,&pid)==EINVAL && pid==0);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data-1,3,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,0,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,(size_t)-1,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,(size_t)-1,3,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1((const uint8_t *)"",1,1,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,cwd,sizeof cwd-1,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,(const uint8_t *)"/t\0mp",6,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(NULL,7,3,cwd,sizeof cwd,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,NULL,5,&pid)==EINVAL);
    assert(fairyc_private_linux_process_start_v1(data,sizeof data,3,cwd,sizeof cwd,NULL)==EINVAL);
    assert(spawn_calls==before && allocations==0 && actions_live==0);
    for(scenario=7;scenario<=10;++scenario) {
        wait_calls=0; assert(fairyc_private_linux_process_wait_v1(4321,&kind,&value)==0);
        assert(kind==(scenario==9 ? 2u : 1u) && value==(scenario==8 ? 127u : scenario==9 ? 15u : 41u));
        assert(wait_calls==(scenario==10 ? 2 : 1));
    }
    assert(fairyc_private_linux_process_wait_v1(0,&kind,&value)==EINVAL && kind==0 && value==0);
    assert(fairyc_private_linux_process_wait_v1(1,&kind,&value)==EINVAL);
    scenario=11;wait_calls=0;queries=0;kills=0;group_kills=0;
    assert(fairyc_private_linux_process_wait_v1(4321,&kind,&value)==EINVAL);
    assert(fairyc_private_linux_process_discard_v1(4321)==0 && kills==1 && group_kills==1);
    scenario=12;kills=0;group_kills=0; assert(fairyc_private_linux_process_discard_v1(4321)==0 && kills==0 && group_kills==0);
    scenario=13; assert(fairyc_private_linux_process_discard_v1(4321)==0 && kills==1 && group_kills==1);
    scenario=14;kills=0;group_kills=0; assert(fairyc_private_linux_process_discard_v1(4321)==ECHILD && kills==1 && group_kills==1);
    scenario=15;kills=0; assert(fairyc_private_linux_process_discard_v1(4321)==EINVAL && kills==0);
    scenario=16; assert(fairyc_private_linux_process_discard_v1(4321)==EPERM && kills==0);
    scenario=20;queries=0;wait_calls=0;
    assert(fairyc_private_linux_process_wait_v1(4321,&kind,&value)==0 && queries==2);
    assert(fairyc_private_linux_process_discard_v1(1)==EINVAL);
    assert(fairyc_private_linux_process_discard_v1(0)==0);
    deadlines();
    return 0;
}
`

const linuxProcessNativeProbe = `
#include <assert.h>
#include <fcntl.h>
#include <stdio.h>
#include <sys/prctl.h>
int main(int argc,char **argv) {
    if(argc>1) {
        if(strcmp(argv[1],"127")==0) { return 127; }
        if(strcmp(argv[1],"signal")==0) { raise(SIGTERM); return 1; }
        if(strcmp(argv[1],"tree")==0 || strcmp(argv[1],"slowtree")==0) {
            assert(getpgrp()==getpid());
            pid_t descendant=fork(); assert(descendant>=0);
            if(descendant==0) { alarm(3); for(;;) { pause(); } }
            FILE *receipt=fopen(argv[2],"w"); assert(receipt!=NULL);
            assert(fprintf(receipt,"%d",(int)descendant)>0 && fclose(receipt)==0);
            if(strcmp(argv[1],"slowtree")==0) { alarm(3); for(;;) { pause(); } }
            return 17;
        }
        char current[4096]; assert(getcwd(current,sizeof current)!=NULL && strcmp(current,argv[2])==0);
        assert(argc==6 && strcmp(argv[3],"")==0 && strcmp(argv[4],"a b\"\\")==0);
        errno=0; assert(fcntl(atoi(argv[5]),F_GETFD)==-1 && errno==EBADF); return 41;
    }
    char image[4096],original[4096],directory[]="/tmp/fairyc-process-XXXXXX",packed[8192];
    ssize_t image_size=readlink("/proc/self/exe",image,sizeof image-1); assert(image_size>0); image[image_size]=0;
    assert(getcwd(original,sizeof original)!=NULL && mkdtemp(directory)!=NULL);
    int descriptor=open("/dev/null",O_RDONLY); assert(descriptor>=3);
    char fd[32]; assert(snprintf(fd,sizeof fd,"%d",descriptor)>0);
    const char *values[]={image,"41",directory,"","a b\"\\",fd}; size_t used=0;
    for(size_t i=0;i<6;++i) { size_t n=strlen(values[i])+1; assert(used+n<=sizeof packed); memcpy(packed+used,values[i],n); used+=n; }
    uint32_t pid=0,kind=0,value=0;
    assert(fairyc_private_linux_process_start_v1((uint8_t *)packed,used,6,(uint8_t *)directory,strlen(directory)+1,&pid)==0 && pid>0);
    assert(fairyc_private_linux_process_wait_v1(pid,&kind,&value)==0 && kind==1 && value==41);
    int status; errno=0; assert(waitpid((pid_t)pid,&status,WNOHANG)==-1 && errno==ECHILD);
    assert(close(descriptor)==0);
    const char *modes[]={"127","signal"};
    for(size_t i=0;i<2;++i) {
        used=strlen(image)+1; memcpy(packed,image,used); memcpy(packed+used,modes[i],strlen(modes[i])+1); used+=strlen(modes[i])+1;
        assert(fairyc_private_linux_process_start_v1((uint8_t *)packed,used,2,(uint8_t *)directory,strlen(directory)+1,&pid)==0);
        assert(fairyc_private_linux_process_wait_v1(pid,&kind,&value)==0 && kind==(i==0 ? 1u : 2u) && value==(i==0 ? 127u : (uint32_t)SIGTERM));
    }
    assert(fairyc_private_linux_process_start_v1((const uint8_t *)"./absent",9,1,(uint8_t *)directory,strlen(directory)+1,&pid)==ENOENT && pid==0);
    assert(fairyc_private_linux_process_start_v1((uint8_t *)image,strlen(image)+1,1,(const uint8_t *)"/fairyc-no-such-cwd",20,&pid)==ENOENT && pid==0);
    char denied[4096]; assert(snprintf(denied,sizeof denied,"%s/denied",directory)>0);
    int file=open(denied,O_WRONLY|O_CREAT|O_EXCL,0600); assert(file>=0 && close(file)==0);
    assert(fairyc_private_linux_process_start_v1((uint8_t *)denied,strlen(denied)+1,1,(uint8_t *)directory,strlen(directory)+1,&pid)==EACCES && pid==0);
    assert(unlink(denied)==0);
    assert(prctl(PR_SET_CHILD_SUBREAPER,1,0,0,0)==0);
    char receipt[4096]; assert(snprintf(receipt,sizeof receipt,"%s/child",directory)>0);
    used=0; const char *tree[]={image,"tree",receipt};
    for(size_t i=0;i<3;++i) { size_t n=strlen(tree[i])+1; assert(used+n<=sizeof packed); memcpy(packed+used,tree[i],n); used+=n; }
    assert(fairyc_private_linux_process_start_v1((uint8_t *)packed,used,3,(uint8_t *)directory,strlen(directory)+1,&pid)==0);
    assert(fairyc_private_linux_process_wait_v1(pid,&kind,&value)==0 && kind==1 && value==17);
    FILE *input=fopen(receipt,"r"); int descendant=0;
    assert(input!=NULL && fscanf(input,"%d",&descendant)==1 && descendant>1 && fclose(input)==0);
    assert(waitpid((pid_t)descendant,&status,0)==descendant && WIFSIGNALED(status) && WTERMSIG(status)==SIGKILL);
    assert(unlink(receipt)==0);
    tree[1]="slowtree";used=0;
    for(size_t i=0;i<3;++i) { size_t n=strlen(tree[i])+1; assert(used+n<=sizeof packed); memcpy(packed+used,tree[i],n); used+=n; }
    assert(fairyc_private_linux_process_start_v1((uint8_t *)packed,used,3,(uint8_t *)directory,strlen(directory)+1,&pid)==0);
    int descendant_ready=0;
    for(int attempt=0;attempt<100;++attempt) {
        input=fopen(receipt,"r");
        if(input!=NULL) {
            int ready=fscanf(input,"%d",&descendant); assert(fclose(input)==0);
            if(ready==1) { descendant_ready=1; break; }
        }
        struct timespec delay={0,10000000};
        int slept=clock_nanosleep(CLOCK_MONOTONIC,0,&delay,NULL); assert(slept==0 || slept==EINTR);
    }
    assert(descendant_ready && descendant>1);
    assert(fairyc_private_linux_process_wait_for_v1(pid,25,&kind,&value)==0 && kind==3 && value==0);
    assert(waitpid((pid_t)descendant,&status,0)==descendant && WIFSIGNALED(status) && WTERMSIG(status)==SIGKILL);
    assert(unlink(receipt)==0 && rmdir(directory)==0);
    char after[4096]; assert(getcwd(after,sizeof after)!=NULL && strcmp(original,after)==0);
    assert(waitpid(-1,&status,WNOHANG)==-1 && errno==ECHILD);
    return 0;
}
`
