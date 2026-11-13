package cstdlib

import (
	"runtime"
	"strings"
	"testing"
)

func TestLinuxDirectoryModel(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC POSIX model")
	}
	var body strings.Builder
	for _, line := range strings.Split(stage0LinuxDirectory, "\n") {
		if !strings.HasPrefix(line, "#include") && !strings.HasPrefix(line, "//go:build") {
			body.WriteString(line + "\n")
		}
	}
	runWindowsFilesystemModel(t, linuxDirectoryModelPrelude+stage0TextView+body.String()+linuxDirectoryModelMain)
}

const linuxDirectoryModelPrelude = `
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
typedef uint32_t mode_t;
struct stat { mode_t st_mode;int64_t st_size; };
struct dirent {char d_name[256];};
typedef struct {int descriptor,position;struct dirent entry;} DIR;
#define O_RDONLY 0
#define O_DIRECTORY 1
#define O_NOFOLLOW 2
#define O_CLOEXEC 4
#define AT_SYMLINK_NOFOLLOW 256
#define S_ISREG(m) ((m)==1)
#define S_ISDIR(m) ((m)==2)
#define S_ISLNK(m) ((m)==3)
static int directory_handles,directory_stat_failure,directory_stream_failure;
static int model_directory_open(const char *p,int flags){(void)p;if((flags&(O_NOFOLLOW|O_DIRECTORY|O_CLOEXEC))!=(O_NOFOLLOW|O_DIRECTORY|O_CLOEXEC)){errno=EINVAL;return -1;}directory_handles++;return 1;}
static int model_directory_openat(int parent,const char *name,int flags){(void)parent;if(!(flags&O_NOFOLLOW)){errno=EINVAL;return -1;}if(!strcmp(name,"linkdir")){errno=ELOOP;return -1;}if(strcmp(name,"root")&&strcmp(name,".")){errno=ENOENT;return -1;}directory_handles++;return 2;}
static int model_directory_mkdirat(int parent,const char *name,mode_t mode){(void)parent;(void)name;(void)mode;return 0;}
static int model_directory_close(int fd){(void)fd;directory_handles--;return 0;}
static DIR *model_directory_fdopendir(int fd){if(directory_stream_failure){errno=EIO;return NULL;}DIR *d=calloc(1,sizeof *d);if(d)d->descriptor=fd;return d;}
static struct dirent *model_directory_readdir(DIR *d){const char *names[]={".","..","file","link"};if(d->position==4)return NULL;const char *name=names[d->position++];memcpy(d->entry.d_name,name,strlen(name)+1);return &d->entry;}
static int model_directory_dirfd(DIR *d){return d->descriptor;}
static int model_directory_closedir(DIR *d){directory_handles--;free(d);return 0;}
static int model_directory_fstatat(int parent,const char *name,struct stat *out,int flags){(void)parent;if(flags!=AT_SYMLINK_NOFOLLOW){errno=EINVAL;return -1;}if(directory_stat_failure){errno=ENOENT;return -1;}out->st_mode=!strcmp(name,"file")?1u:3u;out->st_size=3;return 0;}
#define open model_directory_open
#define openat model_directory_openat
#define mkdirat model_directory_mkdirat
#define close model_directory_close
#define fdopendir model_directory_fdopendir
#define readdir model_directory_readdir
#define dirfd model_directory_dirfd
#define closedir model_directory_closedir
#define fstatat model_directory_fstatat
`

const linuxDirectoryModelMain = `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){struct fy_std_directory_v0 directory={NULL,0};uint8_t name[8];memset(name,85,sizeof name);CHECK(fy_std_windows_read_dir_open_v0((const uint8_t *)"root",4,&directory).status==0&&directory_handles==1);struct fy_std_directory_entry_result_v0 e=fy_std_windows_read_dir_next_v0(&directory,NULL,0);CHECK(e.status==6&&e.name_length==4);e=fy_std_windows_read_dir_next_v0(&directory,name,2);CHECK(e.status==6&&e.name_length==4&&name[0]==85);
directory_stat_failure=1;e=fy_std_windows_read_dir_next_v0(&directory,name,sizeof name);CHECK(e.status==1&&name[0]==85);directory_stat_failure=0;e=fy_std_windows_read_dir_next_v0(&directory,name,sizeof name);CHECK(e.status==0&&e.kind==1&&e.size==3&&e.name_length==4&&!memcmp(name,"file",4)&&name[4]==85);e=fy_std_windows_read_dir_next_v0(&directory,name,sizeof name);CHECK(e.status==0&&e.kind==3&&e.reparse==1&&e.size==0&&!memcmp(name,"link",4));CHECK(fy_std_windows_read_dir_next_v0(&directory,name,sizeof name).kind==0);CHECK(fy_std_windows_read_dir_next_v0(&directory,name,sizeof name).kind==0);CHECK(fy_std_windows_read_dir_close_v0(&directory).status==0&&directory_handles==0&&directory.state==NULL&&directory.open==0);CHECK(fy_std_windows_read_dir_drop_v0(&directory)==0);CHECK(fy_std_windows_read_dir_next_v0(&directory,name,sizeof name).status==3);
CHECK(fy_std_windows_read_dir_open_v0((const uint8_t *)"linkdir/root",12,&directory).status==1&&directory_handles==0);uint8_t invalid[]={255};CHECK(fy_std_windows_read_dir_open_v0(invalid,1,&directory).status==5&&directory_handles==0);uint8_t interior[]={114,0,111,116};CHECK(fy_std_windows_read_dir_open_v0(interior,4,&directory).status==2&&directory_handles==0);directory_stream_failure=1;CHECK(fy_std_windows_read_dir_open_v0((const uint8_t *)"root",4,&directory).status==1&&directory_handles==0);CHECK(fy_std_windows_read_dir_close_v0(&directory).status==3);CHECK(fy_std_windows_read_dir_drop_v0(NULL)==2);return 0;}
`
