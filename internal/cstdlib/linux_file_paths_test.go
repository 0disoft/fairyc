package cstdlib

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Execute the production operation bodies against an explicit POSIX syscall
// model. This is Windows C execution of a model, not Linux execution evidence.
func TestLinuxFilePathsModel(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC model")
	}
	var body strings.Builder
	for _, line := range strings.Split(stage0LinuxFilePaths, "\n") {
		if !strings.HasPrefix(line, "#include") && !strings.HasPrefix(line, "//go:build") {
			body.WriteString(line + "\n")
		}
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, linuxFilePathsModelPrelude+body.String()+linuxFilePathsModelMain)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			exe := filepath.Join(dir, "paths-model.exe")
			if err := os.WriteFile(exe, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			out, err := exec.CommandContext(ctx, exe).CombinedOutput()
			if err != nil || len(out) != 0 {
				t.Fatalf("POSIX operation model: %v %s", err, out)
			}
		})
	}
}

const linuxFilePathsModelPrelude = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <stdarg.h>
#include <errno.h>
typedef uint32_t mode_t;
typedef intptr_t ssize_t;
struct stat { mode_t st_mode; };
#define S_ISDIR(m) ((m)==2)
#define S_ISREG(m) ((m)==1)
#define O_RDONLY 0
#define O_WRONLY 1
#define O_CREAT 2
#define O_EXCL 4
#define O_CLOEXEC 8
#define O_NONBLOCK 16
#define AT_FDCWD -100
#define RENAME_NOREPLACE 1
#define SYS_renameat2 316
#define open model_open
#define close model_close
#define read model_read
#define write model_write
#define lstat model_lstat
#define fstat model_fstat
#define link model_link
#define syscall model_syscall
struct fy_std_file_result_v0 {uint32_t status;int32_t error_number;size_t count;};
static struct fy_std_file_result_v0 fy_std_file_result_v0(uint32_t s,int e,size_t n){struct fy_std_file_result_v0 r={s,e,n};return r;}
static int fy_std_file_path_storage_length_v0(const uint8_t *p,size_t *n){if(!p)return 0;for(size_t i=0;i<4096;i++){if(!p[i]){*n=i+1;return i!=0;}}return 0;}
static int destination_exists,closed,source_kind,interrupted,write_failure,zero_write;
static size_t read_position,model_written;
static uint8_t copied[16];
static void reset(void){destination_exists=0;closed=0;source_kind=1;interrupted=1;write_failure=0;zero_write=0;read_position=model_written=0;memset(copied,85,sizeof copied);}
static int model_lstat(const char *p,struct stat *s){if(!strcmp(p,"missing")){errno=ENOENT;return -1;}s->st_mode=(mode_t)source_kind;return 0;}
static int model_fstat(int fd,struct stat *s){(void)fd;s->st_mode=(mode_t)source_kind;return 0;}
static int model_open(const char *p,int flags,...){if(!strcmp(p,"source")){if(!(flags&O_NONBLOCK)){errno=EINVAL;return -1;}return 10;}if(!(flags&O_EXCL)||destination_exists){errno=EEXIST;return -1;}destination_exists=1;return 11;}
static int model_close(int fd){(void)fd;closed++;return 0;}
static ssize_t model_read(int fd,void *p,size_t n){(void)fd;(void)n;if(interrupted){interrupted=0;errno=EINTR;return -1;}if(read_position==3)return 0;memcpy(p,"abc"+read_position,3-read_position);ssize_t count=(ssize_t)(3-read_position);read_position=3;return count;}
static ssize_t model_write(int fd,const void *p,size_t n){(void)fd;(void)n;if(zero_write)return 0;if(write_failure && model_written==1){errno=EIO;return -1;}copied[model_written++]=*(const uint8_t *)p;return 1;}
static int model_link(const char *s,const char *d){(void)s;(void)d;if(destination_exists){errno=EEXIST;return -1;}destination_exists=1;return 0;}
long model_syscall(long number,...){va_list args;va_start(args,number);int first=va_arg(args,int);(void)va_arg(args,const uint8_t *);int second=va_arg(args,int);(void)va_arg(args,const uint8_t *);int flags=va_arg(args,int);va_end(args);if(number!=SYS_renameat2||first!=AT_FDCWD||second!=AT_FDCWD||flags!=RENAME_NOREPLACE){errno=EINVAL;return -1;}if(destination_exists){errno=EEXIST;return -1;}destination_exists=1;return 0;}
`

const linuxFilePathsModelMain = `
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){struct fy_std_file_result_v0 r;
reset();r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==0&&r.count==3&&model_written==3&&closed==2&&!memcmp(copied,"abc",3)&&copied[3]==85);
reset();destination_exists=1;r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==1&&r.error_number==EEXIST&&r.count==0&&model_written==0&&closed==1);
reset();write_failure=1;r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==1&&r.error_number==EIO&&r.count==1&&closed==2&&copied[0]=='a'&&copied[1]==85);
reset();zero_write=1;r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==4&&r.count==0&&closed==2);
reset();source_kind=2;r=fy_std_windows_file_copy_bytes_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==2&&closed==1&&!destination_exists);
reset();r=fy_std_windows_file_rename_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==0&&destination_exists);
reset();destination_exists=1;r=fy_std_windows_file_rename_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==1&&r.error_number==EEXIST);
reset();source_kind=2;r=fy_std_windows_file_rename_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==2&&!destination_exists);
reset();r=fy_std_windows_file_hard_link_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==0&&destination_exists);r=fy_std_windows_file_hard_link_v0((const uint8_t *)"source",(const uint8_t *)"destination");CHECK(r.status==1&&r.error_number==EEXIST);
CHECK(fy_std_windows_file_rename_v0(NULL,(const uint8_t *)"x").status==2);CHECK(fy_std_windows_file_copy_bytes_v0((const uint8_t *)"",(const uint8_t *)"x").status==2);return 0;}
`
