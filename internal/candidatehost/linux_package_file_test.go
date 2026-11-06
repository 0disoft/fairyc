package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestPrivateLinuxPackageFileSelection(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_package_file_info_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxPackageFile {
		t.Fatalf("package file info selection: %v", err)
	}
	source, err = LinuxSource("fairyc_private_linux_package_file_info_v1( fairyc_private_linux_root_file_erase_if_v1(")
	if err != nil || source != linuxRootedFile+"\n"+linuxPackageFile+"\n"+linuxRetention {
		t.Fatalf("combined package cleanup selection: %v", err)
	}
	if _, err := LinuxSource("fairyc_private_linux_package_file_unknown_v1("); err == nil {
		t.Fatal("unknown package file operation admitted")
	}
}

func TestLinuxPackageFileAdapterBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC POSIX substitutes; native Linux probe is separate")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := linuxRootedFile
	for _, header := range []string{"fcntl.h", "sys/stat.h", "sys/file.h", "unistd.h"} {
		source = strings.ReplaceAll(source, "#include <"+header+">\n", "")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxRootedPOSIXStub+source+linuxPackageInfoPOSIXStub+linuxPackageFile+linuxPackageFileBoundaryProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "package-file.exe")
		})
	}
}

func TestLinuxPackageFileAdapterNative(t *testing.T) {
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
				artifact, err := cdriver.Build(tc, profile, linuxRootedFile+linuxPackageFile+linuxRetention+linuxPackageFileNativeProbe, "typedef int fy_package_file_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxPathProbe(t, artifact.Bytes, "package-file")
			})
		}
	}
}

const linuxPackageInfoPOSIXStub = `
static uint64_t package_links=1;
static int package_fstat(int fd,struct stat *info){int result=root_fstat(fd,info);if(result==0){info->st_nlink=package_links;}return result;}
#undef fstat
#define fstat package_fstat
`

const linuxPackageFileBoundaryProbe = `
static void reset(int value){scenario=value;stats=0;package_links=1;}
int main(void){
    struct fy_linux_root_metadata_v1 info;
    reset(0);assert(fairyc_private_linux_package_file_info_v1(1,&info)==0 && stats==1);
    assert(info.device==71 && info.identity==123 && info.size==3 && info.modified_seconds==91 && info.modified_nanos==42 && info.mode==32768);
    for(uint64_t link_count=0;link_count<=2;++link_count){if(link_count==1){continue;}reset(0);package_links=link_count;
        assert(fairyc_private_linux_package_file_info_v1(1,&info)==EMLINK && stats==1 && info.identity==0 && info.size==0 && info.mode==0);
        /* Generic rooted files still admit aliases; package policy owns rejection. */
        assert(fairyc_private_linux_root_file_info_v1(1,&info)==0);
    }
    const int invalid[]={3,4,9,10};const uint32_t codes[]={EIO,EINVAL,EINVAL,EINVAL};
    for(size_t i=0;i<sizeof invalid/sizeof invalid[0];++i){reset(invalid[i]);memset(&info,255,sizeof info);
        assert(fairyc_private_linux_package_file_info_v1(1,&info)==codes[i] && stats==1 && info.identity==0 && info.modified_nanos==0);
    }
    reset(8);assert(fairyc_private_linux_package_file_info_v1(1,&info)==0 && stats==2);
    reset(0);assert(fairyc_private_linux_package_file_info_v1(0,&info)==EINVAL && stats==0 && info.identity==0);
    assert(fairyc_private_linux_package_file_info_v1(UINT32_MAX,&info)==EINVAL && stats==0);
    assert(fairyc_private_linux_package_file_info_v1(1,NULL)==EINVAL && stats==0);
    assert(fairyc_private_linux_package_file_info_v1(2,&info)==EINVAL && info.identity==0);
    return 0;
}
`

const linuxPackageFileNativeProbe = `
#include <assert.h>
#include <stdlib.h>
int main(void){
    char temp[]="/tmp/fairyc-package-file-XXXXXX";assert(mkdtemp(temp));uint32_t root=0,file=0;
    assert(fairyc_private_linux_directory_open_v1((const uint8_t *)temp,strlen(temp)+1,&root)==0);
    assert(fairyc_private_linux_root_file_create_v1(root,&file)==0);
    size_t copied=0;assert(fairyc_private_linux_root_file_write_v1(file,(const uint8_t *)"a\0\xff",3,&copied)==0 && copied==3);
    assert(fairyc_private_linux_root_file_sync_v1(file)==0);
    struct fy_linux_root_metadata_v1 info;
    assert(fairyc_private_linux_package_file_info_v1(file,&info)==EMLINK && info.identity==0);
    assert(fairyc_private_linux_root_file_publish_new_v1(file,root,(const uint8_t *)"file",5)==0);
    assert(fairyc_private_linux_package_file_info_v1(file,&info)==0 && info.size==3 && info.identity!=0);
    int fd=fy_linux_root_fd(root);assert(linkat(fd,"file",fd,"alias",0)==0);
    struct fy_linux_root_metadata_v1 failed;
    assert(fairyc_private_linux_package_file_info_v1(file,&failed)==EMLINK && failed.identity==0);
    uint32_t removed=99,closed=99;
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"file",5,&info,&removed,&closed)==0 && removed==0 && closed==0);
    assert(unlinkat(fd,"alias",0)==0);
    assert(fairyc_private_linux_package_file_info_v1(file,&failed)==0);
    assert(fairyc_private_linux_root_file_erase_if_v1(root,(const uint8_t *)"file",5,&info,&removed,&closed)==0 && removed==1 && closed==0);
    assert(fairyc_private_linux_root_file_close_v1(file)==0 && fairyc_private_linux_root_file_close_v1(root)==0);
    assert(rmdir(temp)==0);return 0;
}
`
