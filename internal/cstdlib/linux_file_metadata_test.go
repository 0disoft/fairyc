package cstdlib

import (
	"runtime"
	"testing"
)

func TestLinuxFileMetadataModel(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC POSIX model; actual Linux is separate")
	}
	runWindowsFilesystemModel(t, linuxFileMetadataModelPrelude+stage0LinuxFileMetadata+linuxFileMetadataModelMain)
}

const linuxFileMetadataModelPrelude = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <errno.h>
typedef uint32_t mode_t;typedef int64_t off_t;
struct stat {mode_t st_mode;uint64_t st_nlink,st_dev,st_ino;int64_t st_blocks;};
struct fy_std_linux_file_v0 {int descriptor;uint8_t open;};
struct fy_std_file_result_v0 {uint32_t status;int32_t error_number;size_t count;};
struct fy_std_file_size_result_v0 {uint32_t status,os_code;size_t size;};
static struct fy_std_file_result_v0 fy_std_file_result_v0(uint32_t s,int e,size_t n){struct fy_std_file_result_v0 r={s,e,n};return r;}
static struct fy_std_file_size_result_v0 fy_std_file_size_result_v0(uint32_t s,int e,size_t n){struct fy_std_file_size_result_v0 r={s,(uint32_t)e,n};return r;}
static int fy_std_file_path_storage_length_v0(const uint8_t *p,size_t *n){if(!p)return 0;for(size_t i=0;i<4096;i++){if(!p[i]){*n=i+1;return i!=0;}}return 0;}
static int fy_std_file_storage_to_linux_v0(const void *p,struct fy_std_linux_file_v0 *f){if(!p)return 0;*f=*(const struct fy_std_linux_file_v0 *)p;return (!f->open&&f->descriptor==-1)||(f->open==1&&f->descriptor>=0);}
#define S_ISREG(m) (((m)&0170000)==0100000)
#define S_ISDIR(m) (((m)&0170000)==0040000)
#define S_ISBLK(m) (((m)&0170000)==0060000)
#define S_ISCHR(m) (((m)&0170000)==0020000)
#define S_ISFIFO(m) (((m)&0170000)==0010000)
#define S_ISSOCK(m) (((m)&0170000)==0140000)
#define S_IWUSR 0200
#define S_IWGRP 0020
#define S_IWOTH 0002
static struct stat metadata;
static int metadata_failure,metadata_interrupted;
static int64_t metadata_length;
static int model_metadata_fstat(int fd,struct stat *out){(void)fd;if(metadata_interrupted){metadata_interrupted=0;errno=EINTR;return -1;}if(metadata_failure){errno=EIO;return -1;}*out=metadata;return 0;}
static int model_metadata_lstat(const char *path,struct stat *out){(void)path;return model_metadata_fstat(1,out);}
static int model_metadata_fchmod(int fd,mode_t permissions){(void)fd;metadata.st_mode=(metadata.st_mode&0170000)|permissions;return 0;}
static int model_metadata_ftruncate(int fd,off_t length){(void)fd;metadata_length=length;return 0;}
#define fstat model_metadata_fstat
#define lstat model_metadata_lstat
#define fchmod model_metadata_fchmod
#define ftruncate model_metadata_ftruncate
`

const linuxFileMetadataModelMain = `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){struct fy_std_linux_file_v0 live={7,1},closed={-1,0},invalid={9,0};metadata.st_mode=0100644;metadata.st_nlink=2;metadata.st_blocks=3;metadata.st_dev=UINT64_C(0x12345678abcdef01);metadata.st_ino=UINT64_C(0xfedcba9876543210);metadata_interrupted=1;
CHECK(fy_std_windows_file_kind_v0(&live).kind==1);CHECK(fy_std_windows_file_kind_v0(&closed).status==3);CHECK(fy_std_windows_file_kind_v0(&invalid).status==2);CHECK(fy_std_windows_file_kind_v0(NULL).status==2);
CHECK(fy_std_windows_file_read_only_v0(&live).enabled==0);CHECK(fy_std_windows_file_set_readonly_v0(&live,1).status==0);CHECK((metadata.st_mode&07777)==0444);CHECK(fy_std_windows_file_read_only_v0(&live).enabled==1);CHECK(fy_std_windows_file_set_readonly_v0(&live,0).status==0);CHECK((metadata.st_mode&07777)==0644);CHECK(fy_std_windows_file_set_readonly_v0(&live,2).status==2);
CHECK(fy_std_windows_file_link_count_v0(&live).size==2);CHECK(fy_std_windows_file_alloc_size_v0(&live).size==1536);metadata.st_blocks=-1;CHECK(fy_std_windows_file_alloc_size_v0(&live).status==1);metadata.st_blocks=INT64_MAX;CHECK(fy_std_windows_file_alloc_size_v0(&live).status==1);
struct fy_std_file_id_v0 id;memset(&id,85,sizeof id);CHECK(fy_std_windows_file_id_v0(&live,&id).status==0);CHECK(id.volume_low==0xabcdef01u&&id.volume_high==0x12345678u&&id.file_low_0==0x76543210u&&id.file_low_1==0xfedcba98u&&id.file_high_0==0&&id.file_high_1==0);CHECK(fy_std_windows_file_id_v0(&live,NULL).status==2);memset(&id,85,sizeof id);metadata_failure=1;CHECK(fy_std_windows_file_id_v0(&live,&id).status==1&&id.volume_low==0x55555555u);metadata_failure=0;
CHECK(fy_std_windows_file_set_len_v0(&live,123).size==123&&metadata_length==123);CHECK(fy_std_windows_file_set_len_v0(&live,UINT64_MAX).status==2&&metadata_length==123);CHECK(fy_std_windows_file_set_len_v0(&closed,1).status==3);
metadata.st_mode=0040755;CHECK(fy_std_windows_file_path_kind_v0((const uint8_t *)"path").kind==2);metadata.st_mode=0120777;CHECK(fy_std_windows_file_path_kind_v0((const uint8_t *)"path").kind==3);CHECK(fy_std_windows_file_path_kind_v0((const uint8_t *)"").status==2);metadata.st_mode=0020600;CHECK(fy_std_windows_file_kind_v0(&live).kind==2);metadata.st_mode=0010600;CHECK(fy_std_windows_file_kind_v0(&live).kind==3);return 0;}
`
