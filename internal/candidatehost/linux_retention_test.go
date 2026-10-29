package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxRetentionSelection(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_root_file_erase_if_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxRetention {
		t.Fatalf("erase selection: %v", err)
	}
	source, err = LinuxSource("fairyc_private_linux_root_file_erase_if_v1( fairyc_private_linux_directory_scan_open_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxDirectory+"\n"+linuxRetention {
		t.Fatalf("scan/erase dependency order: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_root_file_erase_unknown_v1("); err == nil {
		t.Fatal("unknown deletion admitted")
	}
}

func TestLinuxRetentionAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile + "\n" + linuxRetention
	for _, header := range []string{"fcntl.h", "sys/stat.h", "sys/file.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxRootedPOSIXStub+linuxRetentionPOSIXStub+source+linuxRetentionBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "retention.exe")
		})
	}
}

func TestLinuxRetentionAdapterNative(t *testing.T) {
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
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+"\n"+linuxRetention+linuxRetentionNativeProbe, "typedef int fy_retention_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "retention")
			})
		}
	}
}

const linuxRetentionPOSIXStub = `
#define AT_SYMLINK_NOFOLLOW 256
static int erase_case,erase_stats,erases;
static int erase_fstatat(int fd,const char *leaf,struct stat *info,int flags) {
    assert(fd==1 && strcmp(leaf,"new")==0 && flags==AT_SYMLINK_NOFOLLOW);++erase_stats;
    if(erase_case==1) {errno=ENOENT;return -1;}
    if(erase_case==2) {errno=EACCES;return -1;}
    if(erase_case==3 && erase_stats==1) {errno=EINTR;return -1;}
    memset(info,0,sizeof *info);info->st_dev=71;info->st_ino=123;info->st_size=3;info->st_mode=32768;info->st_nlink=1;info->st_mtim.tv_sec=91;info->st_mtim.tv_nsec=42;
    if(erase_case==4) {info->st_dev++;}if(erase_case==5) {info->st_ino++;}if(erase_case==6) {info->st_size++;}
    if(erase_case==7) {info->st_mtim.tv_sec++;}if(erase_case==8) {info->st_mtim.tv_nsec++;}if(erase_case==9) {info->st_mode++;}
    if(erase_case==10) {info->st_mode=40960;}if(erase_case==11) {info->st_nlink=2;}
    return 0;
}
static int erase_unlinkat(int fd,const char *leaf,int flags) {
    assert(fd==1 && strcmp(leaf,"new")==0 && flags==0);++erases;
    if(erase_case==12) {errno=ENOENT;return -1;}if(erase_case==13) {errno=EINTR;return -1;}if(erase_case==14) {errno=EACCES;return -1;}return 0;
}
#define fstatat erase_fstatat
#define unlinkat erase_unlinkat
`

const linuxRetentionBoundaryProbe = `
int main(void) {
    struct fy_linux_root_metadata_v1 expected={71,123,3,91,42,32768};uint32_t removed,cleanup;
    for(int i=0;i<=14;++i) {
        scenario=0;erase_case=i;erase_stats=erases=opened=closed=0;
        uint32_t code=fairyc_private_linux_root_file_erase_if_v1(1,(const uint8_t *)"new",4,&expected,&removed,&cleanup);
        if(i==2 || i==14) {assert(code==EACCES && removed==0);}
        else if(i==13) {assert(code==EINTR && removed==0 && erases==1);}
        else {assert(code==0 && removed==(i==0 || i==3 ? 1u : 0u));}
        assert(cleanup==0 && closed==1 && opened==1);
        if(i>=4 && i<=11) {assert(erases==0);}if(i==3) {assert(erase_stats==2);}
    }
    scenario=30;erases=erase_stats=opened=closed=0;
    assert(fairyc_private_linux_root_file_erase_if_v1(1,(const uint8_t *)"new",4,&expected,&removed,&cleanup)==EWOULDBLOCK && removed==0 && closed==1 && erase_stats==0);
    links=0;assert(fairyc_private_linux_root_file_publish_new_v1(1,1,(const uint8_t *)"new",4)==EWOULDBLOCK && links==0);
    scenario=5;erase_case=0;closed=erases=0;
    assert(fairyc_private_linux_root_file_erase_if_v1(1,(const uint8_t *)"new",4,&expected,&removed,&cleanup)==0 && removed==1 && cleanup==EINTR && closed==1 && erases==1);
    links=closed=0;assert(fairyc_private_linux_root_file_publish_new_v1(1,1,(const uint8_t *)"new",4)==0 && links==1 && closed==1);
    scenario=0;opened=0;expected.identity=0;
    assert(fairyc_private_linux_root_file_erase_if_v1(1,(const uint8_t *)"new",4,&expected,&removed,&cleanup)==EINVAL && opened==0);
    expected.identity=123;assert(fairyc_private_linux_root_file_erase_if_v1(1,(const uint8_t *)"../new",7,&expected,&removed,&cleanup)==EINVAL && opened==0);
    return 0;
}
`

const linuxRetentionNativeProbe = `
#include <assert.h>
#include <stdlib.h>
int main(void) {
    char path[]="/tmp/fairyc-retention-XXXXXX";assert(mkdtemp(path));uint32_t root=0,file=0,held=0;
    assert(fairyc_private_linux_directory_open_v1((const uint8_t *)path,strlen(path)+1,&root)==0);
    int fd=fy_linux_root_fd(root),out=openat(fd,"new",O_CREAT|O_EXCL|O_WRONLY|O_CLOEXEC,0600);assert(out>=0 && write(out,"abc",3)==3 && close(out)==0);
    assert(fairyc_private_linux_root_file_open_v1(root,(const uint8_t *)"new",4,&file)==0);
    struct fy_linux_root_metadata_v1 info;assert(fairyc_private_linux_root_file_info_v1(file,&info)==0);
    assert(fairyc_private_linux_root_file_close_v1(file)==0);uint32_t removed=0,cleanup=0;
    assert(fy_linux_root_lock(root,&held)==0);
    uint32_t code=fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&info,&removed,&cleanup);
    assert((code==EWOULDBLOCK || code==EAGAIN) && removed==0);
    assert(close(fy_linux_root_fd(held))==0);
    struct fy_linux_root_metadata_v1 stale=info;stale.identity++;
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&stale,&removed,&cleanup)==0 && removed==0);
    assert(linkat(fd,"new",fd,"alias",0)==0);
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&info,&removed,&cleanup)==0 && removed==0);
    assert(unlinkat(fd,"alias",0)==0);
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&info,&removed,&cleanup)==0 && removed==1 && cleanup==0);
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&info,&removed,&cleanup)==0 && removed==0);
    assert(symlinkat("missing",fd,"new")==0);
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"new",4,&info,&removed,&cleanup)==0 && removed==0);
    assert(unlinkat(fd,"new",0)==0 && fairyc_private_linux_root_file_close_v1(root)==0 && rmdir(path)==0);return 0;
}
`
