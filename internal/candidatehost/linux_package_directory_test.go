package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxPackageDirectorySelection(t *testing.T) {
	for _, name := range []string{"create", "edit", "publish", "erase"} {
		source, err := LinuxSource("fairyc_private_linux_package_directory_" + name + "_v1(")
		if err != nil || source != linuxRootedFile+"\n"+linuxPackageDirectory {
			t.Fatalf("%s: missing package-directory source: %v", name, err)
		}
	}
	source, err := LinuxSource("fairyc_private_linux_package_directory_create_v1( fairyc_private_linux_package_directory_erase_v1( fairyc_private_linux_directory_scan_open_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxDirectory+"\n"+linuxPackageDirectory {
		t.Fatalf("combined directory support: %v", err)
	}
	for _, header := range []string{
		"fairyc_private_linux_package_directory_unknown_v1(",
		"fairyc_private_linux_package_directory_create_v1( fairyc_private_windows_ticks_v1(",
	} {
		if _, err := LinuxSource(header); err == nil {
			t.Fatalf("invalid support admitted: %s", header)
		}
	}
}

func TestLinuxPackageDirectoryAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile + "\n" + linuxPackageDirectory
	for _, header := range []string{"fcntl.h", "sys/stat.h", "sys/file.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxPackageDirectoryPOSIXStub+source+linuxPackageDirectoryBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "package-directory.exe")
		})
	}
}

func TestLinuxPackageDirectoryAdapterNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires GNU/Linux amd64, GCC and Clang; maintainer batches execution")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+"\n"+linuxPackageDirectory+linuxPackageDirectoryNativeProbe, "typedef int fy_package_directory_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "package-directory")
			})
		}
	}
}

// Model syscall failures independently of the implementation. Parent and child
// have different identities and fd zero is deliberately a valid child handle.
const linuxPackageDirectoryPOSIXStub = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#ifndef ESTALE
#define ESTALE 116
#endif
#ifndef ENOTEMPTY
#define ENOTEMPTY 39
#endif
#ifndef EOPNOTSUPP
#define EOPNOTSUPP 95
#endif
#define AT_FDCWD -100
#define AT_SYMLINK_NOFOLLOW 256
#define AT_REMOVEDIR 512
#define AT_EMPTY_PATH 4096
#define AT_SYMLINK_FOLLOW 1024
#define O_RDONLY 0
#define O_RDWR 2
#define O_DIRECTORY 65536
#define O_NOFOLLOW 131072
#define O_CLOEXEC 524288
#define O_NONBLOCK 2048
#define O_NOCTTY 256
#define O_TMPFILE 4259840
#define LOCK_EX 2
#define LOCK_NB 4
#define S_ISDIR(x) (((x)&61440)==16384)
#define S_ISREG(x) (((x)&61440)==32768)
typedef intptr_t ssize_t;
struct stat {uint64_t st_dev,st_ino,st_nlink;int64_t st_size;uint32_t st_mode;struct {int64_t tv_sec,tv_nsec;} st_mtim;};
enum { NORMAL, PARENT_OPEN, COLLISION, MKDIR_INTR, CHILD_OPEN, CHILD_STAT,
    ROLLBACK, PARENT_CLOSE, HELD_STAT, NAMED_STAT, REPLACED, LINK,
    RENAME_EXISTS, RENAME_INTR, RENAME_UNSUPPORTED, ERASE_BUSY, ERASE_INTR,
    OBSERVE_INTR, CHILD_TYPE, PARENT_TYPE, PARENT_STAT, CHILD_CLOSE,
    DEVICE_CHANGED, NAME_GONE, COLLISION_CLOSE, OPEN_INTR };
static int scenario,opens,child_opens,closes,closed_parent,closed_child,mkdirs,unlinks,renames,held_stats,named_stats;
static const char *expected_leaf="stage";
static int pkg_openat(int parent,const char *name,int flags,int mode) {
    assert(mode==0600);assert(flags==(O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC));++opens;
    if(strcmp(name,".")==0) {assert(parent==10);if(scenario==PARENT_OPEN){errno=EACCES;return -1;}return 20;}
    assert(parent==20 && strcmp(name,expected_leaf)==0);++child_opens;
    if(scenario==OPEN_INTR && child_opens==1){errno=EINTR;return -1;}
    if(scenario==CHILD_OPEN || scenario==ROLLBACK || scenario==PARENT_CLOSE){errno=EIO;return -1;}
    return 0;
}
static int pkg_fstat(int fd,struct stat *info) {
    assert(fd==0 || fd==20);memset(info,0,sizeof *info);info->st_dev=71;info->st_ino=fd==0?123:321;info->st_mode=16384;
    if(fd==20){if(scenario==PARENT_STAT){errno=EIO;return -1;}if(scenario==PARENT_TYPE){info->st_mode=32768;}return 0;}
    ++held_stats;if(scenario==OBSERVE_INTR && held_stats==1){errno=EINTR;return -1;}
    if(scenario==CHILD_STAT || scenario==CHILD_CLOSE || scenario==HELD_STAT){errno=EIO;return -1;}
    if(scenario==CHILD_TYPE){info->st_mode=32768;}return 0;
}
static int pkg_close(int fd) {
    assert(fd==0 || fd==20);++closes;if(fd==0){++closed_child;}else{++closed_parent;}
    if((fd==20 && (scenario==PARENT_CLOSE || scenario==COLLISION_CLOSE)) || (fd==0 && scenario==CHILD_CLOSE)){errno=EINTR;return -1;}return 0;
}
static int pkg_mkdirat(int parent,const char *name,int mode) {
    assert(parent==20 && strcmp(name,expected_leaf)==0 && mode==0700);++mkdirs;
    if(scenario==COLLISION || scenario==COLLISION_CLOSE){errno=EEXIST;return -1;}
    if(scenario==MKDIR_INTR){errno=EINTR;return -1;}return 0;
}
static int pkg_fstatat(int parent,const char *name,struct stat *info,int flags) {
    assert(parent==20 && strcmp(name,"stage")==0 && flags==AT_SYMLINK_NOFOLLOW);++named_stats;
    if(scenario==OBSERVE_INTR && named_stats==1){errno=EINTR;return -1;}
    if(scenario==NAMED_STAT || scenario==NAME_GONE){errno=scenario==NAME_GONE?ENOENT:EIO;return -1;}
    memset(info,0,sizeof *info);info->st_dev=scenario==DEVICE_CHANGED?72:71;info->st_ino=scenario==REPLACED?124:123;
    info->st_mode=scenario==LINK?40960:16384;return 0;
}
static int pkg_unlinkat(int parent,const char *name,int flags) {
    assert(parent==20 && strcmp(name,"stage")==0 && flags==AT_REMOVEDIR);++unlinks;
    if(scenario==ROLLBACK){errno=EACCES;return -1;}
    if(scenario==ERASE_BUSY){errno=ENOTEMPTY;return -1;}
    if(scenario==ERASE_INTR){errno=EINTR;return -1;}return 0;
}
static int pkg_renameat2(int parent,const char *name,int destination,const char *leaf,unsigned int flags) {
    assert(parent==20 && destination==30 && strcmp(name,"stage")==0 && strcmp(leaf,"package")==0 && flags==1u);++renames;
    if(scenario==RENAME_EXISTS){errno=EEXIST;return -1;}
    if(scenario==RENAME_INTR){errno=EINTR;return -1;}
    if(scenario==RENAME_UNSUPPORTED){errno=EOPNOTSUPP;return -1;}return 0;
}
static ssize_t pkg_read(int fd,void *data,size_t size){(void)fd;(void)data;(void)size;assert(0);return -1;}
static ssize_t pkg_write(int fd,const void *data,size_t size){(void)fd;(void)data;(void)size;assert(0);return -1;}
static int pkg_fsync(int fd){(void)fd;assert(0);return -1;}
static int pkg_flock(int fd,int flags){(void)fd;(void)flags;assert(0);return -1;}
static int pkg_linkat(int old,const char *from,int parent,const char *leaf,int flags){(void)old;(void)from;(void)parent;(void)leaf;(void)flags;assert(0);return -1;}
#define openat pkg_openat
#define fstat pkg_fstat
#define close pkg_close
#define mkdirat pkg_mkdirat
#define fstatat pkg_fstatat
#define unlinkat pkg_unlinkat
#define renameat2 pkg_renameat2
#define read pkg_read
#define write pkg_write
#define fsync pkg_fsync
#define flock pkg_flock
#define linkat pkg_linkat
`

const linuxPackageDirectoryBoundaryProbe = `
static const uint8_t stage[]="stage",package[]="package";
static void reset(int value){scenario=value;opens=child_opens=closes=closed_parent=closed_child=mkdirs=unlinks=renames=held_stats=named_stats=0;}
static uint32_t create(uint32_t *child,uint32_t *parent,uint32_t *cleanup){return fairyc_private_linux_package_directory_create_v1(11,stage,sizeof stage,child,parent,cleanup);}
static uint32_t edit(uint32_t *child,uint32_t *parent,uint32_t *cleanup){return fairyc_private_linux_package_directory_edit_v1(11,stage,sizeof stage,child,parent,cleanup);}
static uint32_t publish(void){return fairyc_private_linux_package_directory_publish_v1(1,21,stage,sizeof stage,31,package,sizeof package);}
static uint32_t erase(void){return fairyc_private_linux_package_directory_erase_v1(1,21,stage,sizeof stage);}
int main(void){
    uint32_t child=99,parent=99,cleanup=99;
    reset(NORMAL);assert(create(&child,&parent,&cleanup)==0 && child==1 && parent==21 && cleanup==0 && opens==2 && mkdirs==1 && closes==0);
    assert(fairyc_private_linux_root_file_close_v1(child)==0 && fairyc_private_linux_root_file_close_v1(parent)==0 && closed_child==1 && closed_parent==1);
    reset(NORMAL);assert(edit(&child,&parent,&cleanup)==0 && child==1 && parent==21 && mkdirs==0 && unlinks==0);
    const char *invalid[]={"",".","..","a/b","x\0y"};size_t sizes[]={1,2,3,4,4};
    for(size_t i=0;i<5;++i){reset(NORMAL);assert(fairyc_private_linux_package_directory_create_v1(11,(const uint8_t *)invalid[i],sizes[i],&child,&parent,&cleanup)==EINVAL && child==0 && parent==0 && cleanup==0 && opens==0);}
    uint8_t longest[257];memset(longest,'a',sizeof longest);longest[256]=0;
    assert(fairyc_private_linux_package_directory_create_v1(11,longest,sizeof longest,&child,&parent,&cleanup)==EINVAL && opens==0);
    assert(fairyc_private_linux_package_directory_create_v1(0,stage,sizeof stage,&child,&parent,&cleanup)==EINVAL && opens==0);
    assert(fairyc_private_linux_package_directory_create_v1(UINT32_MAX,stage,sizeof stage,&child,&parent,&cleanup)==EINVAL && opens==0);
    assert(create(NULL,&parent,&cleanup)==EINVAL && create(&child,NULL,&cleanup)==EINVAL && create(&child,&parent,NULL)==EINVAL);
    assert(create(&child,&child,&cleanup)==EINVAL && create(&child,&parent,&parent)==EINVAL && create(&child,&parent,&child)==EINVAL && opens==0);
    longest[255]=0;expected_leaf=(const char *)longest;
    reset(NORMAL);assert(fairyc_private_linux_package_directory_create_v1(11,longest,256,&child,&parent,&cleanup)==0 && child==1 && parent==21 && mkdirs==1);
    assert(fairyc_private_linux_root_file_close_v1(child)==0 && fairyc_private_linux_root_file_close_v1(parent)==0);
    expected_leaf="stage";
    reset(PARENT_OPEN);assert(create(&child,&parent,&cleanup)==EACCES && child==0 && parent==0 && cleanup==0 && opens==1 && closes==0 && mkdirs==0);
    reset(COLLISION);assert(create(&child,&parent,&cleanup)==EEXIST && child==0 && parent==0 && child_opens==0 && mkdirs==1 && closed_parent==1 && unlinks==0);
    reset(MKDIR_INTR);assert(create(&child,&parent,&cleanup)==EINTR && mkdirs==1 && closed_parent==1 && child_opens==0 && unlinks==0);
    reset(CHILD_OPEN);assert(create(&child,&parent,&cleanup)==EIO && child==0 && parent==0 && cleanup==0 && closed_parent==1 && unlinks==1);
    reset(CHILD_OPEN);assert(edit(&child,&parent,&cleanup)==EIO && mkdirs==0 && unlinks==0 && closed_parent==1);
    reset(CHILD_STAT);assert(create(&child,&parent,&cleanup)==EIO && child==0 && parent==0 && closed_child==1 && closed_parent==1 && unlinks==1);
    reset(ROLLBACK);assert(create(&child,&parent,&cleanup)==EIO && cleanup==EACCES && unlinks==1 && closed_parent==1);
    reset(PARENT_CLOSE);assert(create(&child,&parent,&cleanup)==EIO && cleanup==EINTR && closed_parent==1 && unlinks==1);
    reset(CHILD_CLOSE);assert(create(&child,&parent,&cleanup)==EIO && cleanup==EINTR && closed_child==1 && closed_parent==1 && unlinks==1);
    reset(CHILD_TYPE);assert(create(&child,&parent,&cleanup)==EINVAL && closed_child==1 && closed_parent==1 && unlinks==1);
    reset(PARENT_TYPE);assert(create(&child,&parent,&cleanup)==EINVAL && closed_parent==1 && mkdirs==0);
    reset(PARENT_STAT);assert(create(&child,&parent,&cleanup)==EIO && closed_parent==1 && mkdirs==0);
    reset(COLLISION_CLOSE);assert(create(&child,&parent,&cleanup)==EEXIST && cleanup==EINTR && closed_parent==1 && unlinks==0);
    reset(OPEN_INTR);assert(create(&child,&parent,&cleanup)==0 && opens==3 && mkdirs==1 && cleanup==0);
    reset(NORMAL);assert(publish()==0 && renames==1 && closes==0 && unlinks==0);
    reset(RENAME_EXISTS);assert(publish()==EEXIST && renames==1 && unlinks==0);
    reset(RENAME_INTR);assert(publish()==EINTR && renames==1 && unlinks==0);
    reset(RENAME_UNSUPPORTED);assert(publish()==EOPNOTSUPP && renames==1 && unlinks==0);
    const int unsafe[]={HELD_STAT,NAMED_STAT,REPLACED,LINK,DEVICE_CHANGED,NAME_GONE,CHILD_TYPE};
    const uint32_t errors[]={EIO,EIO,ESTALE,ESTALE,ESTALE,ENOENT,EINVAL};
    for(size_t i=0;i<sizeof unsafe/sizeof unsafe[0];++i){reset(unsafe[i]);assert(publish()==errors[i] && renames==0 && unlinks==0);reset(unsafe[i]);assert(erase()==errors[i] && renames==0 && unlinks==0);}
    reset(OBSERVE_INTR);assert(publish()==0 && held_stats==2 && named_stats==2 && renames==1);
    reset(NORMAL);assert(erase()==0 && unlinks==1 && closes==0);
    reset(ERASE_BUSY);assert(erase()==ENOTEMPTY && unlinks==1);
    reset(ERASE_INTR);assert(erase()==EINTR && unlinks==1);
    reset(NORMAL);assert(fairyc_private_linux_package_directory_publish_v1(1,21,stage,sizeof stage,0,package,sizeof package)==EINVAL && held_stats==0);
    assert(fairyc_private_linux_package_directory_publish_v1(1,21,stage,sizeof stage,31,(const uint8_t *)"../x",5)==EINVAL && held_stats==0);
    assert(fairyc_private_linux_package_directory_publish_v1(0,21,stage,sizeof stage,31,package,sizeof package)==EINVAL && held_stats==0);
    assert(fairyc_private_linux_package_directory_publish_v1(1,0,stage,sizeof stage,31,package,sizeof package)==EINVAL && held_stats==0);
    assert(fairyc_private_linux_package_directory_erase_v1(1,21,(const uint8_t *)"a/b",4)==EINVAL && unlinks==0);
    return 0;
}
`

const linuxPackageDirectoryNativeProbe = `
#include <assert.h>
#include <stdlib.h>
int main(void){
    (void)umask(0077);
    char temp[]="/tmp/fairyc-package-directory-XXXXXX";assert(mkdtemp(temp)!=NULL);
    uint32_t root=0,stage=0,parent=0,cleanup=99;
    const uint8_t from[]="stage",to[]="package";
    assert(fairyc_private_linux_directory_open_v1((const uint8_t *)temp,strlen(temp)+1,&root)==0);
    int directory=fy_linux_root_fd(root);
    assert(fairyc_private_linux_package_directory_create_v1(root,from,sizeof from,&stage,&parent,&cleanup)==0 && cleanup==0);
    assert(stage!=0 && parent!=0 && parent!=root);
    assert((fcntl(fy_linux_root_fd(stage),F_GETFD)&FD_CLOEXEC)!=0 && (fcntl(fy_linux_root_fd(parent),F_GETFD)&FD_CLOEXEC)!=0);
    struct stat identity;assert(fstat(fy_linux_root_fd(stage),&identity)==0 && (identity.st_mode&0777)==0700);
    uint32_t absent=99,absent_parent=99;
    assert(fairyc_private_linux_package_directory_edit_v1(root,(const uint8_t *)"absent",7,&absent,&absent_parent,&cleanup)==ENOENT && absent==0 && absent_parent==0 && cleanup==0);
    assert(fstatat(directory,"absent",&identity,AT_SYMLINK_NOFOLLOW)<0 && errno==ENOENT);
    assert(fairyc_private_linux_package_directory_create_v1(root,from,sizeof from,&absent,&absent_parent,&cleanup)==EEXIST && absent==0 && absent_parent==0);
    int file=openat(fy_linux_root_fd(stage),"data",O_CREAT|O_EXCL|O_WRONLY|O_CLOEXEC,0600);assert(file>=0);
    assert(write(file,"a\0\xff",3)==3 && close(file)==0);
    assert(fairyc_private_linux_package_directory_erase_v1(stage,parent,from,sizeof from)==ENOTEMPTY);
    /* Even an existing empty destination must not be replaced. */
    assert(mkdirat(directory,"package",0700)==0);
    assert(fairyc_private_linux_package_directory_publish_v1(stage,parent,from,sizeof from,root,to,sizeof to)==EEXIST);
    assert(unlinkat(directory,"package",AT_REMOVEDIR)==0);
    char moved[256];int length=snprintf(moved,sizeof moved,"%s-moved",temp);assert(length>0 && (size_t)length<sizeof moved);
    assert(rename(temp,moved)==0);
    /* The original root can close; the retained parent survives its rename. */
    assert(fairyc_private_linux_root_file_close_v1(root)==0);root=0;
    assert(fairyc_private_linux_package_directory_publish_v1(stage,parent,from,sizeof from,parent,to,sizeof to)==0);
    assert(fstatat(fy_linux_root_fd(parent),"stage",&identity,AT_SYMLINK_NOFOLLOW)<0 && errno==ENOENT);
    assert(fairyc_private_linux_package_directory_erase_v1(stage,parent,from,sizeof from)==ENOENT);
    uint32_t edited=0,edited_parent=0;
    assert(fairyc_private_linux_package_directory_edit_v1(parent,to,sizeof to,&edited,&edited_parent,&cleanup)==0 && cleanup==0);
    file=openat(fy_linux_root_fd(edited),"data",O_RDONLY|O_CLOEXEC);assert(file>=0);uint8_t data[4]={0};
    assert(read(file,data,sizeof data)==3 && memcmp(data,"a\0\xff",3)==0 && close(file)==0);
    assert(unlinkat(fy_linux_root_fd(edited),"data",0)==0);
    /* Replacement by a different empty directory cannot acquire cleanup rights. */
    assert(renameat(fy_linux_root_fd(parent),"package",fy_linux_root_fd(parent),"saved")==0);
    assert(mkdirat(fy_linux_root_fd(parent),"package",0700)==0);
    assert(fairyc_private_linux_package_directory_erase_v1(edited,edited_parent,to,sizeof to)==ESTALE);
    assert(fairyc_private_linux_package_directory_publish_v1(edited,edited_parent,to,sizeof to,parent,from,sizeof from)==ESTALE);
    assert(unlinkat(fy_linux_root_fd(parent),"package",AT_REMOVEDIR)==0);
    assert(symlinkat("saved",fy_linux_root_fd(parent),"package")==0);
    assert(fairyc_private_linux_package_directory_erase_v1(edited,edited_parent,to,sizeof to)==ESTALE);
    assert(fairyc_private_linux_package_directory_edit_v1(parent,to,sizeof to,&absent,&absent_parent,&cleanup)!=0 && absent==0 && absent_parent==0);
    assert(fairyc_private_linux_package_directory_create_v1(parent,to,sizeof to,&absent,&absent_parent,&cleanup)==EEXIST);
    assert(unlinkat(fy_linux_root_fd(parent),"package",0)==0);
    assert(renameat(fy_linux_root_fd(parent),"saved",fy_linux_root_fd(parent),"package")==0);
    assert(fairyc_private_linux_package_directory_erase_v1(edited,edited_parent,to,sizeof to)==0);
    assert(fairyc_private_linux_root_file_close_v1(edited)==0 && fairyc_private_linux_root_file_close_v1(edited_parent)==0);
    assert(fairyc_private_linux_root_file_close_v1(stage)==0 && fairyc_private_linux_root_file_close_v1(parent)==0);
    assert(rmdir(moved)==0);return 0;
}
`
