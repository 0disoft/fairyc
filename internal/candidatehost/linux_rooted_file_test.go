package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxRootedFileSelection(t *testing.T) {
	for _, name := range []string{"directory_open", "root_directory_open", "root_file_open", "root_file_info", "root_file_read", "root_file_close"} {
		source, err := LinuxSource("fairyc_private_linux_" + name + "_v1(")
		if err != nil || source != linuxRootedFile {
			t.Fatalf("%s: %v", name, err)
		}
	}
	source, err := LinuxSource("fairyc_private_linux_host_path_v1( fairyc_private_linux_root_file_read_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxPaths {
		t.Fatalf("feature macro order: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_root_file_unknown_v1("); err == nil {
		t.Fatal("unknown operation admitted")
	}
}

func TestLinuxRootedFileAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile
	for _, header := range []string{"fcntl.h", "sys/stat.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxRootedPOSIXStub+source+linuxRootedBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "rooted.exe")
		})
	}
}

func TestLinuxRootedFileAdapterNative(t *testing.T) {
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
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+linuxRootedNativeProbe, "typedef int fy_rooted_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "rooted")
			})
		}
	}
}

const linuxRootedPOSIXStub = `
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
#define EIO 5
#define EINTR 4
#define EINVAL 22
#define AT_FDCWD -100
#define O_RDONLY 0
#define O_DIRECTORY 65536
#define O_NOFOLLOW 131072
#define O_CLOEXEC 524288
#define O_NONBLOCK 2048
#define O_NOCTTY 256
#define S_ISDIR(x) (((x)&61440)==16384)
#define S_ISREG(x) (((x)&61440)==32768)
typedef intptr_t ssize_t;
struct stat {uint64_t st_dev,st_ino;int64_t st_size;uint32_t st_mode;struct {int64_t tv_sec,tv_nsec;} st_mtim;};
static int scenario,opened,closed,reads,stats,last_flags,last_parent;
static int root_openat(int parent,const char *path,int flags,int mode) {
    assert(path && mode==0600);++opened;last_flags=flags;last_parent=parent;
    if(scenario==1 && opened==1) {errno=EINTR;return -1;}
    if(scenario==2) {errno=2;return -1;}
    return 0;
}
static int root_fstat(int fd,struct stat *info) {
    assert(fd==0);++stats;
    if(scenario==3) {errno=5;return -1;}
    if(scenario==8 && stats==1) {errno=EINTR;return -1;}
    memset(info,0,sizeof *info);info->st_dev=71;info->st_ino=123;info->st_size=3;
    info->st_mode=scenario==4 ? 4096 : last_flags&O_DIRECTORY ? 16384 : 32768;
    info->st_mtim.tv_sec=91;info->st_mtim.tv_nsec=scenario==9 ? 1000000000 : 42;
    if(scenario==10) {info->st_size=-1;}
    return 0;
}
static int root_close(int fd) {assert(fd==0);++closed;if(scenario==5) {errno=EINTR;return -1;}return 0;}
static ssize_t root_read(int fd,void *data,size_t size) {
    assert(fd==0 && size>0 && size<=65536);++reads;
    if(scenario==6 && reads==1) {errno=EINTR;return -1;}
    if(scenario==7) {errno=EIO;return -1;}
    if(scenario==11) {return (ssize_t)size+1;}
    if(size>3) {size=3;}memcpy(data,"a\0\xff",size);return (ssize_t)size;
}
#define openat root_openat
#define fstat root_fstat
#define close root_close
#define read root_read
`

const linuxRootedBoundaryProbe = `
static void reset(int value) {scenario=value;opened=closed=reads=stats=0;}
int main(void) {
    const uint8_t root[]="/work",leaf[]="file";uint32_t handle=99;size_t count=99;uint8_t bytes[4];
    reset(0);assert(fairyc_private_linux_directory_open_v1(root,sizeof root,&handle)==0 && handle==1);
    assert(last_parent==AT_FDCWD && (last_flags&(O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY))==(O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY));
    assert(fairyc_private_linux_root_file_close_v1(handle)==0 && closed==1);
    assert(fairyc_private_linux_root_file_close_v1(0)==0 && closed==1);
    assert(fairyc_private_linux_root_file_close_v1(UINT32_MAX)==EINVAL);
    const char *invalid[]={"",".","..","a/b","x\0y"};size_t sizes[]={1,2,3,4,4};
    for(size_t i=0;i<5;++i) {handle=99;assert(fairyc_private_linux_root_file_open_v1(1,(const uint8_t *)invalid[i],sizes[i],&handle)==EINVAL && handle==0 && opened==1);}
    assert(fairyc_private_linux_directory_open_v1(root,sizeof root-1,&handle)==EINVAL && handle==0);
    assert(fairyc_private_linux_root_file_open_v1(0,leaf,sizeof leaf,&handle)==EINVAL);
    reset(1);assert(fairyc_private_linux_root_file_open_v1(1,leaf,sizeof leaf,&handle)==0 && opened==2 && handle==1);
    assert(last_parent==0 && (last_flags&(O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC|O_NOCTTY))==(O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC|O_NOCTTY));
    reset(2);assert(fairyc_private_linux_root_file_open_v1(1,leaf,sizeof leaf,&handle)==2 && handle==0 && closed==0);
    reset(3);assert(fairyc_private_linux_root_file_open_v1(1,leaf,sizeof leaf,&handle)==5 && handle==0 && closed==1);
    reset(4);assert(fairyc_private_linux_root_file_open_v1(1,leaf,sizeof leaf,&handle)==EINVAL && handle==0 && closed==1);
    reset(5);assert(fairyc_private_linux_root_file_close_v1(1)==EINTR && closed==1);
    reset(6);assert(fairyc_private_linux_root_file_read_v1(1,bytes,4,&count)==0 && count==3 && reads==2 && memcmp(bytes,"a\0\xff",3)==0);
    reset(7);assert(fairyc_private_linux_root_file_read_v1(1,bytes,4,&count)==EIO && count==0);
    assert(fairyc_private_linux_root_file_read_v1(1,bytes,65537,&count)==EINVAL && reads==1);
    assert(fairyc_private_linux_root_file_read_v1(1,NULL,0,&count)==0 && count==0 && reads==1);
    struct fy_linux_root_metadata_v1 info;
    reset(8);assert(fairyc_private_linux_root_file_info_v1(1,&info)==0 && stats==2);
    assert(info.device==71 && info.identity==123 && info.size==3 && info.modified_seconds==91 && info.modified_nanos==42 && info.mode==32768);
    reset(9);assert(fairyc_private_linux_root_file_info_v1(1,&info)==EINVAL && info.identity==0);
    reset(10);assert(fairyc_private_linux_root_file_info_v1(1,&info)==EINVAL && info.identity==0);
    reset(11);assert(fairyc_private_linux_root_file_read_v1(1,bytes,4,&count)==EIO && count==0);
    reset(0);assert(fairyc_private_linux_root_directory_open_v1(1,leaf,sizeof leaf,&handle)==0 && handle==1 && last_parent==0);
    return 0;
}
`

const linuxRootedNativeProbe = `
#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
int main(void) {
    char temp[]="/tmp/fairyc-rooted-XXXXXX";assert(mkdtemp(temp));uint32_t root=0,file=0,child=0;
    assert(fairyc_private_linux_directory_open_v1((const uint8_t *)temp,strlen(temp)+1,&root)==0);
    int directory=fy_linux_root_fd(root);assert(fcntl(directory,F_GETFD)&FD_CLOEXEC);
    int fd=openat(directory,"input",O_CREAT|O_EXCL|O_WRONLY|O_CLOEXEC,0600);assert(fd>=0);
    assert(write(fd,"a\0\xff",3)==3 && close(fd)==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"input",6,&file)==0);
    struct fy_linux_root_metadata_v1 info;assert(fairyc_private_linux_root_file_info_v1(file,&info)==0 && info.size==3);
    uint8_t bytes[4];size_t copied=99;assert(fairyc_private_linux_root_file_read_v1(file,bytes,4,&copied)==0 && copied==3 && memcmp(bytes,"a\0\xff",3)==0);
    assert(fairyc_private_linux_root_file_read_v1(file,bytes,4,&copied)==0 && copied==0);
    assert(fairyc_private_linux_root_file_close_v1(file)==0);
    assert(symlinkat("input",directory,"link")==0 && mkfifoat(directory,"pipe",0600)==0 && mkdirat(directory,"child",0700)==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"link",5,&file)!=0 && file==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"pipe",5,&file)!=0 && file==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"child",6,&file)!=0 && file==0);
    assert(fairyc_private_linux_root_directory_open_v1(root,(const uint8_t *)"child",6,&child)==0);
    assert(fairyc_private_linux_root_file_close_v1(child)==0);
    char moved[256];snprintf(moved,sizeof moved,"%s-moved",temp);assert(rename(temp,moved)==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"input",6,&file)==0);
    assert(fairyc_private_linux_root_file_close_v1(file)==0);
    assert(unlinkat(directory,"input",0)==0 && unlinkat(directory,"link",0)==0 && unlinkat(directory,"pipe",0)==0 && unlinkat(directory,"child",AT_REMOVEDIR)==0);
    assert(fairyc_private_linux_root_file_close_v1(root)==0 && rmdir(moved)==0);return 0;
}
`
