package candidatehost

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxGNUAdapterBoundaries(t *testing.T) {
	selected, err := LinuxSource("fairyc_private_linux_gnu_run_v1(")
	if err != nil || selected != linuxRootedFile+"\n"+linuxProcess+"\n"+linuxCapture+"\n"+linuxGNU {
		t.Fatalf("GNU dependency order: %v", err)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; Linux execution is covered separately")
	}
	source := strings.ReplaceAll(linuxGNU, "#include <dirent.h>\n", "")
	source = strings.ReplaceAll(source, "#include <sys/random.h>\n", "")
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxGNUPOSIXStub+source+linuxGNUBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "gnu.exe")
		})
	}
}

func TestLinuxGNUAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Linux amd64")
	}
	source, err := LinuxSource("fairyc_private_linux_gnu_run_v1(")
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.ResolveNativeToolchain(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: source + linuxGNUNativeProbe, RuntimeSource: "typedef int fy_gnu_probe_runtime;\n"})
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{WorkingDirectory: t.TempDir(), RejectOutput: true, Timeout: 15 * time.Second})
				if err != nil || outcome.ExitCode != 0 {
					t.Fatalf("GNU adapter: %+v %v", outcome, err)
				}
			})
		}
	}
}

const linuxGNUPOSIXStub = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <limits.h>
typedef intptr_t ssize_t;
typedef int pid_t;
typedef unsigned id_t;
typedef struct {int active;} posix_spawn_file_actions_t;
typedef struct {pid_t si_pid;} siginfo_t;
struct stat {unsigned st_mode;int64_t st_size;struct {long tv_sec,tv_nsec;} st_mtim;};
struct dirent {char d_name[256];};
typedef struct {int at;} DIR;
struct pollfd {int fd;short events,revents;};
#define O_RDONLY 0
#define O_WRONLY 1
#define O_CREAT 2
#define O_EXCL 4
#define O_NOFOLLOW 8
#define O_CLOEXEC 16
#define O_DIRECTORY 32
#define O_NONBLOCK 64
#define AT_SYMLINK_NOFOLLOW 256
#define S_ISREG(m) (((m)&0170000)==0100000)
#define S_ISDIR(m) (((m)&0170000)==0040000)
#define ESTALE 116
#define EOVERFLOW 132
#define P_PID 1
#define WEXITED 4
#define WNOWAIT 8
#define WNOHANG 1
#define POLLIN 1
#define POLLNVAL 32
#define SIGKILL 9
#define WIFEXITED(s) (((s)&127)==0)
#define WEXITSTATUS(s) (((s)>>8)&255)
#define WIFSIGNALED(s) (((s)&127)!=0)
#define WTERMSIG(s) ((s)&127)
static int mode,closes,creates,renames,unlinks,kills,gnu_probe_actions,reads,stats,nextfd;
static uint64_t gnu_probe_now;
static char *parent_env[]={"PATH=/tools","TMPDIR=/old","LC_ALL=old","LANG=old","KEEP=present",NULL};
#undef environ
#define environ gnu_probe_environment
static char **gnu_probe_environment=parent_env;
static int fy_linux_root_fd(uint32_t handle){return handle?((int)handle-1):-1;}
static int fy_linux_root_path(const uint8_t *p,size_t n,int leaf){return p && n>=2 && n<=4096 && p[n-1]==0 && memchr(p,0,n-1)==NULL && (!leaf || !memchr(p,'/',n-1));}
static uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *p,size_t n,uint32_t *out){assert(p && n);*out=4;return 0;}
static uint32_t fairyc_private_linux_root_file_close_v1(uint32_t fd){if(fd){++closes;}return 0;}
static int close(int fd){assert(fd>=0);++closes;return 0;}
static int open(const char *p,int flags){assert(p && flags);return 3;}
static int openat(int fd,const char *p,int flags,...){assert(fd>=0 && p);if(flags&O_CREAT){++creates;if(mode==4){errno=EEXIST;return -1;}return 7;}return (flags&O_DIRECTORY)?4:6;}
static int fstat(int fd,struct stat *info){memset(info,0,sizeof *info);++stats;info->st_mode=fd==3?(0040000|(mode==1?0755:0700)):0100000;info->st_size=mode==7?67108865:mode==8 && stats>1?9:8;return 0;}
static int fstatat(int fd,const char *p,struct stat *info,int flags){assert(fd>=0 && p && flags);memset(info,0,sizeof *info);info->st_mode=mode==3 || mode==15?0120000:0100000;return 0;}
static DIR gnu_probe_directory;
static DIR *fdopendir(int fd){assert(fd==4);gnu_probe_directory.at=0;return &gnu_probe_directory;}
static struct dirent *readdir(DIR *p){static struct dirent entry;if(p->at++){return NULL;}memcpy(entry.d_name,mode==2?"unexpected":"packet.bin",11);return &entry;}
static int closedir(DIR *p){assert(p==&gnu_probe_directory);return 0;}
static ssize_t write(int fd,const void *p,size_t n){assert(fd>=0 && p && n);if(mode==6 || mode==16){errno=EIO;return -1;}return (ssize_t)(mode==5 && n>1?1:n);}
static ssize_t read(int fd,void *p,size_t n){assert(fd==6);memset(p,'X',n);++reads;return (ssize_t)n;}
static int access(const char *p,int flags){assert(p && flags);return mode==10?-1:0;}
#define X_OK 1
static char *realpath(const char *p,char *ignored){assert(ignored==NULL);size_t n=strlen(p)+1;char *out=malloc(n);memcpy(out,p,n);return out;}
static int stat(const char *p,struct stat *info){assert(p);memset(info,0,sizeof *info);info->st_mode=0100000;return 0;}
static int mkdirat(int fd,const char *p,int permissions){assert(fd>=0 && p && permissions==0755);return 0;}
static ssize_t getrandom(void *out,size_t n,unsigned flags){assert(flags==0);if(mode==20){errno=EIO;return -1;}memset(out,1,n);return (ssize_t)n;}
static int fchmod(int fd,int permissions){assert(fd==7 && permissions==0700);return 0;}
static int fsync(int fd){assert(fd==7);if(mode==17){errno=EIO;return -1;}return 0;}
static int renameat(int a,const char *from,int b,const char *to){assert(a==b && from && to);if(mode==18){errno=EACCES;return -1;}++renames;return 0;}
static int unlinkat(int fd,const char *name,int flags){assert(fd>=0 && name && flags==0);++unlinks;return 0;}
static uint32_t fy_linux_capture_close(int *fd){if(*fd>=0){(void)close(*fd);*fd=-1;}return 0;}
static uint32_t fy_linux_capture_pipe(int fds[2],int minimum){assert(minimum==3);fds[0]=nextfd++;fds[1]=nextfd++;return 0;}
static uint32_t fy_linux_process_now(uint64_t *p){gnu_probe_now+=10000000;*p=gnu_probe_now;return 0;}
static uint32_t fy_linux_capture_read(int *fd,uint8_t *out,size_t cap,size_t *length,int *exceeded){assert(cap==65536);(void)exceeded;if(*fd>=0){if(mode==14){*length=65536;}else{memcpy(out,"version\n",8);*length=8;}(void)fy_linux_capture_close(fd);}return 0;}
static int posix_spawn_file_actions_init(posix_spawn_file_actions_t *p){p->active=1;++gnu_probe_actions;return 0;}
static int posix_spawn_file_actions_destroy(posix_spawn_file_actions_t *p){assert(p->active);--gnu_probe_actions;p->active=0;return 0;}
static int posix_spawn_file_actions_addchdir_np(posix_spawn_file_actions_t *p,const char *cwd){assert(p->active && strcmp(cwd,"/stage")==0);return 0;}
static int posix_spawn_file_actions_adddup2(posix_spawn_file_actions_t *p,int from,int to){assert(p->active && from>=3 && to>=0 && to<=2);return 0;}
static int posix_spawn_file_actions_addclosefrom_np(posix_spawn_file_actions_t *p,int from){assert(p->active && from==3);return 0;}
static int posix_spawn(pid_t *pid,const char *file,const posix_spawn_file_actions_t *p,const void *attr,char *const argv[],char *const env[]){assert(p->active && attr==NULL && strcmp(file,"/tools/gcc")==0 && strcmp(argv[1],"--version")==0 && argv[2]==NULL);int tmp=0,lc=0,lang=0,keep=0;for(size_t i=0;env[i];++i){tmp+=strcmp(env[i],"TMPDIR=/stage")==0;lc+=strcmp(env[i],"LC_ALL=C")==0;lang+=strcmp(env[i],"LANG=C")==0;keep+=strcmp(env[i],"KEEP=present")==0;}assert(tmp==1 && lc==1 && lang==1 && keep==1 && environ==parent_env);if(mode==11){return EACCES;}*pid=81;return 0;}
static int waitid(int what,id_t pid,siginfo_t *out,int flags){assert(what==P_PID && pid==81 && flags==(WEXITED|WNOWAIT|WNOHANG));out->si_pid=mode==13?0:81;return 0;}
static int waitpid(pid_t pid,int *status,int flags){assert(pid==81 && flags==0);*status=42<<8;return pid;}
static int kill(pid_t pid,int signal){assert(pid==81 && signal==SIGKILL);++kills;return 0;}
static int poll(struct pollfd *fds,int count,int timeout){assert(fds && count==2 && timeout==10);return 0;}
static char *probe_getenv(const char *name){assert(strcmp(name,"PATH")==0);return "/tools";}
#define getenv probe_getenv
static void reset(int value){mode=value;closes=creates=renames=unlinks=kills=gnu_probe_actions=reads=stats=0;gnu_probe_now=0;nextfd=10;}
`

const linuxGNUBoundaryProbe = `
int main(void){
    const uint8_t stage[]="/stage",leaf[]="part.o",root[]="/root",destination[]="build/program",name[]="gcc";uint32_t fd=0,kind=0,value=0;uint8_t out[65536],err[65536];size_t n=0,m=0;
    reset(0);assert(fairyc_private_linux_gnu_stage_v1(stage,sizeof stage,&fd)==0 && fd==4);
    reset(1);assert(fairyc_private_linux_gnu_stage_v1(stage,sizeof stage,&fd)!=0 && fd==0 && closes==1);
    reset(2);assert(fairyc_private_linux_gnu_stage_v1(stage,sizeof stage,&fd)!=0 && fd==0);
    reset(3);assert(fairyc_private_linux_gnu_stage_v1(stage,sizeof stage,&fd)!=0 && fd==0);
    reset(4);assert(fairyc_private_linux_gnu_write_v1(4,leaf,sizeof leaf,(const uint8_t *)"data",4)==EEXIST);
    reset(5);assert(fairyc_private_linux_gnu_write_v1(4,leaf,sizeof leaf,(const uint8_t *)"data",4)==0 && closes==1);
    reset(6);assert(fairyc_private_linux_gnu_write_v1(4,leaf,sizeof leaf,(const uint8_t *)"data",4)==EIO && closes==1);
    reset(0);assert(fairyc_private_linux_gnu_read_v1(4,leaf,sizeof leaf,NULL,0,&n)==0 && n==8);assert(fairyc_private_linux_gnu_read_v1(4,leaf,sizeof leaf,out,8,&n)==0 && n==8 && reads==1);
    reset(7);assert(fairyc_private_linux_gnu_read_v1(4,leaf,sizeof leaf,NULL,0,&n)!=0 && n==0);
    reset(8);assert(fairyc_private_linux_gnu_read_v1(4,leaf,sizeof leaf,out,8,&n)==ESTALE && n==0);
    reset(0);assert(fairyc_private_linux_gnu_find_v1(name,sizeof name,out,sizeof out,&n)==0 && n==10 && strcmp((const char *)out,"/tools/gcc")==0);
    reset(10);assert(fairyc_private_linux_gnu_find_v1(name,sizeof name,out,sizeof out,&n)==ENOENT);
    const uint8_t args[]="/tools/gcc\0--version";
    reset(0);assert(fairyc_private_linux_gnu_run_v1(args,sizeof args,2,stage,sizeof stage,1000,out,&n,err,&m,&kind,&value)==0 && kind==1 && value==42 && gnu_probe_actions==0 && kills==0);
    reset(11);assert(fairyc_private_linux_gnu_run_v1(args,sizeof args,2,stage,sizeof stage,1000,out,&n,err,&m,&kind,&value)==EACCES && gnu_probe_actions==0 && kills==0);
    reset(13);assert(fairyc_private_linux_gnu_run_v1(args,sizeof args,2,stage,sizeof stage,1,out,&n,err,&m,&kind,&value)==0 && kind==3 && kills==1);
    reset(14);assert(fairyc_private_linux_gnu_run_v1(args,sizeof args,2,stage,sizeof stage,1000,out,&n,err,&m,&kind,&value)==0 && kind==4 && kills==1);
    reset(0);assert(fairyc_private_linux_gnu_publish_v1(root,sizeof root,destination,sizeof destination,(const uint8_t *)"image",5)==0 && renames==1 && unlinks==1);
    for(int i=15;i<=18;++i){reset(i);assert(fairyc_private_linux_gnu_publish_v1(root,sizeof root,destination,sizeof destination,(const uint8_t *)"image",5)!=0 && renames==0);assert(unlinks==(i==15?0:1));}
    reset(20);assert(fairyc_private_linux_gnu_publish_v1(root,sizeof root,destination,sizeof destination,(const uint8_t *)"image",5)!=0 && creates==0 && unlinks==0);
    const uint8_t escape[]="../escape";reset(0);assert(fairyc_private_linux_gnu_publish_v1(root,sizeof root,escape,sizeof escape,(const uint8_t *)"image",5)==EINVAL && renames==0);
    return 0;
}
`

const linuxGNUNativeProbe = `
#include <assert.h>
int main(void){
    char root[4096];assert(getcwd(root,sizeof root)!=NULL);size_t root_length=strlen(root);assert(root_length<4000);char path[4096];memcpy(path,root,root_length);memcpy(path+root_length,"/stage",7);assert(mkdir(path,0700)==0);uint32_t stage=0;
    assert(fairyc_private_linux_gnu_stage_v1((uint8_t *)path,strlen(path)+1,&stage)==0);
    assert(fairyc_private_linux_gnu_write_v1(stage,(const uint8_t *)"part.o",7,(const uint8_t *)"data",4)==0);
    assert(fairyc_private_linux_gnu_write_v1(stage,(const uint8_t *)"part.o",7,(const uint8_t *)"new",3)==EEXIST);
    uint8_t data[4];size_t n=0;assert(fairyc_private_linux_gnu_read_v1(stage,(const uint8_t *)"part.o",7,data,4,&n)==0 && n==4 && memcmp(data,"data",4)==0);
    assert(fairyc_private_linux_gnu_publish_v1((uint8_t *)root,strlen(root)+1,(const uint8_t *)"build/program",14,data,4)==0);
    struct stat info;assert(stat("build/program",&info)==0 && (info.st_mode&0777)==0700);
    assert(symlink("program","build/link")==0);assert(fairyc_private_linux_gnu_publish_v1((uint8_t *)root,strlen(root)+1,(const uint8_t *)"build/link",11,data,4)!=0);
    uint8_t tool[4096],out[65536],error[65536];size_t length=0,err_length=0;uint32_t kind=0,status=0;
    assert(fairyc_private_linux_gnu_find_v1((const uint8_t *)"getconf",8,tool,sizeof tool,&length)==0);
    uint8_t args[8192];memcpy(args,tool,length+1);memcpy(args+length+1,"GNU_LIBC_VERSION",17); /* includes its terminator */
    assert(fairyc_private_linux_gnu_run_v1(args,length+18,2,(uint8_t *)path,strlen(path)+1,5000,out,&n,error,&err_length,&kind,&status)==0 && kind==1 && status==0 && n>=10);
    assert(fairyc_private_linux_root_file_close_v1(stage)==0);return 0;
}
`
