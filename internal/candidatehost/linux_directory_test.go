package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxDirectorySelection(t *testing.T) {
	for _, name := range []string{"open", "next", "close"} {
		source, err := LinuxSource("fairyc_private_linux_directory_scan_" + name + "_v1(")
		if err != nil || source != linuxRootedFile+"\n"+linuxDirectory {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := LinuxSource("fairyc_private_linux_directory_scan_unknown_v1("); err == nil {
		t.Fatal("unknown scan admitted")
	}
}

func TestLinuxDirectoryAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile + "\n" + linuxDirectory
	for _, header := range []string{"fcntl.h", "sys/stat.h", "unistd.h", "dirent.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxRootedPOSIXStub+linuxDirectoryPOSIXStub+source+linuxDirectoryBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "directory.exe")
		})
	}
}

func TestLinuxDirectoryAdapterNative(t *testing.T) {
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
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+"\n"+linuxDirectory+linuxDirectoryNativeProbe, "typedef int fy_directory_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "directory")
			})
		}
	}
}

const linuxDirectoryPOSIXStub = `
#define AT_SYMLINK_NOFOLLOW 256
typedef struct {int used,index;} DIR;
struct dirent {uint64_t d_ino;uint16_t d_reclen;char d_name[512];};
static DIR streams[2];static struct dirent row;
static int dir_case,dir_reads,dir_stats,dir_closes;
static DIR *scan_fdopendir(int fd) {
    assert(fd==0 && last_parent==0 && (last_flags&(O_DIRECTORY|O_CLOEXEC))==(O_DIRECTORY|O_CLOEXEC));
    if(dir_case==1) {errno=ENOMEM;return NULL;}
    for(int i=0;i<2;++i) {if(!streams[i].used) {streams[i].used=1;streams[i].index=0;return &streams[i];}}
    assert(0);return NULL;
}
static int scan_dirfd(DIR *p) {assert(p->used);return 0;}
static struct dirent *scan_readdir(DIR *p) {
    assert(p->used);++dir_reads;assert(errno==0);
    if(dir_case==10) {errno=EIO;return NULL;}
    const char *names[]={".","..","alpha","\xed\x95\x9c","link","folder","pipe"};
    if(p->index==7) {return NULL;}
    const char *name=names[p->index++];memset(&row,0,sizeof row);memcpy(row.d_name,name,strlen(name)+1);
    row.d_ino=999;row.d_reclen=(uint16_t)(offsetof(struct dirent,d_name)+strlen(name)+1);
    if(dir_case==5) {row.d_reclen=0;}
    if(dir_case==6) {row.d_reclen=(uint16_t)(offsetof(struct dirent,d_name)+2);memcpy(row.d_name,"abc",3);}
    if(dir_case==7) {row.d_reclen=(uint16_t)(offsetof(struct dirent,d_name)+257);memset(row.d_name,65,256);row.d_name[256]=0;}
    if(dir_case==8) {memcpy(row.d_name,"a/b",4);row.d_reclen=(uint16_t)(offsetof(struct dirent,d_name)+4);}
    if(dir_case==9) {row.d_name[0]=0;}
    return &row;
}
static int scan_fstatat(int fd,const char *name,struct stat *info,int flags) {
    assert(fd==0 && flags==AT_SYMLINK_NOFOLLOW);++dir_stats;
    if(dir_case==2 && dir_stats==1) {errno=EINTR;return -1;}
    if(dir_case==3) {errno=EACCES;return -1;}
    if(dir_case==4) {errno=ENOENT;return -1;}
    memset(info,0,sizeof *info);info->st_dev=71;info->st_ino=123;info->st_size=3;info->st_mtim.tv_sec=-2;info->st_mtim.tv_nsec=42;
    info->st_mode=strcmp(name,"link")==0 ? 40960 : strcmp(name,"folder")==0 ? 16384 : strcmp(name,"pipe")==0 ? 4096 : 32768;
    if(dir_case==11) {info->st_mtim.tv_nsec=1000000000;}
    if(dir_case==12) {info->st_size=-1;}
    return 0;
}
static int scan_closedir(DIR *p) {assert(p->used);p->used=0;++dir_closes;if(dir_case==13) {errno=EINTR;return -1;}return 0;}
#define fdopendir scan_fdopendir
#define readdir scan_readdir
#define dirfd scan_dirfd
#define fstatat scan_fstatat
#define closedir scan_closedir
`

const linuxDirectoryBoundaryProbe = `
static void scan_reset(int value) {dir_case=value;dir_reads=dir_stats=dir_closes=0;scenario=opened=closed=0;memset(streams,0,sizeof streams);}
int main(void) {
    size_t first=99,second=0;uint8_t name[256];uint32_t length=99,state=99;struct fy_linux_root_metadata_v1 info;
    scan_reset(0);assert(fairyc_private_linux_directory_scan_open_v1(0,&first)==EINVAL && first==0 && opened==0);
    assert(fairyc_private_linux_directory_scan_open_v1(1,&first)==0 && first!=0 && opened==1);
    assert(fairyc_private_linux_directory_scan_open_v1(1,&second)==0 && second!=first && opened==2);
    assert(fairyc_private_linux_directory_scan_next_v1(first,name,255,&length,&state,&info)==EINVAL && length==0 && state==0 && dir_reads==0);
    for(int i=0;i<7;++i) {
        assert(fairyc_private_linux_directory_scan_next_v1(first,name,256,&length,&state,&info)==0);
        assert(state==(i<2 ? 2u : 1u) && length>0 && name[length]==0);
        if(i>=2) {assert(info.device==71 && info.identity==123 && info.modified_seconds==-2 && info.modified_nanos==42);}
        if(i==4) {assert(info.mode==40960);}if(i==5) {assert(info.mode==16384);}if(i==6) {assert(info.mode==4096);}
    }
    errno=EACCES;assert(fairyc_private_linux_directory_scan_next_v1(first,name,256,&length,&state,&info)==0 && state==0 && length==0);
    assert(fairyc_private_linux_directory_scan_next_v1(second,name,256,&length,&state,&info)==0 && state==2 && length==1);
    assert(fairyc_private_linux_directory_scan_close_v1(first)==0 && fairyc_private_linux_directory_scan_close_v1(second)==0 && dir_closes==2);
    assert(fairyc_private_linux_directory_scan_close_v1(0)==0 && dir_closes==2);
    scan_reset(1);assert(fairyc_private_linux_directory_scan_open_v1(1,&first)==ENOMEM && first==0 && closed==1);
    for(int mode=2;mode<=13;++mode) {
        scan_reset(mode);assert(fairyc_private_linux_directory_scan_open_v1(1,&first)==0);
        uint32_t code=fairyc_private_linux_directory_scan_next_v1(first,name,256,&length,&state,&info);
        if((mode>=2 && mode<=4) || mode==11 || mode==12) {
            assert(code==0 && state==2);code=fairyc_private_linux_directory_scan_next_v1(first,name,256,&length,&state,&info);assert(code==0 && state==2);
            code=fairyc_private_linux_directory_scan_next_v1(first,name,256,&length,&state,&info);
        }
        if(mode==2) {assert(code==0 && state==1 && dir_stats==2);}
        else if(mode==3) {assert(code==EACCES && state==0 && length==0);}
        else if(mode==4) {assert(code==0 && state==2 && length==5 && info.identity==0);}
        else if(mode==7) {assert(code==ENAMETOOLONG && length==0 && state==0);}
        else if(mode!=13) {assert(code==EIO && length==0 && state==0);}
        assert(fairyc_private_linux_directory_scan_close_v1(first)==(uint32_t)(mode==13 ? EINTR : 0) && dir_closes==1);
    }
    return 0;
}
`

const linuxDirectoryNativeProbe = `
#include <assert.h>
#include <stdlib.h>
static size_t enumerate(size_t stream) {
    uint8_t name[256];uint32_t length,state;struct fy_linux_root_metadata_v1 info;size_t count=0;
    for(;;) {
        assert(fairyc_private_linux_directory_scan_next_v1(stream,name,256,&length,&state,&info)==0);
        if(state==0) {break;}if(state==2) {continue;}++count;
        if(strcmp((char *)name,"link")==0) {assert((info.mode&S_IFMT)==S_IFLNK);}
        if(strcmp((char *)name,"folder")==0) {assert((info.mode&S_IFMT)==S_IFDIR);}
        if(strcmp((char *)name,"pipe")==0) {assert((info.mode&S_IFMT)==S_IFIFO);}
    }return count;
}
int main(void) {
    char path[]="/tmp/fairyc-directory-XXXXXX";assert(mkdtemp(path));uint32_t root=0;
    assert(fairyc_private_linux_directory_open_v1((const uint8_t *)path,strlen(path)+1,&root)==0);
    int fd=fy_linux_root_fd(root);char leaf[32];
    for(int i=0;i<601;++i) {snprintf(leaf,sizeof leaf,"entry-%d",i);int file=openat(fd,leaf,O_CREAT|O_EXCL|O_WRONLY|O_CLOEXEC,0600);assert(file>=0 && close(file)==0);}
    assert(symlinkat("entry-0",fd,"link")==0 && mkdirat(fd,"folder",0700)==0 && mkfifoat(fd,"pipe",0600)==0);
    size_t first=0,second=0;assert(fairyc_private_linux_directory_scan_open_v1(root,&first)==0 && fairyc_private_linux_directory_scan_open_v1(root,&second)==0);
    assert((fcntl(dirfd((DIR *)(uintptr_t)first),F_GETFD)&FD_CLOEXEC)!=0);
    char moved[256];snprintf(moved,sizeof moved,"%s-moved",path);assert(rename(path,moved)==0);
    assert(enumerate(first)==604 && enumerate(second)==604);
    assert(fairyc_private_linux_directory_scan_close_v1(first)==0 && fairyc_private_linux_directory_scan_close_v1(second)==0);
    assert(fairyc_private_linux_directory_scan_open_v1(root,&first)==0 && enumerate(first)==604 && fairyc_private_linux_directory_scan_close_v1(first)==0);
    for(int i=0;i<601;++i) {snprintf(leaf,sizeof leaf,"entry-%d",i);assert(unlinkat(fd,leaf,0)==0);}
    assert(unlinkat(fd,"link",0)==0 && unlinkat(fd,"pipe",0)==0 && unlinkat(fd,"folder",AT_REMOVEDIR)==0);
    assert(fairyc_private_linux_root_file_close_v1(root)==0 && rmdir(moved)==0);return 0;
}
`
